package handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/resolver"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/netmap"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type handlerContext struct {
	owner   user.ID
	t       *testing.T
	h       *handler
	tp      *layer.TestNeoFS
	context context.Context
}

func (hc *handlerContext) Handler() *handler {
	return hc.h
}

func (hc *handlerContext) MockedPool() *layer.TestNeoFS {
	return hc.tp
}

func (hc *handlerContext) Layer() layer.Client {
	return hc.h.obj
}

func (hc *handlerContext) Context() context.Context {
	return hc.context
}

type placementPolicyMock struct {
	defaultPolicy netmap.PlacementPolicy
}

func (p *placementPolicyMock) Default() netmap.PlacementPolicy {
	return p.defaultPolicy
}

func (p *placementPolicyMock) Get(string) (netmap.PlacementPolicy, bool) {
	return netmap.PlacementPolicy{}, false
}

func prepareHandlerContext(t *testing.T) *handlerContext {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	l := zap.NewExample()
	tp := layer.NewTestNeoFS()

	testResolver := &resolver.Resolver{Name: "test_resolver"}
	testResolver.SetResolveFunc(func(_ context.Context, name string) (cid.ID, error) {
		return tp.ContainerID(name)
	})

	var owner user.ID
	user.IDFromKey(&owner, key.PrivateKey.PublicKey)

	layerCfg := &layer.Config{
		Caches:      layer.DefaultCachesConfigs(zap.NewExample()),
		AnonKey:     layer.AnonymousKey{Key: key},
		Resolver:    testResolver,
		TreeService: layer.NewTreeService(),
	}

	var pp netmap.PlacementPolicy
	err = pp.DecodeString("REP 1")
	require.NoError(t, err)

	h := &handler{
		log: l,
		obj: layer.NewLayer(l, tp, layerCfg),
		cfg: &Config{
			Policy: &placementPolicyMock{defaultPolicy: pp},
		},
	}

	return &handlerContext{
		owner:   owner,
		t:       t,
		h:       h,
		tp:      tp,
		context: context.WithValue(context.Background(), api.BoxData, newTestAccessBox(t, key)),
	}
}

func createTestBucket(hc *handlerContext, bktName string) *data.BucketInfo {
	_, err := hc.MockedPool().CreateContainer(hc.Context(), layer.PrmContainerCreate{
		Creator: hc.owner,
		Name:    bktName,
	})
	require.NoError(hc.t, err)

	bktInfo, err := hc.Layer().GetBucketInfo(hc.Context(), bktName)
	require.NoError(hc.t, err)
	return bktInfo
}

func createTestBucketWithLock(hc *handlerContext, bktName string, conf *data.ObjectLockConfiguration) *data.BucketInfo {
	cnrID, err := hc.MockedPool().CreateContainer(hc.Context(), layer.PrmContainerCreate{
		Creator:              hc.owner,
		Name:                 bktName,
		AdditionalAttributes: [][2]string{{layer.AttributeLockEnabled, "true"}},
	})
	require.NoError(hc.t, err)

	var ownerID user.ID

	bktInfo := &data.BucketInfo{
		CID:               cnrID,
		Name:              bktName,
		ObjectLockEnabled: true,
		Owner:             ownerID,
	}

	sp := &layer.PutSettingsParams{
		BktInfo: bktInfo,
		Settings: &data.BucketSettings{
			Versioning:        data.VersioningEnabled,
			LockConfiguration: conf,
		},
	}

	err = hc.Layer().PutBucketSettings(hc.Context(), sp)
	require.NoError(hc.t, err)

	return bktInfo
}

func createTestObject(hc *handlerContext, bktInfo *data.BucketInfo, objName string) *data.ObjectInfo {
	content := make([]byte, 1024)
	_, err := rand.Read(content)
	require.NoError(hc.t, err)

	header := map[string]string{
		object.AttributeTimestamp: strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}

	extObjInfo, err := hc.Layer().PutObject(hc.Context(), &layer.PutObjectParams{
		BktInfo: bktInfo,
		Object:  objName,
		Size:    int64(len(content)),
		Reader:  bytes.NewReader(content),
		Header:  header,
	})
	require.NoError(hc.t, err)

	return extObjInfo.ObjectInfo
}

func prepareTestRequest(hc *handlerContext, bktName, objName string, body interface{}) (*httptest.ResponseRecorder, *http.Request) {
	return prepareTestFullRequest(hc, bktName, objName, make(url.Values), body)
}

func prepareTestFullRequest(hc *handlerContext, bktName, objName string, query url.Values, body interface{}) (*httptest.ResponseRecorder, *http.Request) {
	rawBody, err := xml.Marshal(body)
	require.NoError(hc.t, err)

	return prepareTestRequestWithQuery(hc, bktName, objName, query, rawBody)
}

func prepareTestRequestWithQuery(hc *handlerContext, bktName, objName string, query url.Values, body []byte) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, defaultURL, bytes.NewReader(body))
	r.URL.RawQuery = query.Encode()

	reqInfo := api.NewReqInfo(w, r, api.ObjectRequest{Bucket: bktName, Object: objName})
	r = r.WithContext(api.SetReqInfo(hc.Context(), reqInfo))

	return w, r
}

func prepareTestPayloadRequest(hc *handlerContext, bktName, objName string, payload io.Reader) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, defaultURL, payload)

	reqInfo := api.NewReqInfo(w, r, api.ObjectRequest{Bucket: bktName, Object: objName})
	r = r.WithContext(api.SetReqInfo(hc.Context(), reqInfo))

	return w, r
}

func parseTestResponse(t *testing.T, response *httptest.ResponseRecorder, body interface{}) {
	assertStatus(t, response, http.StatusOK)
	err := xml.NewDecoder(response.Result().Body).Decode(body)
	require.NoError(t, err)
}

func existInMockedNeoFS(tc *handlerContext, bktInfo *data.BucketInfo, objInfo *data.ObjectInfo) bool {
	p := &layer.GetObjectParams{
		BucketInfo: bktInfo,
		ObjectInfo: objInfo,
		Writer:     io.Discard,
	}

	return tc.Layer().GetObject(tc.Context(), p) == nil
}

func listOIDsFromMockedNeoFS(t *testing.T, tc *handlerContext, bktName string) []oid.ID {
	bktInfo, err := tc.Layer().GetBucketInfo(tc.Context(), bktName)
	require.NoError(t, err)

	return tc.MockedPool().AllObjects(bktInfo.CID)
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	if w.Code != status {
		resp, err := io.ReadAll(w.Result().Body)
		require.NoError(t, err)
		require.Failf(t, "unexpected status", "expected: %d, actual: %d, resp: '%s'", status, w.Code, string(resp))
	}
}

func readResponse(t *testing.T, w *httptest.ResponseRecorder, status int, model interface{}) {
	assertStatus(t, w, status)
	err := xml.NewDecoder(w.Result().Body).Decode(model)
	require.NoError(t, err)
}
