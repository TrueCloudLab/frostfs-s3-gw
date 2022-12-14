package cache

import (
	"fmt"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

// BucketCache contains cache with objects and the lifetime of cache entries.
type BucketCache struct {
	cache  gcache.Cache
	logger *zap.Logger
}

const (
	// DefaultBucketCacheSize is a default maximum number of entries in cache.
	DefaultBucketCacheSize = 1e3
	// DefaultBucketCacheLifetime is a default lifetime of entries in cache.
	DefaultBucketCacheLifetime = time.Minute
)

// DefaultBucketConfig returns new default cache expiration values.
func DefaultBucketConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultBucketCacheSize,
		Lifetime: DefaultBucketCacheLifetime,
		Logger:   logger,
	}
}

// NewBucketCache creates an object of BucketCache.
func NewBucketCache(config *Config) *BucketCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &BucketCache{cache: gc, logger: config.Logger}
}

// Get returns a cached object.
func (o *BucketCache) Get(key string) *data.BucketInfo {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.BucketInfo)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// Put puts an object to cache.
func (o *BucketCache) Put(bkt *data.BucketInfo) error {
	return o.cache.Set(bkt.Name, bkt)
}

// Delete deletes an object from cache.
func (o *BucketCache) Delete(key string) bool {
	return o.cache.Remove(key)
}
