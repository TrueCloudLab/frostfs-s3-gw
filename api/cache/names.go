package cache

import (
	"fmt"
	"time"

	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

// ObjectsNameCache provides lru cache for objects.
// This cache contains mapping nice names to object addresses.
// Key is bucketName+objectName.
type ObjectsNameCache struct {
	cache  gcache.Cache
	logger *zap.Logger
}

const (
	// DefaultObjectsNameCacheSize is a default maximum number of entries in cache.
	DefaultObjectsNameCacheSize = 1e4
	// DefaultObjectsNameCacheLifetime is a default lifetime of entries in cache.
	DefaultObjectsNameCacheLifetime = time.Minute
)

// DefaultObjectsNameConfig returns new default cache expiration values.
func DefaultObjectsNameConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultObjectsNameCacheSize,
		Lifetime: DefaultObjectsNameCacheLifetime,
		Logger:   logger,
	}
}

// NewObjectsNameCache creates an object of ObjectsNameCache.
func NewObjectsNameCache(config *Config) *ObjectsNameCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &ObjectsNameCache{cache: gc, logger: config.Logger}
}

// Get returns a cached object. Returns nil if value is missing.
func (o *ObjectsNameCache) Get(key string) *oid.Address {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(oid.Address)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return &result
}

// Put puts an object to cache.
func (o *ObjectsNameCache) Put(key string, address oid.Address) error {
	return o.cache.Set(key, address)
}

// Delete deletes an object from cache.
func (o *ObjectsNameCache) Delete(key string) bool {
	return o.cache.Remove(key)
}
