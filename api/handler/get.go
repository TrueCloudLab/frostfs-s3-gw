package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"go.uber.org/zap"
)

type conditionalArgs struct {
	IfModifiedSince   *time.Time
	IfUnmodifiedSince *time.Time
	IfMatch           string
	IfNoneMatch       string
}

func fetchRangeHeader(headers http.Header, fullSize uint64) (*layer.RangeParams, error) {
	const prefix = "bytes="
	rangeHeader := headers.Get("Range")
	if len(rangeHeader) == 0 {
		return nil, nil
	}
	if fullSize == 0 {
		return nil, errors.GetAPIError(errors.ErrInvalidRange)
	}
	if !strings.HasPrefix(rangeHeader, prefix) {
		return nil, fmt.Errorf("unknown unit in range header")
	}
	arr := strings.Split(strings.TrimPrefix(rangeHeader, prefix), "-")
	if len(arr) != 2 || (len(arr[0]) == 0 && len(arr[1]) == 0) {
		return nil, fmt.Errorf("unknown byte-range-set")
	}

	var end, start uint64
	var err0, err1 error
	base, bitSize := 10, 64

	if len(arr[0]) == 0 {
		end, err1 = strconv.ParseUint(arr[1], base, bitSize)
		start = fullSize - end
		end = fullSize - 1
	} else if len(arr[1]) == 0 {
		start, err0 = strconv.ParseUint(arr[0], base, bitSize)
		end = fullSize - 1
	} else {
		start, err0 = strconv.ParseUint(arr[0], base, bitSize)
		end, err1 = strconv.ParseUint(arr[1], base, bitSize)
		if end > fullSize-1 {
			end = fullSize - 1
		}
	}

	if err0 != nil || err1 != nil || start > end || start > fullSize {
		return nil, errors.GetAPIError(errors.ErrInvalidRange)
	}
	return &layer.RangeParams{Start: start, End: end}, nil
}

func overrideResponseHeaders(h http.Header, query url.Values) {
	for key, value := range query {
		if hdr, ok := api.ResponseModifiers[strings.ToLower(key)]; ok {
			h[hdr] = value
		}
	}
}

func addSSECHeaders(responseHeader http.Header, requestHeader http.Header) {
	responseHeader.Set(api.AmzServerSideEncryptionCustomerAlgorithm, requestHeader.Get(api.AmzServerSideEncryptionCustomerAlgorithm))
	responseHeader.Set(api.AmzServerSideEncryptionCustomerKeyMD5, requestHeader.Get(api.AmzServerSideEncryptionCustomerKeyMD5))
}

func writeHeaders(h http.Header, requestHeader http.Header, extendedInfo *data.ExtendedObjectInfo, tagSetLength int, isBucketUnversioned bool) {
	info := extendedInfo.ObjectInfo
	if len(info.ContentType) > 0 && h.Get(api.ContentType) == "" {
		h.Set(api.ContentType, info.ContentType)
	}
	h.Set(api.LastModified, info.Created.UTC().Format(http.TimeFormat))

	if len(info.Headers[layer.AttributeEncryptionAlgorithm]) > 0 {
		h.Set(api.ContentLength, info.Headers[layer.AttributeDecryptedSize])
		addSSECHeaders(h, requestHeader)
	} else {
		h.Set(api.ContentLength, strconv.FormatInt(info.Size, 10))
	}

	h.Set(api.ETag, info.HashSum)
	h.Set(api.AmzTaggingCount, strconv.Itoa(tagSetLength))

	if !isBucketUnversioned {
		h.Set(api.AmzVersionID, extendedInfo.Version())
	}

	if cacheControl := info.Headers[api.CacheControl]; cacheControl != "" {
		h.Set(api.CacheControl, cacheControl)
	}
	if expires := info.Headers[api.Expires]; expires != "" {
		h.Set(api.Expires, expires)
	}

	for key, val := range info.Headers {
		if layer.IsSystemHeader(key) {
			continue
		}
		h[api.MetadataPrefix+key] = []string{val}
	}
}

func (h *handler) GetObjectHandler(w http.ResponseWriter, r *http.Request) {
	var (
		params *layer.RangeParams

		reqInfo = api.GetReqInfo(r.Context())
	)

	conditional, err := parseConditionalHeaders(r.Header)
	if err != nil {
		h.logAndSendError(w, "could not parse request params", reqInfo, err)
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
		VersionID: reqInfo.URL.Query().Get(api.QueryVersionID),
	}

	extendedInfo, err := h.obj.GetExtendedObjectInfo(r.Context(), p)
	if err != nil {
		h.logAndSendError(w, "could not find object", reqInfo, err)
		return
	}
	info := extendedInfo.ObjectInfo

	if err = checkPreconditions(info, conditional); err != nil {
		h.logAndSendError(w, "precondition failed", reqInfo, err)
		return
	}

	encryptionParams, err := formEncryptionParams(r)
	if err != nil {
		h.logAndSendError(w, "invalid sse headers", reqInfo, err)
		return
	}

	if err = encryptionParams.MatchObjectEncryption(layer.FormEncryptionInfo(info.Headers)); err != nil {
		h.logAndSendError(w, "encryption doesn't match object", reqInfo, errors.GetAPIError(errors.ErrBadRequest), zap.Error(err))
		return
	}

	fullSize := info.Size
	if encryptionParams.Enabled() {
		if fullSize, err = strconv.ParseInt(info.Headers[layer.AttributeDecryptedSize], 10, 64); err != nil {
			h.logAndSendError(w, "invalid decrypted size header", reqInfo, errors.GetAPIError(errors.ErrBadRequest))
			return
		}
	}

	if params, err = fetchRangeHeader(r.Header, uint64(fullSize)); err != nil {
		h.logAndSendError(w, "could not parse range header", reqInfo, err)
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

	if layer.IsAuthenticatedRequest(r.Context()) {
		overrideResponseHeaders(w.Header(), reqInfo.URL.Query())
	}

	if err = h.setLockingHeaders(bktInfo, lockInfo, w.Header()); err != nil {
		h.logAndSendError(w, "could not get locking info", reqInfo, err)
		return
	}

	if err = h.setExpirationHeader(r.Context(), bktInfo, info, w.Header()); err != nil {
		h.logAndSendError(w, "could not get expiration info", reqInfo, err)
		return
	}

	bktSettings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	writeHeaders(w.Header(), r.Header, extendedInfo, len(tagSet), bktSettings.Unversioned())
	if params != nil {
		writeRangeHeaders(w, params, info.Size)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	getParams := &layer.GetObjectParams{
		ObjectInfo: info,
		Writer:     w,
		Range:      params,
		BucketInfo: bktInfo,
		Encryption: encryptionParams,
	}
	if err = h.obj.GetObject(r.Context(), getParams); err != nil {
		h.logAndSendError(w, "could not get object", reqInfo, err)
	}
}

func checkPreconditions(info *data.ObjectInfo, args *conditionalArgs) error {
	if len(args.IfMatch) > 0 && args.IfMatch != info.HashSum {
		return errors.GetAPIError(errors.ErrPreconditionFailed)
	}
	if len(args.IfNoneMatch) > 0 && args.IfNoneMatch == info.HashSum {
		return errors.GetAPIError(errors.ErrNotModified)
	}
	if args.IfModifiedSince != nil && info.Created.Before(*args.IfModifiedSince) {
		return errors.GetAPIError(errors.ErrNotModified)
	}
	if args.IfUnmodifiedSince != nil && info.Created.After(*args.IfUnmodifiedSince) {
		if len(args.IfMatch) == 0 {
			return errors.GetAPIError(errors.ErrPreconditionFailed)
		}
	}

	return nil
}

func parseConditionalHeaders(headers http.Header) (*conditionalArgs, error) {
	var err error
	args := &conditionalArgs{
		IfMatch:     headers.Get(api.IfMatch),
		IfNoneMatch: headers.Get(api.IfNoneMatch),
	}

	if args.IfModifiedSince, err = parseHTTPTime(headers.Get(api.IfModifiedSince)); err != nil {
		return nil, err
	}
	if args.IfUnmodifiedSince, err = parseHTTPTime(headers.Get(api.IfUnmodifiedSince)); err != nil {
		return nil, err
	}

	return args, nil
}

func parseHTTPTime(data string) (*time.Time, error) {
	if len(data) == 0 {
		return nil, nil
	}

	result, err := time.Parse(http.TimeFormat, data)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse http time %s: %w", data, err)
	}
	return &result, nil
}

func writeRangeHeaders(w http.ResponseWriter, params *layer.RangeParams, size int64) {
	w.Header().Set(api.AcceptRanges, "bytes")
	w.Header().Set(api.ContentRange, fmt.Sprintf("bytes %d-%d/%d", params.Start, params.End, size))
	w.Header().Set(api.ContentLength, strconv.FormatUint(params.End-params.Start+1, 10))
	w.WriteHeader(http.StatusPartialContent)
}
