package metrics

import (
	"net/http"
	"sync"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/internal/frostfs"
	"go.uber.org/zap"
)

type AppMetrics struct {
	logger  *zap.Logger
	gate    *GateMetrics
	mu      sync.RWMutex
	enabled bool
}

func NewAppMetrics(logger *zap.Logger, poolStatistics *frostfs.PoolStatistic, enabled bool) *AppMetrics {
	if !enabled {
		logger.Warn("metrics are disabled")
	}
	return &AppMetrics{
		logger:  logger,
		gate:    NewGateMetrics(poolStatistics),
		enabled: enabled,
	}
}

func (m *AppMetrics) SetEnabled(enabled bool) {
	if !enabled {
		m.logger.Warn("metrics are disabled")
	}

	m.mu.Lock()
	m.enabled = enabled
	m.mu.Unlock()
}

func (m *AppMetrics) SetHealth(status int32) {
	if !m.isEnabled() {
		return
	}

	m.gate.State.SetHealth(status)
}

func (m *AppMetrics) Shutdown() {
	m.mu.Lock()
	if m.enabled {
		m.gate.State.SetHealth(0)
		m.enabled = false
	}
	m.gate.Unregister()
	m.mu.Unlock()
}

func (m *AppMetrics) isEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

func (m *AppMetrics) Handler() http.Handler {
	return m.gate.Handler()
}

func (m *AppMetrics) Update(user, bucket, cnrID string, reqType api.RequestType, in, out uint64) {
	if !m.isEnabled() {
		return
	}

	m.gate.Billing.apiStat.Update(user, bucket, cnrID, reqType, in, out)
}
