package cache

import (
	"fmt"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/bluele/gcache"
	"go.uber.org/zap"
)

type (
	// AccessBoxCache stores an access box by its address.
	AccessBoxCache struct {
		logger *zap.Logger
		cache  gcache.Cache
	}

	// Config stores expiration params for cache.
	Config struct {
		Size     int
		Lifetime time.Duration
		Logger   *zap.Logger
	}
)

const (
	// DefaultAccessBoxCacheSize is a default maximum number of entries in cache.
	DefaultAccessBoxCacheSize = 100
	// DefaultAccessBoxCacheLifetime is a default lifetime of entries in cache.
	DefaultAccessBoxCacheLifetime = 10 * time.Minute
)

// DefaultAccessBoxConfig returns new default cache expiration values.
func DefaultAccessBoxConfig(logger *zap.Logger) *Config {
	return &Config{
		Size:     DefaultAccessBoxCacheSize,
		Lifetime: DefaultAccessBoxCacheLifetime,
		Logger:   logger,
	}
}

// NewAccessBoxCache creates an object of BucketCache.
func NewAccessBoxCache(config *Config) *AccessBoxCache {
	gc := gcache.New(config.Size).LRU().Expiration(config.Lifetime).Build()

	return &AccessBoxCache{cache: gc, logger: config.Logger}
}

// Get returns a cached object.
func (o *AccessBoxCache) Get(address oid.Address) *accessbox.Box {
	entry, err := o.cache.Get(address)
	if err != nil {
		return nil
	}

	result, ok := entry.(*accessbox.Box)
	if !ok {
		o.logger.Warn("invalid cache entry type", zap.String("actual", fmt.Sprintf("%T", entry)),
			zap.String("expected", fmt.Sprintf("%T", result)))
		return nil
	}

	return result
}

// Put stores an object to cache.
func (o *AccessBoxCache) Put(address oid.Address, box *accessbox.Box) error {
	return o.cache.Set(address, box)
}
