package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
)

// ListObjectsV1Handler handles objects listing requests for API version 1.
func (h *handler) ListObjectsV1Handler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	params, err := parseListObjectsArgsV1(reqInfo)
	if err != nil {
		h.logAndSendError(w, "failed to parse arguments", reqInfo, err)
		return
	}

	if params.BktInfo, err = h.getBucketAndCheckOwner(r, reqInfo.BucketName); err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	list, err := h.obj.ListObjectsV1(r.Context(), params)
	if err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, encodeV1(params, list)); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func encodeV1(p *layer.ListObjectsParamsV1, list *layer.ListObjectsInfoV1) *ListObjectsV1Response {
	res := &ListObjectsV1Response{
		Name:         p.BktInfo.Name,
		EncodingType: p.Encode,
		Marker:       p.Marker,
		Prefix:       p.Prefix,
		MaxKeys:      p.MaxKeys,
		Delimiter:    p.Delimiter,
		IsTruncated:  list.IsTruncated,
		NextMarker:   list.NextMarker,
	}

	res.CommonPrefixes = fillPrefixes(list.Prefixes, p.Encode)

	res.Contents = fillContentsWithOwner(list.Objects, p.Encode)

	return res
}

// ListObjectsV2Handler handles objects listing requests for API version 2.
func (h *handler) ListObjectsV2Handler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	params, err := parseListObjectsArgsV2(reqInfo)
	if err != nil {
		h.logAndSendError(w, "failed to parse arguments", reqInfo, err)
		return
	}

	if params.BktInfo, err = h.getBucketAndCheckOwner(r, reqInfo.BucketName); err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	list, err := h.obj.ListObjectsV2(r.Context(), params)
	if err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, encodeV2(params, list)); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func encodeV2(p *layer.ListObjectsParamsV2, list *layer.ListObjectsInfoV2) *ListObjectsV2Response {
	res := &ListObjectsV2Response{
		Name:                  p.BktInfo.Name,
		EncodingType:          p.Encode,
		Prefix:                s3PathEncode(p.Prefix, p.Encode),
		KeyCount:              len(list.Objects) + len(list.Prefixes),
		MaxKeys:               p.MaxKeys,
		Delimiter:             s3PathEncode(p.Delimiter, p.Encode),
		StartAfter:            s3PathEncode(p.StartAfter, p.Encode),
		IsTruncated:           list.IsTruncated,
		ContinuationToken:     p.ContinuationToken,
		NextContinuationToken: list.NextContinuationToken,
	}

	res.CommonPrefixes = fillPrefixes(list.Prefixes, p.Encode)

	res.Contents = fillContents(list.Objects, p.Encode, p.FetchOwner)

	return res
}

func parseListObjectsArgsV1(reqInfo *api.ReqInfo) (*layer.ListObjectsParamsV1, error) {
	var (
		res         layer.ListObjectsParamsV1
		queryValues = reqInfo.URL.Query()
	)

	common, err := parseListObjectArgs(reqInfo)
	if err != nil {
		return nil, err
	}
	res.ListObjectsParamsCommon = *common

	res.Marker = queryValues.Get("marker")

	return &res, nil
}

func parseListObjectsArgsV2(reqInfo *api.ReqInfo) (*layer.ListObjectsParamsV2, error) {
	var (
		res         layer.ListObjectsParamsV2
		queryValues = reqInfo.URL.Query()
	)

	common, err := parseListObjectArgs(reqInfo)
	if err != nil {
		return nil, err
	}
	res.ListObjectsParamsCommon = *common

	res.ContinuationToken, err = parseContinuationToken(queryValues)
	if err != nil {
		return nil, err
	}

	res.StartAfter = queryValues.Get("start-after")
	res.FetchOwner, _ = strconv.ParseBool(queryValues.Get("fetch-owner"))
	return &res, nil
}

func parseListObjectArgs(reqInfo *api.ReqInfo) (*layer.ListObjectsParamsCommon, error) {
	var (
		err         error
		res         layer.ListObjectsParamsCommon
		queryValues = reqInfo.URL.Query()
	)

	res.Delimiter = queryValues.Get("delimiter")
	res.Encode = queryValues.Get("encoding-type")

	if queryValues.Get("max-keys") == "" {
		res.MaxKeys = maxObjectList
	} else if res.MaxKeys, err = strconv.Atoi(queryValues.Get("max-keys")); err != nil || res.MaxKeys < 0 {
		return nil, errors.GetAPIError(errors.ErrInvalidMaxKeys)
	}

	res.Prefix = queryValues.Get("prefix")

	return &res, nil
}

func parseContinuationToken(queryValues url.Values) (string, error) {
	if val, ok := queryValues["continuation-token"]; ok {
		var objID oid.ID
		if err := objID.DecodeString(val[0]); err != nil {
			return "", errors.GetAPIError(errors.ErrIncorrectContinuationToken)
		}
		return val[0], nil
	}
	return "", nil
}

func fillPrefixes(src []string, encode string) []CommonPrefix {
	var dst []CommonPrefix
	for _, obj := range src {
		dst = append(dst, CommonPrefix{
			Prefix: s3PathEncode(obj, encode),
		})
	}
	return dst
}

func fillContentsWithOwner(src []*data.ObjectInfo, encode string) []Object {
	return fillContents(src, encode, true)
}

func fillContents(src []*data.ObjectInfo, encode string, fetchOwner bool) []Object {
	var dst []Object
	for _, obj := range src {
		res := Object{
			Key:          s3PathEncode(obj.Name, encode),
			Size:         obj.Size,
			LastModified: obj.Created.UTC().Format(time.RFC3339),
			ETag:         obj.HashSum,
		}

		if fetchOwner {
			res.Owner = &Owner{
				ID:          obj.Owner.String(),
				DisplayName: obj.Owner.String(),
			}
		}

		dst = append(dst, res)
	}
	return dst
}

func (h *handler) ListBucketObjectVersionsHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	p, err := parseListObjectVersionsRequest(reqInfo)
	if err != nil {
		h.logAndSendError(w, "failed to parse request", reqInfo, err)
		return
	}

	if p.BktInfo, err = h.getBucketAndCheckOwner(r, reqInfo.BucketName); err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	info, err := h.obj.ListObjectVersions(r.Context(), p)
	if err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
		return
	}

	response := encodeListObjectVersionsToResponse(info, p.BktInfo.Name)
	if err = api.EncodeToResponse(w, response); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func parseListObjectVersionsRequest(reqInfo *api.ReqInfo) (*layer.ListObjectVersionsParams, error) {
	var (
		err         error
		res         layer.ListObjectVersionsParams
		queryValues = reqInfo.URL.Query()
	)

	if queryValues.Get("max-keys") == "" {
		res.MaxKeys = maxObjectList
	} else if res.MaxKeys, err = strconv.Atoi(queryValues.Get("max-keys")); err != nil || res.MaxKeys <= 0 {
		return nil, errors.GetAPIError(errors.ErrInvalidMaxKeys)
	}

	res.Prefix = queryValues.Get("prefix")
	res.KeyMarker = queryValues.Get("marker")
	res.Delimiter = queryValues.Get("delimiter")
	res.Encode = queryValues.Get("encoding-type")
	res.VersionIDMarker = queryValues.Get("version-id-marker")

	return &res, nil
}

func encodeListObjectVersionsToResponse(info *layer.ListObjectVersionsInfo, bucketName string) *ListObjectsVersionsResponse {
	res := ListObjectsVersionsResponse{
		Name:                bucketName,
		IsTruncated:         info.IsTruncated,
		KeyMarker:           info.KeyMarker,
		NextKeyMarker:       info.NextKeyMarker,
		NextVersionIDMarker: info.NextVersionIDMarker,
		VersionIDMarker:     info.VersionIDMarker,
	}

	for _, prefix := range info.CommonPrefixes {
		res.CommonPrefixes = append(res.CommonPrefixes, CommonPrefix{Prefix: prefix})
	}

	for _, ver := range info.Version {
		res.Version = append(res.Version, ObjectVersionResponse{
			IsLatest:     ver.IsLatest,
			Key:          ver.ObjectInfo.Name,
			LastModified: ver.ObjectInfo.Created.UTC().Format(time.RFC3339),
			Owner: Owner{
				ID:          ver.ObjectInfo.Owner.String(),
				DisplayName: ver.ObjectInfo.Owner.String(),
			},
			Size:      ver.ObjectInfo.Size,
			VersionID: ver.Version(),
			ETag:      ver.ObjectInfo.HashSum,
		})
	}
	// this loop is not starting till versioning is not implemented
	for _, del := range info.DeleteMarker {
		res.DeleteMarker = append(res.DeleteMarker, DeleteMarkerEntry{
			IsLatest:     del.IsLatest,
			Key:          del.ObjectInfo.Name,
			LastModified: del.ObjectInfo.Created.UTC().Format(time.RFC3339),
			Owner: Owner{
				ID:          del.ObjectInfo.Owner.String(),
				DisplayName: del.ObjectInfo.Owner.String(),
			},
			VersionID: del.Version(),
		})
	}

	return &res
}
