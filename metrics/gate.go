package metrics

import (
	"net/http"

	"github.com/TrueCloudLab/frostfs-sdk-go/pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "frostfs_s3_gw"

type StatisticScraper interface {
	Statistic() pool.Statistic
}

type GateMetrics struct {
	State   stateMetrics
	Pool    poolMetricsCollector
	Billing *billingMetrics
}

func NewGateMetrics(scraper StatisticScraper) *GateMetrics {
	stateMetric := newStateMetrics()
	stateMetric.register()

	poolMetric := newPoolMetricsCollector(scraper)
	poolMetric.register()

	billingMetric := newBillingMetrics()
	billingMetric.register()

	return &GateMetrics{
		State:   *stateMetric,
		Pool:    *poolMetric,
		Billing: billingMetric,
	}
}

func (g *GateMetrics) Unregister() {
	g.State.unregister()
	prometheus.Unregister(&g.Pool)
	g.Billing.unregister()
}

func (g *GateMetrics) Handler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/", promhttp.Handler())
	handler.Handle("/metrics/billing", promhttp.HandlerFor(g.Billing.registry, promhttp.HandlerOpts{}))
	return handler
}
