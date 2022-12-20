package cache

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

/*
	This is an implementation of cache which keeps unsorted lists of objects' IDs (all versions)
	for a specified bucket and a prefix.

	The cache contains gcache whose entries have a key: ObjectsListKey struct and a value: list of ids.
	After putting a record, it lives for a while (default value is 60 seconds).

	When we receive a request from a user, we try to find the suitable and non-expired cache entry, go through the list
	and get ObjectInfos from common object cache or with a request to FrostFS.

	When we put an object into a container, we invalidate entries with prefixes that are prefixes of the object's name.
*/

type (
	// ObjectsListCache contains cache for ListObjects and ListObjectVersions.
	ObjectsListCache struct {
		cache  gcache.Cache
		logger *zap.Logger
	}

	// ObjectsListKey is a key to find a ObjectsListCache's entry.
	ObjectsListKey struct {
		cid        cid.ID
		prefix     string
		latestOnly bool
	}
)

const (
	// DefaultObjectsListCacheLifetime is a default lifetime of entries in cache of ListObjects.
	DefaultObjectsListCacheLifetime = time.Second * 60
	// DefaultObjectsListCacheSize is a default size of cache of ListObjects.
	DefaultObjectsListCacheSize = 1e5
)

// DefaultObjectsListConfig returns new default cache expiration values.
func DefaultObjectsListConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultObjectsListCacheSize,
		Lifetime: DefaultObjectsListCacheLifetime,
		Logger:   logger,
	}
}

func (k *ObjectsListKey) String() string {
	return k.cid.EncodeToString() + k.prefix + strconv.FormatBool(k.latestOnly)
}

// NewObjectsListCache is a constructor which creates an object of ListObjectsCache with the given lifetime of entries.
func NewObjectsListCache(config *Config) *ObjectsListCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &ObjectsListCache{cache: gc, logger: config.Logger}
}

// GetVersions returns a list of ObjectInfo.
func (l *ObjectsListCache) GetVersions(key ObjectsListKey) []*data.NodeVersion {
	entry, err := l.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.([]*data.NodeVersion)
	if !ok {
		l.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// PutVersions puts a list of object versions to cache.
func (l *ObjectsListCache) PutVersions(key ObjectsListKey, versions []*data.NodeVersion) error {
	return l.cache.Set(key, versions)
}

// CleanCacheEntriesContainingObject deletes entries containing specified object.
func (l *ObjectsListCache) CleanCacheEntriesContainingObject(objectName string, cnr cid.ID) {
	keys := l.cache.Keys(true)
	for _, key := range keys {
		k, ok := key.(ObjectsListKey)
		if !ok {
			l.logger.Warn("invalid cache key type", zap.String("actual", fmt.Sprintf("%T", key)),
				zap.String("expected", fmt.Sprintf("%T", k)))
			continue
		}
		if cnr.Equals(k.cid) && strings.HasPrefix(objectName, k.prefix) {
			l.cache.Remove(k)
		}
	}
}

// CreateObjectsListCacheKey returns ObjectsListKey with the given CID, prefix and latestOnly flag.
func CreateObjectsListCacheKey(cnr cid.ID, prefix string, latestOnly bool) ObjectsListKey {
	p := ObjectsListKey{
		cid:        cnr,
		prefix:     prefix,
		latestOnly: latestOnly,
	}

	return p
}
