package handler

import (
	"net/http"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
)

func (h *handler) DeleteBucketPolicyHandler(w http.ResponseWriter, r *http.Request) {
	h.logAndSendError(w, "not supported", api.GetReqInfo(r.Context()), errors.GetAPIError(errors.ErrNotSupported))
}

func (h *handler) DeleteBucketEncryptionHandler(w http.ResponseWriter, r *http.Request) {
	h.logAndSendError(w, "not supported", api.GetReqInfo(r.Context()), errors.GetAPIError(errors.ErrNotSupported))
}
