package layer

import (
	"bytes"
	"context"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	bearertest "github.com/TrueCloudLab/frostfs-sdk-go/bearer/test"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (tc *testContext) putObject(content []byte) *data.ObjectInfo {
	extObjInfo, err := tc.layer.PutObject(tc.ctx, &PutObjectParams{
		BktInfo: tc.bktInfo,
		Object:  tc.obj,
		Size:    int64(len(content)),
		Reader:  bytes.NewReader(content),
		Header:  make(map[string]string),
	})
	require.NoError(tc.t, err)

	return extObjInfo.ObjectInfo
}

func (tc *testContext) getObject(objectName, versionID string, needError bool) (*data.ObjectInfo, []byte) {
	objInfo, err := tc.layer.GetObjectInfo(tc.ctx, &HeadObjectParams{
		BktInfo:   tc.bktInfo,
		Object:    objectName,
		VersionID: versionID,
	})
	if needError {
		require.Error(tc.t, err)
		return nil, nil
	}
	require.NoError(tc.t, err)

	content := bytes.NewBuffer(nil)
	err = tc.layer.GetObject(tc.ctx, &GetObjectParams{
		ObjectInfo: objInfo,
		Writer:     content,
		BucketInfo: tc.bktInfo,
	})
	require.NoError(tc.t, err)

	return objInfo, content.Bytes()
}

func (tc *testContext) deleteObject(objectName, versionID string, settings *data.BucketSettings) {
	p := &DeleteObjectParams{
		BktInfo:  tc.bktInfo,
		Settings: settings,
		Objects: []*VersionedObject{
			{Name: objectName, VersionID: versionID},
		},
	}
	deletedObjects := tc.layer.DeleteObjects(tc.ctx, p)
	for _, obj := range deletedObjects {
		require.NoError(tc.t, obj.Error)
	}
}

func (tc *testContext) listObjectsV1() []*data.ObjectInfo {
	res, err := tc.layer.ListObjectsV1(tc.ctx, &ListObjectsParamsV1{
		ListObjectsParamsCommon: ListObjectsParamsCommon{
			BktInfo: tc.bktInfo,
			MaxKeys: 1000,
		},
	})
	require.NoError(tc.t, err)
	return res.Objects
}

func (tc *testContext) listObjectsV2() []*data.ObjectInfo {
	res, err := tc.layer.ListObjectsV2(tc.ctx, &ListObjectsParamsV2{
		ListObjectsParamsCommon: ListObjectsParamsCommon{
			BktInfo: tc.bktInfo,
			MaxKeys: 1000,
		},
	})
	require.NoError(tc.t, err)
	return res.Objects
}

func (tc *testContext) listVersions() *ListObjectVersionsInfo {
	res, err := tc.layer.ListObjectVersions(tc.ctx, &ListObjectVersionsParams{
		BktInfo: tc.bktInfo,
		MaxKeys: 1000,
	})
	require.NoError(tc.t, err)
	return res
}

func (tc *testContext) checkListObjects(ids ...oid.ID) {
	objs := tc.listObjectsV1()
	require.Equal(tc.t, len(ids), len(objs))
	for _, id := range ids {
		require.Contains(tc.t, ids, id)
	}

	objs = tc.listObjectsV2()
	require.Equal(tc.t, len(ids), len(objs))
	for _, id := range ids {
		require.Contains(tc.t, ids, id)
	}
}

func (tc *testContext) getObjectByID(objID oid.ID) *object.Object {
	for _, obj := range tc.testNeoFS.Objects() {
		id, _ := obj.ID()
		if id.Equals(objID) {
			return obj
		}
	}
	return nil
}

type testContext struct {
	t         *testing.T
	ctx       context.Context
	layer     Client
	bktInfo   *data.BucketInfo
	obj       string
	testNeoFS *TestNeoFS
}

func prepareContext(t *testing.T, cachesConfig ...*CachesConfig) *testContext {
	logger := zap.NewExample()

	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	bearerToken := bearertest.Token()
	require.NoError(t, bearerToken.Sign(key.PrivateKey))

	ctx := context.WithValue(context.Background(), api.BoxData, &accessbox.Box{
		Gate: &accessbox.GateData{
			BearerToken: &bearerToken,
			GateKey:     key.PublicKey(),
		},
	})
	tp := NewTestNeoFS()

	bktName := "testbucket1"
	bktID, err := tp.CreateContainer(ctx, PrmContainerCreate{
		Name: bktName,
	})
	require.NoError(t, err)

	config := DefaultCachesConfigs(logger)
	if len(cachesConfig) != 0 {
		config = cachesConfig[0]
	}

	var owner user.ID
	user.IDFromKey(&owner, key.PrivateKey.PublicKey)

	layerCfg := &Config{
		Caches:      config,
		AnonKey:     AnonymousKey{Key: key},
		TreeService: NewTreeService(),
	}

	return &testContext{
		ctx:   ctx,
		layer: NewLayer(logger, tp, layerCfg),
		bktInfo: &data.BucketInfo{
			Name:  bktName,
			Owner: owner,
			CID:   bktID,
		},
		obj:       "obj1",
		t:         t,
		testNeoFS: tp,
	}
}

func TestSimpleVersioning(t *testing.T) {
	tc := prepareContext(t)
	err := tc.layer.PutBucketSettings(tc.ctx, &PutSettingsParams{
		BktInfo:  tc.bktInfo,
		Settings: &data.BucketSettings{Versioning: data.VersioningEnabled},
	})
	require.NoError(t, err)

	obj1Content1 := []byte("content obj1 v1")
	obj1v1 := tc.putObject(obj1Content1)

	obj1Content2 := []byte("content obj1 v2")
	obj1v2 := tc.putObject(obj1Content2)

	_, buffer2 := tc.getObject(tc.obj, "", false)
	require.Equal(t, obj1Content2, buffer2)

	_, buffer1 := tc.getObject(tc.obj, obj1v1.ID.EncodeToString(), false)
	require.Equal(t, obj1Content1, buffer1)

	tc.checkListObjects(obj1v2.ID)
}

func TestSimpleNoVersioning(t *testing.T) {
	tc := prepareContext(t)

	obj1Content1 := []byte("content obj1 v1")
	obj1v1 := tc.putObject(obj1Content1)

	obj1Content2 := []byte("content obj1 v2")
	obj1v2 := tc.putObject(obj1Content2)

	_, buffer2 := tc.getObject(tc.obj, "", false)
	require.Equal(t, obj1Content2, buffer2)

	tc.getObject(tc.obj, obj1v1.ID.EncodeToString(), true)
	tc.checkListObjects(obj1v2.ID)
}

func TestVersioningDeleteObject(t *testing.T) {
	tc := prepareContext(t)
	settings := &data.BucketSettings{Versioning: data.VersioningEnabled}
	err := tc.layer.PutBucketSettings(tc.ctx, &PutSettingsParams{
		BktInfo:  tc.bktInfo,
		Settings: settings,
	})
	require.NoError(t, err)

	tc.putObject([]byte("content obj1 v1"))
	tc.putObject([]byte("content obj1 v2"))

	tc.deleteObject(tc.obj, "", settings)
	tc.getObject(tc.obj, "", true)

	tc.checkListObjects()
}

func TestGetUnversioned(t *testing.T) {
	tc := prepareContext(t)

	objContent := []byte("content obj1 v1")
	objInfo := tc.putObject(objContent)

	settings := &data.BucketSettings{Versioning: data.VersioningUnversioned}
	err := tc.layer.PutBucketSettings(tc.ctx, &PutSettingsParams{
		BktInfo:  tc.bktInfo,
		Settings: settings,
	})
	require.NoError(t, err)

	resInfo, buffer := tc.getObject(tc.obj, data.UnversionedObjectVersionID, false)
	require.Equal(t, objContent, buffer)
	require.Equal(t, objInfo.VersionID(), resInfo.VersionID())
}

func TestVersioningDeleteSpecificObjectVersion(t *testing.T) {
	tc := prepareContext(t)
	settings := &data.BucketSettings{Versioning: data.VersioningEnabled}
	err := tc.layer.PutBucketSettings(tc.ctx, &PutSettingsParams{
		BktInfo:  tc.bktInfo,
		Settings: settings,
	})
	require.NoError(t, err)

	tc.putObject([]byte("content obj1 v1"))
	objV2Info := tc.putObject([]byte("content obj1 v2"))
	objV3Content := []byte("content obj1 v3")
	objV3Info := tc.putObject(objV3Content)

	tc.deleteObject(tc.obj, objV2Info.VersionID(), settings)
	tc.getObject(tc.obj, objV2Info.VersionID(), true)

	_, buffer3 := tc.getObject(tc.obj, "", false)
	require.Equal(t, objV3Content, buffer3)

	tc.deleteObject(tc.obj, "", settings)
	tc.getObject(tc.obj, "", true)

	versions := tc.listVersions()
	for _, ver := range versions.DeleteMarker {
		if ver.IsLatest {
			tc.deleteObject(tc.obj, ver.ObjectInfo.VersionID(), settings)
		}
	}

	resInfo, buffer := tc.getObject(tc.obj, "", false)
	require.Equal(t, objV3Content, buffer)
	require.Equal(t, objV3Info.VersionID(), resInfo.VersionID())
}

func TestNoVersioningDeleteObject(t *testing.T) {
	tc := prepareContext(t)

	tc.putObject([]byte("content obj1 v1"))
	tc.putObject([]byte("content obj1 v2"))

	settings, err := tc.layer.GetBucketSettings(tc.ctx, tc.bktInfo)
	require.NoError(t, err)

	tc.deleteObject(tc.obj, "", settings)
	tc.getObject(tc.obj, "", true)
	tc.checkListObjects()
}
