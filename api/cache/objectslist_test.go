package cache

import (
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	cidtest "github.com/TrueCloudLab/frostfs-sdk-go/container/id/test"
	oidtest "github.com/TrueCloudLab/frostfs-sdk-go/object/id/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testingCacheLifetime = 5 * time.Second
const testingCacheSize = 10

func getTestObjectsListConfig() *Config {
	return &Config{
		Size:     testingCacheSize,
		Lifetime: testingCacheLifetime,
		Logger:   zap.NewExample(),
	}
}

func TestObjectsListCache(t *testing.T) {
	var (
		listSize        = 10
		versions        []*data.NodeVersion
		cidKey, cidKey2 = cidtest.ID(), cidtest.ID()
	)

	for i := 0; i < listSize; i++ {
		versions = append(versions, &data.NodeVersion{BaseNodeVersion: data.BaseNodeVersion{OID: oidtest.ID()}})
	}

	t.Run("lifetime", func(t *testing.T) {
		var (
			config  = getTestObjectsListConfig()
			cache   = NewObjectsListCache(config)
			listKey = ObjectsListKey{cid: cidKey}
		)

		err := cache.PutVersions(listKey, versions)
		require.NoError(t, err)

		condition := func() bool {
			return cache.GetVersions(listKey) == nil
		}

		require.Never(t, condition, config.Lifetime, time.Second)
		require.Eventually(t, condition, time.Second, 10*time.Millisecond)
	})

	t.Run("get cache with empty prefix", func(t *testing.T) {
		var (
			cache   = NewObjectsListCache(getTestObjectsListConfig())
			listKey = ObjectsListKey{cid: cidKey}
		)
		err := cache.PutVersions(listKey, versions)
		require.NoError(t, err)

		actual := cache.GetVersions(listKey)

		require.Equal(t, len(versions), len(actual))
		for i := range versions {
			require.Equal(t, versions[i], actual[i])
		}
	})

	t.Run("get cache with prefix", func(t *testing.T) {
		listKey := ObjectsListKey{
			cid:    cidKey,
			prefix: "dir",
		}

		cache := NewObjectsListCache(getTestObjectsListConfig())
		err := cache.PutVersions(listKey, versions)
		require.NoError(t, err)

		actual := cache.GetVersions(listKey)

		require.Equal(t, len(versions), len(actual))
		for i := range versions {
			require.Equal(t, versions[i], actual[i])
		}
	})

	t.Run("get cache with other prefix", func(t *testing.T) {
		var (
			listKey = ObjectsListKey{
				cid:    cidKey,
				prefix: "dir",
			}

			newKey = ObjectsListKey{
				cid:    cidKey,
				prefix: "obj",
			}
		)

		cache := NewObjectsListCache(getTestObjectsListConfig())
		err := cache.PutVersions(listKey, versions)
		require.NoError(t, err)

		actual := cache.GetVersions(newKey)
		require.Nil(t, actual)
	})

	t.Run("get cache with non-existing key", func(t *testing.T) {
		var (
			listKey = ObjectsListKey{
				cid: cidKey,
			}
			newKey = ObjectsListKey{
				cid: cidKey2,
			}
		)

		cache := NewObjectsListCache(getTestObjectsListConfig())
		err := cache.PutVersions(listKey, versions)
		require.NoError(t, err)

		actual := cache.GetVersions(newKey)
		require.Nil(t, actual)
	})
}

func TestCleanCacheEntriesChangedWithPutObject(t *testing.T) {
	var (
		id       = cidtest.ID()
		versions = []*data.NodeVersion{{BaseNodeVersion: data.BaseNodeVersion{OID: oidtest.ID()}}}
		keys     []ObjectsListKey
	)

	for _, p := range []string{"", "dir/", "dir/lol/"} {
		keys = append(keys, ObjectsListKey{cid: id, prefix: p})
	}

	t.Run("put object to the root of the bucket", func(t *testing.T) {
		config := getTestObjectsListConfig()
		config.Lifetime = time.Minute
		cache := NewObjectsListCache(config)
		for _, k := range keys {
			err := cache.PutVersions(k, versions)
			require.NoError(t, err)
		}
		cache.CleanCacheEntriesContainingObject("obj1", id)
		for _, k := range keys {
			list := cache.GetVersions(k)
			if k.prefix == "" {
				require.Nil(t, list)
			} else {
				require.NotNil(t, list)
			}
		}
	})

	t.Run("put object to dir/", func(t *testing.T) {
		config := getTestObjectsListConfig()
		config.Lifetime = time.Minute
		cache := NewObjectsListCache(config)
		for _, k := range keys {
			err := cache.PutVersions(k, versions)
			require.NoError(t, err)
		}
		cache.CleanCacheEntriesContainingObject("dir/obj", id)
		for _, k := range keys {
			list := cache.GetVersions(k)
			if k.prefix == "" || k.prefix == "dir/" {
				require.Nil(t, list)
			} else {
				require.NotNil(t, list)
			}
		}
	})

	t.Run("put object to dir/lol/", func(t *testing.T) {
		config := getTestObjectsListConfig()
		config.Lifetime = time.Minute
		cache := NewObjectsListCache(config)
		for _, k := range keys {
			err := cache.PutVersions(k, versions)
			require.NoError(t, err)
		}
		cache.CleanCacheEntriesContainingObject("dir/lol/obj", id)
		for _, k := range keys {
			list := cache.GetVersions(k)
			require.Nil(t, list)
		}
	})
}
