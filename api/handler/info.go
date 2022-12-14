package handler

import (
	"net/http"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
)

func (h *handler) GetBucketLocationHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, LocationResponse{Location: bktInfo.LocationConstraint}); err != nil {
		h.logAndSendError(w, "couldn't encode bucket location response", reqInfo, err)
	}
}
