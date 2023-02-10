package handler

import (
	"errors"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/TrueCloudLab/frostfs-sdk-go/netmap"
	"go.uber.org/zap"
)

type (
	handler struct {
		log         *zap.Logger
		obj         layer.Client
		notificator Notificator
		cfg         *Config
	}

	Notificator interface {
		SendNotifications(topics map[string]string, p *SendNotificationParams) error
		SendTestNotification(topic, bucketName, requestID, HostID string, now time.Time) error
	}

	// Config contains data which handler needs to keep.
	Config struct {
		Policy             PlacementPolicy
		DefaultMaxAge      int
		NotificatorEnabled bool
		CopiesNumber       uint32
		ResolveZoneList    []string
		IsResolveListAllow bool // True if ResolveZoneList contains allowed zones
	}

	PlacementPolicy interface {
		Default() netmap.PlacementPolicy
		Get(string) (netmap.PlacementPolicy, bool)
	}
)

const (
	// DefaultPolicy is a default policy of placing containers in FrostFS if it's not set at the request.
	DefaultPolicy = "REP 3"
	// DefaultCopiesNumber is a default number of object copies that is enough to consider put successful if it's not set in config.
	DefaultCopiesNumber uint32 = 0
)

var _ api.Handler = (*handler)(nil)

// New creates new api.Handler using given logger and client.
func New(log *zap.Logger, obj layer.Client, notificator Notificator, cfg *Config) (api.Handler, error) {
	switch {
	case obj == nil:
		return nil, errors.New("empty FrostFS Object Layer")
	case log == nil:
		return nil, errors.New("empty logger")
	}

	if !cfg.NotificatorEnabled {
		log.Warn("notificator is disabled, s3 won't produce notification events")
	} else if notificator == nil {
		return nil, errors.New("empty notificator")
	}

	return &handler{
		log:         log,
		obj:         obj,
		cfg:         cfg,
		notificator: notificator,
	}, nil
}
