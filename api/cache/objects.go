package cache

import (
	"fmt"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

// ObjectsCache provides lru cache for objects.
type ObjectsCache struct {
	cache  gcache.Cache
	logger *zap.Logger
}

const (
	// DefaultObjectsCacheLifetime is a default lifetime of entries in objects' cache.
	DefaultObjectsCacheLifetime = time.Minute * 5
	// DefaultObjectsCacheSize is a default maximum number of entries in objects' cache.
	DefaultObjectsCacheSize = 1e6
)

// DefaultObjectsConfig returns new default cache expiration values.
func DefaultObjectsConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultObjectsCacheSize,
		Lifetime: DefaultObjectsCacheLifetime,
		Logger:   logger,
	}
}

// New creates an object of ObjectHeadersCache.
func New(config *Config) *ObjectsCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &ObjectsCache{cache: gc, logger: config.Logger}
}

// GetObject returns a cached object info.
func (o *ObjectsCache) GetObject(address oid.Address) *data.ExtendedObjectInfo {
	entry, err := o.cache.Get(address)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.ExtendedObjectInfo)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// PutObject puts an object info to cache.
func (o *ObjectsCache) PutObject(obj *data.ExtendedObjectInfo) error {
	return o.cache.Set(obj.ObjectInfo.Address(), obj)
}

// Delete deletes an object from cache.
func (o *ObjectsCache) Delete(address oid.Address) bool {
	return o.cache.Remove(address)
}
