package cache

import (
	"fmt"
	"time"

	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

// AccessControlCache provides lru cache for objects.
type AccessControlCache struct {
	cache  gcache.Cache
	logger *zap.Logger
}

const (
	// DefaultAccessControlCacheLifetime is a default lifetime of entries in access' cache.
	DefaultAccessControlCacheLifetime = 1 * time.Minute
	// DefaultAccessControlCacheSize is a default maximum number of entries in access' cache.
	DefaultAccessControlCacheSize = 1e5
)

// DefaultAccessControlConfig returns new default cache expiration values.
func DefaultAccessControlConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultAccessControlCacheSize,
		Lifetime: DefaultAccessControlCacheLifetime,
		Logger:   logger,
	}
}

// NewAccessControlCache creates an object of AccessControlCache.
func NewAccessControlCache(config *Config) *AccessControlCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()
	return &AccessControlCache{cache: gc, logger: config.Logger}
}

// Get returns true if such key exists.
func (o *AccessControlCache) Get(owner user.ID, key string) bool {
	entry, err := o.cache.Get(cacheKey(owner, key))
	if err != nil {
		return false
	}

	result, ok := entry.(bool)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return false
	}

	return result
}

// Put puts an item to cache.
func (o *AccessControlCache) Put(owner user.ID, key string) error {
	return o.cache.Set(cacheKey(owner, key), true)
}

// Delete deletes an object from cache.
func (o *AccessControlCache) Delete(owner user.ID, key string) bool {
	return o.cache.Remove(cacheKey(owner, key))
}

func cacheKey(owner user.ID, key string) string {
	return owner.EncodeToString() + key
}
