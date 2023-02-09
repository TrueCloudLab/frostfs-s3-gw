package metrics

import (
	"github.com/TrueCloudLab/frostfs-sdk-go/pool"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	poolSubsystem = "pool"

	methodGetBalance       = "get_balance"
	methodPutContainer     = "put_container"
	methodGetContainer     = "get_container"
	methodListContainer    = "list_container"
	methodDeleteContainer  = "delete_container"
	methodGetContainerEacl = "get_container_eacl"
	methodSetContainerEacl = "set_container_eacl"
	methodEndpointInfo     = "endpoint_info"
	methodNetworkInfo      = "network_info"
	methodPutObject        = "put_object"
	methodDeleteObject     = "delete_object"
	methodGetObject        = "get_object"
	methodHeadObject       = "head_object"
	methodRangeObject      = "range_object"
	methodCreateSession    = "create_session"
)

type poolMetricsCollector struct {
	poolStatScraper     StatisticScraper
	overallErrors       prometheus.Gauge
	overallNodeErrors   *prometheus.GaugeVec
	overallNodeRequests *prometheus.GaugeVec
	currentErrors       *prometheus.GaugeVec
	requestDuration     *prometheus.GaugeVec
}

func newPoolMetricsCollector(scraper StatisticScraper) *poolMetricsCollector {
	overallErrors := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: poolSubsystem,
			Name:      "overall_errors",
			Help:      "Total number of errors in pool",
		},
	)

	overallNodeErrors := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: poolSubsystem,
			Name:      "overall_node_errors",
			Help:      "Total number of errors for connection in pool",
		},
		[]string{
			"node",
		},
	)

	overallNodeRequests := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: poolSubsystem,
			Name:      "overall_node_requests",
			Help:      "Total number of requests to specific node in pool",
		},
		[]string{
			"node",
		},
	)

	currentErrors := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: poolSubsystem,
			Name:      "current_errors",
			Help:      "Number of errors on current connections that will be reset after the threshold",
		},
		[]string{
			"node",
		},
	)

	requestsDuration := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: poolSubsystem,
			Name:      "avg_request_duration",
			Help:      "Average request duration (in milliseconds) for specific method on node in pool",
		},
		[]string{
			"node",
			"method",
		},
	)

	return &poolMetricsCollector{
		poolStatScraper:     scraper,
		overallErrors:       overallErrors,
		overallNodeErrors:   overallNodeErrors,
		overallNodeRequests: overallNodeRequests,
		currentErrors:       currentErrors,
		requestDuration:     requestsDuration,
	}
}

func (m *poolMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	m.updateStatistic()
	m.overallErrors.Collect(ch)
	m.overallNodeErrors.Collect(ch)
	m.overallNodeRequests.Collect(ch)
	m.currentErrors.Collect(ch)
	m.requestDuration.Collect(ch)
}

func (m *poolMetricsCollector) Describe(descs chan<- *prometheus.Desc) {
	m.overallErrors.Describe(descs)
	m.overallNodeErrors.Describe(descs)
	m.overallNodeRequests.Describe(descs)
	m.currentErrors.Describe(descs)
	m.requestDuration.Describe(descs)
}

func (m *poolMetricsCollector) register() {
	prometheus.MustRegister(m)
}

func (m *poolMetricsCollector) updateStatistic() {
	stat := m.poolStatScraper.Statistic()

	m.overallNodeErrors.Reset()
	m.overallNodeRequests.Reset()
	m.currentErrors.Reset()
	m.requestDuration.Reset()

	for _, node := range stat.Nodes() {
		m.overallNodeErrors.WithLabelValues(node.Address()).Set(float64(node.OverallErrors()))
		m.overallNodeRequests.WithLabelValues(node.Address()).Set(float64(node.Requests()))

		m.currentErrors.WithLabelValues(node.Address()).Set(float64(node.CurrentErrors()))
		m.updateRequestsDuration(node)
	}

	m.overallErrors.Set(float64(stat.OverallErrors()))
}

func (m *poolMetricsCollector) updateRequestsDuration(node pool.NodeStatistic) {
	m.requestDuration.WithLabelValues(node.Address(), methodGetBalance).Set(float64(node.AverageGetBalance().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodPutContainer).Set(float64(node.AveragePutContainer().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodGetContainer).Set(float64(node.AverageGetContainer().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodListContainer).Set(float64(node.AverageListContainer().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodDeleteContainer).Set(float64(node.AverageDeleteContainer().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodGetContainerEacl).Set(float64(node.AverageGetContainerEACL().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodSetContainerEacl).Set(float64(node.AverageSetContainerEACL().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodEndpointInfo).Set(float64(node.AverageEndpointInfo().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodNetworkInfo).Set(float64(node.AverageNetworkInfo().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodPutObject).Set(float64(node.AveragePutObject().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodDeleteObject).Set(float64(node.AverageDeleteObject().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodGetObject).Set(float64(node.AverageGetObject().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodHeadObject).Set(float64(node.AverageHeadObject().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodRangeObject).Set(float64(node.AverageRangeObject().Milliseconds()))
	m.requestDuration.WithLabelValues(node.Address(), methodCreateSession).Set(float64(node.AverageCreateSession().Milliseconds()))
}
