package cache

import (
	"fmt"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

// SystemCache provides lru cache for objects.
// This cache contains "system" objects (bucket versioning settings, tagging object etc.).
// Key is bucketName+systemFilePath.
type SystemCache struct {
	cache  gcache.Cache
	logger *zap.Logger
}

const (
	// DefaultSystemCacheSize is a default maximum number of entries in cache.
	DefaultSystemCacheSize = 1e4
	// DefaultSystemCacheLifetime is a default lifetime of entries in cache.
	DefaultSystemCacheLifetime = 5 * time.Minute
)

// DefaultSystemConfig returns new default cache expiration values.
func DefaultSystemConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultSystemCacheSize,
		Lifetime: DefaultSystemCacheLifetime,
		Logger:   logger,
	}
}

// NewSystemCache creates an object of SystemCache.
func NewSystemCache(config *Config) *SystemCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &SystemCache{cache: gc, logger: config.Logger}
}

// GetObject returns a cached object.
func (o *SystemCache) GetObject(key string) *data.ObjectInfo {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.ObjectInfo)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// GetLockInfo returns a cached object.
func (o *SystemCache) GetLockInfo(key string) *data.LockInfo {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.LockInfo)
	if !ok {
		return nil
	}

	return result
}

func (o *SystemCache) GetCORS(key string) *data.CORSConfiguration {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.CORSConfiguration)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

func (o *SystemCache) GetSettings(key string) *data.BucketSettings {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.BucketSettings)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

func (o *SystemCache) GetNotificationConfiguration(key string) *data.NotificationConfiguration {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(*data.NotificationConfiguration)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// GetTagging returns tags of a bucket or an object.
func (o *SystemCache) GetTagging(key string) map[string]string {
	entry, err := o.cache.Get(key)
	if err != nil {
		return nil
	}

	result, ok := entry.(map[string]string)
	if !ok {
		return nil
	}

	return result
}

// PutObject puts an object to cache.
func (o *SystemCache) PutObject(key string, obj *data.ObjectInfo) error {
	return o.cache.Set(key, obj)
}

// PutLockInfo puts an object to cache.
func (o *SystemCache) PutLockInfo(key string, lockInfo *data.LockInfo) error {
	return o.cache.Set(key, lockInfo)
}

func (o *SystemCache) PutCORS(key string, obj *data.CORSConfiguration) error {
	return o.cache.Set(key, obj)
}

func (o *SystemCache) PutSettings(key string, settings *data.BucketSettings) error {
	return o.cache.Set(key, settings)
}

func (o *SystemCache) PutNotificationConfiguration(key string, obj *data.NotificationConfiguration) error {
	return o.cache.Set(key, obj)
}

// PutTagging puts tags of a bucket or an object.
func (o *SystemCache) PutTagging(key string, tagSet map[string]string) error {
	return o.cache.Set(key, tagSet)
}

// Delete deletes an object from cache.
func (o *SystemCache) Delete(key string) bool {
	return o.cache.Remove(key)
}
