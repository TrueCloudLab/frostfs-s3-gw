package handler

import (
	"bytes"
	"net/http"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"go.uber.org/zap"
)

const sizeToDetectType = 512

func getRangeToDetectContentType(maxSize int64) *layer.RangeParams {
	end := uint64(maxSize)
	if sizeToDetectType < end {
		end = sizeToDetectType
	}

	return &layer.RangeParams{
		Start: 0,
		End:   end - 1,
	}
}

func (h *handler) HeadObjectHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	conditional, err := parseConditionalHeaders(r.Header)
	if err != nil {
		h.logAndSendError(w, "could not parse request params", reqInfo, err)
		return
	}

	p := &layer.HeadObjectParams{
		BktInfo:   bktInfo,
		Object:    reqInfo.ObjectName,
		VersionID: reqInfo.URL.Query().Get(api.QueryVersionID),
	}

	extendedInfo, err := h.obj.GetExtendedObjectInfo(r.Context(), p)
	if err != nil {
		h.logAndSendError(w, "could not find object", reqInfo, err)
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

	if err = checkPreconditions(info, conditional); err != nil {
		h.logAndSendError(w, "precondition failed", reqInfo, err)
		return
	}

	t := &layer.ObjectVersion{
		BktInfo:    bktInfo,
		ObjectName: info.Name,
		VersionID:  info.VersionID(),
	}

	tagSet, lockInfo, err := h.obj.GetObjectTaggingAndLock(r.Context(), t, extendedInfo.NodeVersion)
	if err != nil && !errors.IsS3Error(err, errors.ErrNoSuchKey) {
		h.logAndSendError(w, "could not get object meta data", reqInfo, err)
		return
	}

	if len(info.ContentType) == 0 {
		if info.ContentType = layer.MimeByFilePath(info.Name); len(info.ContentType) == 0 {
			buffer := bytes.NewBuffer(make([]byte, 0, sizeToDetectType))
			getParams := &layer.GetObjectParams{
				ObjectInfo: info,
				Writer:     buffer,
				Range:      getRangeToDetectContentType(info.Size),
				BucketInfo: bktInfo,
			}
			if err = h.obj.GetObject(r.Context(), getParams); err != nil {
				h.logAndSendError(w, "could not get object", reqInfo, err, zap.Stringer("oid", info.ID))
				return
			}
			info.ContentType = http.DetectContentType(buffer.Bytes())
		}
	}

	if err = h.setLockingHeaders(bktInfo, lockInfo, w.Header()); err != nil {
		h.logAndSendError(w, "could not get locking info", reqInfo, err)
		return
	}

	bktSettings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	writeHeaders(w.Header(), r.Header, extendedInfo, len(tagSet), bktSettings.Unversioned())
	w.WriteHeader(http.StatusOK)
}

func (h *handler) HeadBucketHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	w.Header().Set(api.ContainerID, bktInfo.CID.EncodeToString())
	w.Header().Set(api.ContainerName, bktInfo.Name)
	w.Header().Set(api.AmzBucketRegion, bktInfo.LocationConstraint)
	api.WriteResponse(w, http.StatusOK, nil, api.MimeNone)
}

func (h *handler) setLockingHeaders(bktInfo *data.BucketInfo, lockInfo *data.LockInfo, header http.Header) error {
	if !bktInfo.ObjectLockEnabled {
		return nil
	}

	legalHold := &data.LegalHold{Status: legalHoldOff}
	retention := &data.Retention{Mode: governanceMode}

	if lockInfo.IsLegalHoldSet() {
		legalHold.Status = legalHoldOn
	}
	if lockInfo.IsRetentionSet() {
		retention.RetainUntilDate = lockInfo.UntilDate()
		if lockInfo.IsCompliance() {
			retention.Mode = complianceMode
		}
	}

	writeLockHeaders(header, legalHold, retention)
	return nil
}

func writeLockHeaders(h http.Header, legalHold *data.LegalHold, retention *data.Retention) {
	h.Set(api.AmzObjectLockLegalHold, legalHold.Status)

	if retention.RetainUntilDate != "" {
		h.Set(api.AmzObjectLockRetainUntilDate, retention.RetainUntilDate)
		h.Set(api.AmzObjectLockMode, retention.Mode)
	}
}
