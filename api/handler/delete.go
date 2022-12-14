package handler

import (
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	apistatus "github.com/TrueCloudLab/frostfs-sdk-go/client/status"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DeleteObjectsRequest -- xml carrying the object key names which should be deleted.
type DeleteObjectsRequest struct {
	// Element to enable quiet mode for the request
	Quiet bool
	// List of objects to be deleted
	Objects []ObjectIdentifier `xml:"Object"`
}

// ObjectIdentifier carries the key name for the object to delete.
type ObjectIdentifier struct {
	ObjectName string `xml:"Key"`
	VersionID  string `xml:"VersionId,omitempty"`
}

// DeletedObject carries the key name for the object to delete.
type DeletedObject struct {
	ObjectIdentifier
	DeleteMarker          bool   `xml:"DeleteMarker,omitempty"`
	DeleteMarkerVersionID string `xml:"DeleteMarkerVersionId,omitempty"`
}

// DeleteError structure.
type DeleteError struct {
	Code      string
	Message   string
	Key       string
	VersionID string `xml:"versionId,omitempty"`
}

// DeleteObjectsResponse container for multiple object deletes.
type DeleteObjectsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ DeleteResult" json:"-"`

	// Collection of all deleted objects
	DeletedObjects []DeletedObject `xml:"Deleted,omitempty"`

	// Collection of errors deleting certain objects.
	Errors []DeleteError `xml:"Error,omitempty"`
}

func (h *handler) DeleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	versionID := reqInfo.URL.Query().Get(api.QueryVersionID)
	versionedObject := []*layer.VersionedObject{{
		Name:      reqInfo.ObjectName,
		VersionID: versionID,
	}}

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	bktSettings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	p := &layer.DeleteObjectParams{
		BktInfo:  bktInfo,
		Objects:  versionedObject,
		Settings: bktSettings,
	}
	deletedObjects := h.obj.DeleteObjects(r.Context(), p)
	deletedObject := deletedObjects[0]
	if deletedObject.Error != nil {
		if isErrObjectLocked(deletedObject.Error) {
			h.logAndSendError(w, "object is locked", reqInfo, errors.GetAPIError(errors.ErrAccessDenied))
		} else {
			h.logAndSendError(w, "could not delete object", reqInfo, deletedObject.Error)
		}
		return
	}

	var m *SendNotificationParams

	if bktSettings.VersioningEnabled() && len(versionID) == 0 {
		m = &SendNotificationParams{
			Event: EventObjectRemovedDeleteMarkerCreated,
			NotificationInfo: &data.NotificationInfo{
				Name:    reqInfo.ObjectName,
				HashSum: deletedObject.DeleteMarkerEtag,
			},
			BktInfo: bktInfo,
			ReqInfo: reqInfo,
		}
	} else {
		var objID oid.ID
		if len(versionID) != 0 {
			if err = objID.DecodeString(versionID); err != nil {
				h.log.Error("couldn't send notification: %w", zap.Error(err))
			}
		}

		m = &SendNotificationParams{
			Event: EventObjectRemovedDelete,
			NotificationInfo: &data.NotificationInfo{
				Name:    reqInfo.ObjectName,
				Version: objID.EncodeToString(),
			},
			BktInfo: bktInfo,
			ReqInfo: reqInfo,
		}
	}

	if err = h.sendNotifications(r.Context(), m); err != nil {
		h.log.Error("couldn't send notification: %w", zap.Error(err))
	}

	if deletedObject.VersionID != "" {
		w.Header().Set(api.AmzVersionID, deletedObject.VersionID)
	}
	if deletedObject.DeleteMarkVersion != "" {
		w.Header().Set(api.AmzDeleteMarker, strconv.FormatBool(true))
		if deletedObject.VersionID == "" {
			w.Header().Set(api.AmzVersionID, deletedObject.DeleteMarkVersion)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func isErrObjectLocked(err error) bool {
	switch err.(type) {
	default:
		return strings.Contains(err.Error(), "object is locked")
	case apistatus.ObjectLocked,
		*apistatus.ObjectLocked:
		return true
	}
}

// DeleteMultipleObjectsHandler handles multiple delete requests.
func (h *handler) DeleteMultipleObjectsHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	// Content-Md5 is required and should be set
	// http://docs.aws.amazon.com/AmazonS3/latest/API/multiobjectdeleteapi.html
	if _, ok := r.Header[api.ContentMD5]; !ok {
		h.logAndSendError(w, "missing Content-MD5", reqInfo, errors.GetAPIError(errors.ErrMissingContentMD5))
		return
	}

	// Content-Length is required and should be non-zero
	// http://docs.aws.amazon.com/AmazonS3/latest/API/multiobjectdeleteapi.html
	if r.ContentLength <= 0 {
		h.logAndSendError(w, "missing Content-Length", reqInfo, errors.GetAPIError(errors.ErrMissingContentLength))
		return
	}

	// Unmarshal list of keys to be deleted.
	requested := &DeleteObjectsRequest{}
	if err := xml.NewDecoder(r.Body).Decode(requested); err != nil {
		h.logAndSendError(w, "couldn't decode body", reqInfo, errors.GetAPIError(errors.ErrMalformedXML))
		return
	}

	removed := make(map[string]*layer.VersionedObject)
	toRemove := make([]*layer.VersionedObject, 0, len(requested.Objects))
	for _, obj := range requested.Objects {
		versionedObj := &layer.VersionedObject{
			Name:      obj.ObjectName,
			VersionID: obj.VersionID,
		}
		toRemove = append(toRemove, versionedObj)
		removed[versionedObj.String()] = versionedObj
	}

	response := &DeleteObjectsResponse{
		Errors:         make([]DeleteError, 0, len(toRemove)),
		DeletedObjects: make([]DeletedObject, 0, len(toRemove)),
	}

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	bktSettings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	marshaler := zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, obj := range toRemove {
			encoder.AppendString(obj.String())
		}
		return nil
	})

	p := &layer.DeleteObjectParams{
		BktInfo:  bktInfo,
		Objects:  toRemove,
		Settings: bktSettings,
	}
	deletedObjects := h.obj.DeleteObjects(r.Context(), p)

	var errs []error
	for _, obj := range deletedObjects {
		if obj.Error != nil {
			code := "BadRequest"
			if s3err, ok := obj.Error.(errors.Error); ok {
				code = s3err.Code
			}
			response.Errors = append(response.Errors, DeleteError{
				Code:      code,
				Message:   obj.Error.Error(),
				Key:       obj.Name,
				VersionID: obj.VersionID,
			})
			errs = append(errs, obj.Error)
		} else if !requested.Quiet {
			deletedObj := DeletedObject{
				ObjectIdentifier: ObjectIdentifier{
					ObjectName: obj.Name,
					VersionID:  obj.VersionID,
				},
				DeleteMarkerVersionID: obj.DeleteMarkVersion,
			}
			if deletedObj.DeleteMarkerVersionID != "" {
				deletedObj.DeleteMarker = true
			}
			response.DeletedObjects = append(response.DeletedObjects, deletedObj)
		}
	}
	if len(errs) != 0 {
		fields := []zap.Field{
			zap.Array("objects", marshaler),
			zap.Errors("errors", errs),
		}
		h.log.Error("couldn't delete objects", fields...)
	}

	if err = api.EncodeToResponse(w, response); err != nil {
		h.logAndSendError(w, "could not write response", reqInfo, err, zap.Array("objects", marshaler))
		return
	}
}

func (h *handler) DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	var sessionToken *session.Container

	boxData, err := layer.GetBoxData(r.Context())
	if err == nil {
		sessionToken = boxData.Gate.SessionTokenForDelete()
	}

	if err = h.obj.DeleteBucket(r.Context(), &layer.DeleteBucketParams{
		BktInfo:      bktInfo,
		SessionToken: sessionToken,
	}); err != nil {
		h.logAndSendError(w, "couldn't delete bucket", reqInfo, err)
	}
	w.WriteHeader(http.StatusNoContent)
}
