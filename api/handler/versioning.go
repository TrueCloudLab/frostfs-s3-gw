package handler

import (
	"encoding/xml"
	"net/http"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
)

func (h *handler) PutBucketVersioningHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	configuration := new(VersioningConfiguration)
	if err := xml.NewDecoder(r.Body).Decode(configuration); err != nil {
		h.logAndSendError(w, "couldn't decode versioning configuration", reqInfo, errors.GetAPIError(errors.ErrIllegalVersioningConfigurationException))
		return
	}

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	settings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "couldn't get bucket settings", reqInfo, err)
		return
	}

	if configuration.Status != data.VersioningEnabled && configuration.Status != data.VersioningSuspended {
		h.logAndSendError(w, "invalid versioning configuration", reqInfo, errors.GetAPIError(errors.ErrMalformedXML))
		return
	}

	// settings pointer is stored in the cache, so modify a copy of the settings
	newSettings := *settings
	newSettings.Versioning = configuration.Status

	p := &layer.PutSettingsParams{
		BktInfo:  bktInfo,
		Settings: &newSettings,
	}

	if p.Settings.VersioningSuspended() && bktInfo.ObjectLockEnabled {
		h.logAndSendError(w, "couldn't suspend bucket versioning", reqInfo, errors.GetAPIError(errors.ErrObjectLockConfigurationVersioningCannotBeChanged))
		return
	}

	if err = h.obj.PutBucketSettings(r.Context(), p); err != nil {
		h.logAndSendError(w, "couldn't put update versioning settings", reqInfo, err)
	}
}

// GetBucketVersioningHandler implements bucket versioning getter handler.
func (h *handler) GetBucketVersioningHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	settings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "couldn't get version settings", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, formVersioningConfiguration(settings)); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func formVersioningConfiguration(settings *data.BucketSettings) *VersioningConfiguration {
	res := &VersioningConfiguration{}
	if !settings.Unversioned() {
		res.Status = settings.Versioning
	}

	return res
}
