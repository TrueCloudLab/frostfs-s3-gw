package metrics

import (
	"github.com/TrueCloudLab/frostfs-s3-gw/internal/version"
	"github.com/prometheus/client_golang/prometheus"
)

type stats struct {
	desc *prometheus.Desc
}

var (
	versionInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "frostfs_s3",
			Name:      "version_info",
			Help:      "Version of current FrostFS S3 Gate instance",
		},
		[]string{
			// current version
			"version",
		},
	)

	statsMetrics = &stats{
		desc: prometheus.NewDesc("frostfs_s3_stats", "Statistics exposed by FrostFS S3 Gate instance", nil, nil),
	}
)

func init() {
	prometheus.MustRegister(versionInfo)
	prometheus.MustRegister(statsMetrics)
	prometheus.MustRegister(httpRequestsDuration)
}

func collectNetworkMetrics(ch chan<- prometheus.Metric) {
	// Network Sent/Received Bytes (Outbound)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName("frostfs_s3", "tx", "bytes_total"),
			"Total number of bytes sent by current FrostFS S3 Gate instance",
			nil, nil),
		prometheus.CounterValue,
		float64(httpStatsMetric.getInputBytes()),
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName("frostfs_s3", "rx", "bytes_total"),
			"Total number of bytes received by current FrostFS S3 Gate instance",
			nil, nil),
		prometheus.CounterValue,
		float64(httpStatsMetric.getOutputBytes()),
	)
}

func (s *stats) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.desc
}

func (s *stats) Collect(ch chan<- prometheus.Metric) {
	// Expose current version information
	versionInfo.WithLabelValues(version.Version).Set(1.0)

	// connect collectors
	collectHTTPMetrics(ch)
	collectNetworkMetrics(ch)
}
