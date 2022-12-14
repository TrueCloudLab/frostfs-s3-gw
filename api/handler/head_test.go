package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/stretchr/testify/require"
)

func TestConditionalHead(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-conditional", "object"
	_, objInfo := createBucketAndObject(tc, bktName, objName)

	w, r := prepareTestRequest(tc, bktName, objName, nil)
	tc.Handler().HeadObjectHandler(w, r)
	assertStatus(t, w, http.StatusOK)
	etag := w.Result().Header.Get(api.ETag)

	headers := map[string]string{api.IfMatch: etag}
	headObject(t, tc, bktName, objName, headers, http.StatusOK)

	headers = map[string]string{api.IfMatch: "etag"}
	headObject(t, tc, bktName, objName, headers, http.StatusPreconditionFailed)

	headers = map[string]string{api.IfUnmodifiedSince: objInfo.Created.Add(time.Minute).Format(http.TimeFormat)}
	headObject(t, tc, bktName, objName, headers, http.StatusOK)

	var zeroTime time.Time
	headers = map[string]string{api.IfUnmodifiedSince: zeroTime.UTC().Format(http.TimeFormat)}
	headObject(t, tc, bktName, objName, headers, http.StatusPreconditionFailed)

	headers = map[string]string{
		api.IfMatch:           etag,
		api.IfUnmodifiedSince: zeroTime.UTC().Format(http.TimeFormat),
	}
	headObject(t, tc, bktName, objName, headers, http.StatusOK)

	headers = map[string]string{api.IfNoneMatch: etag}
	headObject(t, tc, bktName, objName, headers, http.StatusNotModified)

	headers = map[string]string{api.IfNoneMatch: "etag"}
	headObject(t, tc, bktName, objName, headers, http.StatusOK)

	headers = map[string]string{api.IfModifiedSince: zeroTime.UTC().Format(http.TimeFormat)}
	headObject(t, tc, bktName, objName, headers, http.StatusOK)

	headers = map[string]string{api.IfModifiedSince: time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)}
	headObject(t, tc, bktName, objName, headers, http.StatusNotModified)

	headers = map[string]string{
		api.IfNoneMatch:     etag,
		api.IfModifiedSince: zeroTime.UTC().Format(http.TimeFormat),
	}
	headObject(t, tc, bktName, objName, headers, http.StatusNotModified)
}

func headObject(t *testing.T, tc *handlerContext, bktName, objName string, headers map[string]string, status int) {
	w, r := prepareTestRequest(tc, bktName, objName, nil)

	for key, val := range headers {
		r.Header.Set(key, val)
	}

	tc.Handler().HeadObjectHandler(w, r)
	assertStatus(t, w, status)
}

func TestInvalidAccessThroughCache(t *testing.T) {
	tc := prepareHandlerContext(t)
	bktName, objName := "bucket-for-cache", "obj-for-cache"
	createBucketAndObject(tc, bktName, objName)

	headObject(t, tc, bktName, objName, nil, http.StatusOK)

	w, r := prepareTestRequest(tc, bktName, objName, nil)
	tc.Handler().HeadObjectHandler(w, r.WithContext(context.WithValue(r.Context(), api.BoxData, newTestAccessBox(t, nil))))
	assertStatus(t, w, http.StatusForbidden)
}

func newTestAccessBox(t *testing.T, key *keys.PrivateKey) *accessbox.Box {
	var err error
	if key == nil {
		key, err = keys.NewPrivateKey()
		require.NoError(t, err)
	}

	var btoken bearer.Token
	btoken.SetEACLTable(*eacl.NewTable())
	err = btoken.Sign(key.PrivateKey)
	require.NoError(t, err)

	return &accessbox.Box{
		Gate: &accessbox.GateData{
			BearerToken: &btoken,
		},
	}
}
