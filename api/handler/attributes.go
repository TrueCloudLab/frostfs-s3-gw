package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"go.uber.org/zap"
)

type (
	GetObjectAttributesResponse struct {
		ETag         string       `xml:"ETag,omitempty"`
		Checksum     *Checksum    `xml:"Checksum,omitempty"`
		ObjectSize   int64        `xml:"ObjectSize,omitempty"`
		StorageClass string       `xml:"StorageClass,omitempty"`
		ObjectParts  *ObjectParts `xml:"ObjectParts,omitempty"`
	}

	Checksum struct {
		ChecksumSHA256 string `xml:"ChecksumSHA256,omitempty"`
	}

	ObjectParts struct {
		IsTruncated          bool   `xml:"IsTruncated,omitempty"`
		MaxParts             int    `xml:"MaxParts,omitempty"`
		NextPartNumberMarker int    `xml:"NextPartNumberMarker,omitempty"`
		PartNumberMarker     int    `xml:"PartNumberMarker,omitempty"`
		Parts                []Part `xml:"Part,omitempty"`
		PartsCount           int    `xml:"PartsCount,omitempty"`
	}

	Part struct {
		ChecksumSHA256 string `xml:"ChecksumSHA256,omitempty"`
		PartNumber     int    `xml:"PartNumber,omitempty"`
		Size           int    `xml:"Size,omitempty"`
	}

	GetObjectAttributesArgs struct {
		MaxParts         int
		PartNumberMarker int
		Attributes       []string
		VersionID        string
		Conditional      *conditionalArgs
	}
)

const (
	eTag         = "ETag"
	checksum     = "Checksum"
	objectParts  = "ObjectParts"
	storageClass = "StorageClass"
	objectSize   = "ObjectSize"
)

var validAttributes = map[string]struct{}{
	eTag:         {},
	checksum:     {},
	objectParts:  {},
	storageClass: {},
	objectSize:   {},
}

func (h *handler) GetObjectAttributesHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	params, err := parseGetObjectAttributeArgs(r)
	if err != nil {
		h.logAndSendError(w, "invalid request", reqInfo, err)
		return
	}

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	p := &layer.HeadObjectParams{
		BktInfo:   bktInfo,
		Object:    reqInfo.ObjectName,
		VersionID: params.VersionID,
	}

	extendedInfo, err := h.obj.GetExtendedObjectInfo(r.Context(), p)
	if err != nil {
		h.logAndSendError(w, "could not fetch object info", reqInfo, err)
		return
	}
	info := extendedInfo.ObjectInfo

	encryptionParams, err := formEncryptionParams(r)
	if err != nil {
		h.logAndSendError(w, "invalid sse headers", reqInfo, err)
		return
	}

	if err = encryptionParams.MatchObjectEncryption(layer.FormEncryptionInfo(info.Headers)); err != nil {
		h.logAndSendError(w, "encryption doesn't match object", reqInfo, errors.GetAPIError(errors.ErrBadRequest), zap.Error(err))
		return
	}

	if err = checkPreconditions(info, params.Conditional); err != nil {
		h.logAndSendError(w, "precondition failed", reqInfo, err)
		return
	}

	bktSettings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	response, err := encodeToObjectAttributesResponse(info, params)
	if err != nil {
		h.logAndSendError(w, "couldn't encode object info to response", reqInfo, err)
		return
	}

	writeAttributesHeaders(w.Header(), extendedInfo, bktSettings.Unversioned())
	if err = api.EncodeToResponse(w, response); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func writeAttributesHeaders(h http.Header, info *data.ExtendedObjectInfo, isBucketUnversioned bool) {
	h.Set(api.LastModified, info.ObjectInfo.Created.UTC().Format(http.TimeFormat))
	if !isBucketUnversioned {
		h.Set(api.AmzVersionID, info.Version())
	}

	if info.NodeVersion.IsDeleteMarker() {
		h.Set(api.AmzDeleteMarker, strconv.FormatBool(true))
	}

	// x-amz-request-charged
}

func parseGetObjectAttributeArgs(r *http.Request) (*GetObjectAttributesArgs, error) {
	res := &GetObjectAttributesArgs{
		VersionID: r.URL.Query().Get(api.QueryVersionID),
	}

	attributesVal := r.Header.Get(api.AmzObjectAttributes)
	if attributesVal == "" {
		return nil, errors.GetAPIError(errors.ErrInvalidAttributeName)
	}

	attributes := strings.Split(attributesVal, ",")
	for _, a := range attributes {
		if _, ok := validAttributes[a]; !ok {
			return nil, errors.GetAPIError(errors.ErrInvalidAttributeName)
		}
		res.Attributes = append(res.Attributes, a)
	}

	var err error
	maxPartsVal := r.Header.Get(api.AmzMaxParts)
	if maxPartsVal == "" {
		res.MaxParts = layer.MaxSizePartsList
	} else if res.MaxParts, err = strconv.Atoi(maxPartsVal); err != nil || res.MaxParts < 0 {
		return nil, errors.GetAPIError(errors.ErrInvalidMaxKeys)
	}

	markerVal := r.Header.Get(api.AmzPartNumberMarker)
	if markerVal != "" {
		if res.PartNumberMarker, err = strconv.Atoi(markerVal); err != nil || res.PartNumberMarker < 0 {
			return nil, errors.GetAPIError(errors.ErrInvalidPartNumberMarker)
		}
	}

	res.Conditional, err = parseConditionalHeaders(r.Header)
	return res, err
}

func encodeToObjectAttributesResponse(info *data.ObjectInfo, p *GetObjectAttributesArgs) (*GetObjectAttributesResponse, error) {
	resp := &GetObjectAttributesResponse{}

	for _, attr := range p.Attributes {
		switch attr {
		case eTag:
			resp.ETag = info.HashSum
		case storageClass:
			resp.StorageClass = "STANDARD"
		case objectSize:
			resp.ObjectSize = info.Size
		case checksum:
			resp.Checksum = &Checksum{ChecksumSHA256: info.HashSum}
		case objectParts:
			parts, err := formUploadAttributes(info, p.MaxParts, p.PartNumberMarker)
			if err != nil {
				return nil, fmt.Errorf("form upload attributes: %w", err)
			}
			if parts != nil {
				resp.ObjectParts = parts
			}
		}
	}

	return resp, nil
}

func formUploadAttributes(info *data.ObjectInfo, maxParts, marker int) (*ObjectParts, error) {
	completedParts, ok := info.Headers[layer.UploadCompletedParts]
	if !ok {
		return nil, nil
	}

	partInfos := strings.Split(completedParts, ",")
	parts := make([]Part, len(partInfos))
	for i, p := range partInfos {
		part, err := layer.ParseCompletedPartHeader(p)
		if err != nil {
			return nil, fmt.Errorf("invalid completed part: %w", err)
		}
		parts[i] = Part{
			PartNumber:     part.PartNumber,
			Size:           int(part.Size),
			ChecksumSHA256: part.ETag,
		}
	}

	res := &ObjectParts{
		PartsCount: len(parts),
	}

	if marker != 0 {
		res.PartNumberMarker = marker
		var found bool
		for i, n := range parts {
			if n.PartNumber == marker {
				parts = parts[i:]
				found = true
				break
			}
		}
		if !found {
			return nil, errors.GetAPIError(errors.ErrInvalidPartNumberMarker)
		}
	}

	res.MaxParts = maxParts
	if len(parts) > maxParts {
		res.IsTruncated = true
		res.NextPartNumberMarker = parts[maxParts].PartNumber
		parts = parts[:maxParts]
	}

	res.Parts = parts

	return res, nil
}
