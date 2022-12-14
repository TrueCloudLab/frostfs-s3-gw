package cache

import (
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	cidtest "github.com/TrueCloudLab/frostfs-sdk-go/container/id/test"
	oidtest "github.com/TrueCloudLab/frostfs-sdk-go/object/id/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestAccessBoxCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewAccessBoxCache(DefaultAccessBoxConfig(logger))

	addr := oidtest.Address()
	box := &accessbox.Box{}

	err := cache.Put(addr, box)
	require.NoError(t, err)
	val := cache.Get(addr)
	require.Equal(t, box, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(addr, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.Get(addr), observedLog)
}

func TestBucketsCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewBucketCache(DefaultBucketConfig(logger))

	bktInfo := &data.BucketInfo{Name: "bucket"}

	err := cache.Put(bktInfo)
	require.NoError(t, err)
	val := cache.Get(bktInfo.Name)
	require.Equal(t, bktInfo, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(bktInfo.Name, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.Get(bktInfo.Name), observedLog)
}

func TestObjectNamesCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewObjectsNameCache(DefaultObjectsNameConfig(logger))

	key := "name"
	addr := oidtest.Address()

	err := cache.Put(key, addr)
	require.NoError(t, err)
	val := cache.Get(key)
	require.Equal(t, addr, *val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.Get(key), observedLog)
}

func TestObjectCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := New(DefaultObjectsConfig(logger))

	addr := oidtest.Address()

	extObjInfo := &data.ExtendedObjectInfo{
		ObjectInfo: &data.ObjectInfo{
			ID:  addr.Object(),
			CID: addr.Container(),
		},
		NodeVersion: &data.NodeVersion{
			BaseNodeVersion: data.BaseNodeVersion{
				FilePath: "obj",
				Size:     50,
			},
			IsUnversioned: true,
		},
		IsLatest: true,
	}

	err := cache.PutObject(extObjInfo)
	require.NoError(t, err)
	val := cache.GetObject(addr)
	require.Equal(t, extObjInfo, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(addr, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetObject(addr), observedLog)
}

func TestObjectsListCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewObjectsListCache(DefaultObjectsListConfig(logger))

	cnrID := cidtest.ID()
	key := ObjectsListKey{cid: cnrID, prefix: "obj"}
	versions := []*data.NodeVersion{{BaseNodeVersion: data.BaseNodeVersion{OID: oidtest.ID()}}}

	err := cache.PutVersions(key, versions)
	require.NoError(t, err)
	val := cache.GetVersions(key)
	require.Equal(t, versions, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetVersions(key), observedLog)

	err = cache.cache.Set("key", "tmp")
	require.NoError(t, err)
	cache.CleanCacheEntriesContainingObject(key.prefix, cnrID)
	require.Equal(t, 2, observedLog.Len())
	require.Equal(t, observedLog.All()[1].Message, "invalid cache key type")
}

func TestObjectInfoCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewSystemCache(DefaultSystemConfig(logger))

	key := "key"
	objInfo := &data.ObjectInfo{Name: key}

	err := cache.PutObject(key, objInfo)
	require.NoError(t, err)
	val := cache.GetObject(key)
	require.Equal(t, objInfo, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetObject(key), observedLog)
}

func TestCORsCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewSystemCache(DefaultSystemConfig(logger))

	key := "key"
	cors := &data.CORSConfiguration{}

	err := cache.PutCORS(key, cors)
	require.NoError(t, err)
	val := cache.GetCORS(key)
	require.Equal(t, cors, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetCORS(key), observedLog)
}

func TestSettingsCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewSystemCache(DefaultSystemConfig(logger))

	key := "key"
	settings := &data.BucketSettings{Versioning: data.VersioningEnabled}

	err := cache.PutSettings(key, settings)
	require.NoError(t, err)
	val := cache.GetSettings(key)
	require.Equal(t, settings, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetSettings(key), observedLog)
}

func TestNotificationConfigurationCacheType(t *testing.T) {
	logger, observedLog := getObservedLogger()
	cache := NewSystemCache(DefaultSystemConfig(logger))

	key := "key"
	notificationConfig := &data.NotificationConfiguration{}

	err := cache.PutNotificationConfiguration(key, notificationConfig)
	require.NoError(t, err)
	val := cache.GetNotificationConfiguration(key)
	require.Equal(t, notificationConfig, val)
	require.Equal(t, 0, observedLog.Len())

	err = cache.cache.Set(key, "tmp")
	require.NoError(t, err)
	assertInvalidCacheEntry(t, cache.GetNotificationConfiguration(key), observedLog)
}

func assertInvalidCacheEntry(t *testing.T, val interface{}, observedLog *observer.ObservedLogs) {
	require.Nil(t, val)
	require.Equal(t, 1, observedLog.Len())
	require.Equal(t, observedLog.All()[0].Message, "invalid cache entry type")
}

func getObservedLogger() (*zap.Logger, *observer.ObservedLogs) {
	loggerCore, observedLog := observer.New(zap.WarnLevel)
	return zap.New(loggerCore), observedLog
}
