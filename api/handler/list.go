package handler

import (
	"net/http"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
)

const maxObjectList = 1000 // Limit number of objects in a listObjectsResponse/listObjectsVersionsResponse.

// ListBucketsHandler handles bucket listing requests.
func (h *handler) ListBucketsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		own     user.ID
		res     *ListBucketsResponse
		reqInfo = api.GetReqInfo(r.Context())
	)

	list, err := h.obj.ListBuckets(r.Context())
	if err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
		return
	}

	if len(list) > 0 {
		own = list[0].Owner
	}

	res = &ListBucketsResponse{
		Owner: Owner{
			ID:          own.String(),
			DisplayName: own.String(),
		},
	}

	for _, item := range list {
		res.Buckets.Buckets = append(res.Buckets.Buckets, Bucket{
			Name:         item.Name,
			CreationDate: item.Created.UTC().Format(time.RFC3339),
		})
	}

	if err = api.EncodeToResponse(w, res); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}
