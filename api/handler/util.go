package handler

import (
	"context"
	errorsStd "errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"go.uber.org/zap"
)

func (h *handler) logAndSendError(w http.ResponseWriter, logText string, reqInfo *api.ReqInfo, err error, additional ...zap.Field) {
	code := api.WriteErrorResponse(w, reqInfo, transformToS3Error(err))
	fields := []zap.Field{
		zap.Int("status", code),
		zap.String("request_id", reqInfo.RequestID),
		zap.String("method", reqInfo.API),
		zap.String("bucket", reqInfo.BucketName),
		zap.String("object", reqInfo.ObjectName),
		zap.String("description", logText),
		zap.Error(err)}
	fields = append(fields, additional...)
	h.log.Error("call method", fields...)
}

func transformToS3Error(err error) error {
	if _, ok := err.(errors.Error); ok {
		return err
	}

	if errorsStd.Is(err, layer.ErrAccessDenied) ||
		errorsStd.Is(err, layer.ErrNodeAccessDenied) {
		return errors.GetAPIError(errors.ErrAccessDenied)
	}

	return errors.GetAPIError(errors.ErrInternalError)
}

func (h *handler) ResolveBucket(ctx context.Context, bucket string) (*data.BucketInfo, error) {
	return h.obj.GetBucketInfo(ctx, bucket)
}

func (h *handler) getBucketAndCheckOwner(r *http.Request, bucket string, header ...string) (*data.BucketInfo, error) {
	bktInfo, err := h.obj.GetBucketInfo(r.Context(), bucket)
	if err != nil {
		return nil, err
	}

	var expected string
	if len(header) == 0 {
		expected = r.Header.Get(api.AmzExpectedBucketOwner)
	} else {
		expected = r.Header.Get(header[0])
	}

	if len(expected) == 0 {
		return bktInfo, nil
	}

	return bktInfo, checkOwner(bktInfo, expected)
}

func parseRange(s string) (*layer.RangeParams, error) {
	if s == "" {
		return nil, nil
	}

	prefix := "bytes="

	if !strings.HasPrefix(s, prefix) {
		return nil, errors.GetAPIError(errors.ErrInvalidRange)
	}

	s = strings.TrimPrefix(s, prefix)

	valuesStr := strings.Split(s, "-")
	if len(valuesStr) != 2 {
		return nil, errors.GetAPIError(errors.ErrInvalidRange)
	}

	values := make([]uint64, 0, len(valuesStr))
	for _, v := range valuesStr {
		num, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, errors.GetAPIError(errors.ErrInvalidRange)
		}
		values = append(values, num)
	}
	if values[0] > values[1] {
		return nil, errors.GetAPIError(errors.ErrInvalidRange)
	}

	return &layer.RangeParams{
		Start: values[0],
		End:   values[1],
	}, nil
}

func getSessionTokenSetEACL(ctx context.Context) (*session.Container, error) {
	boxData, err := layer.GetBoxData(ctx)
	if err != nil {
		return nil, err
	}
	sessionToken := boxData.Gate.SessionTokenForSetEACL()
	if sessionToken == nil {
		return nil, errors.GetAPIError(errors.ErrAccessDenied)
	}

	return sessionToken, nil
}
