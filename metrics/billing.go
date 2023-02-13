package metrics

import (
	"sync"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/prometheus/client_golang/prometheus"
)

const billingSubsystem = "billing"

type TrafficType int

const (
	UnknownTraffic TrafficType = iota
	INTraffic      TrafficType = iota
	OUTTraffic     TrafficType = iota
)

func (t TrafficType) String() string {
	switch t {
	case 1:
		return "IN"
	case 2:
		return "OUT"
	default:
		return "Unknown"
	}
}

type (
	OperationList [6]int

	UsersAPIStats struct {
		users map[string]*userAPIStats
		sync.RWMutex
	}

	bucketKey struct {
		name string
		cid  string
	}

	bucketStat struct {
		Operations OperationList
		InTraffic  uint64
		OutTraffic uint64
	}

	userAPIStats struct {
		buckets map[bucketKey]bucketStat
		user    string
	}

	UserBucketInfo struct {
		User        string
		Bucket      string
		ContainerID string
	}

	UserMetricsInfo struct {
		UserBucketInfo
		Operation api.RequestType
		Requests  int
	}

	UserTrafficMetricsInfo struct {
		UserBucketInfo
		Type  TrafficType
		Value uint64
	}

	UserMetrics struct {
		Requests []UserMetricsInfo
		Traffic  []UserTrafficMetricsInfo
	}
)

func (u *UsersAPIStats) Update(user, bucket, cnrID string, reqType api.RequestType, in, out uint64) {
	u.Lock()
	defer u.Unlock()

	usersStat := u.users[user]
	if usersStat == nil {
		if u.users == nil {
			u.users = make(map[string]*userAPIStats)
		}
		usersStat = &userAPIStats{
			buckets: make(map[bucketKey]bucketStat, 1),
			user:    user,
		}
		u.users[user] = usersStat
	}

	key := bucketKey{
		name: bucket,
		cid:  cnrID,
	}

	bktStat := usersStat.buckets[key]
	bktStat.Operations[reqType]++
	bktStat.InTraffic += in
	bktStat.OutTraffic += out
	usersStat.buckets[key] = bktStat
}

func (u *UsersAPIStats) DumpMetrics() UserMetrics {
	u.Lock()
	defer u.Unlock()

	result := UserMetrics{
		Requests: make([]UserMetricsInfo, 0, len(u.users)),
		Traffic:  make([]UserTrafficMetricsInfo, 0, len(u.users)),
	}

	for user, userStat := range u.users {
		for key, bktStat := range userStat.buckets {
			userBktInfo := UserBucketInfo{
				User:        user,
				Bucket:      key.name,
				ContainerID: key.cid,
			}

			if bktStat.InTraffic != 0 {
				result.Traffic = append(result.Traffic, UserTrafficMetricsInfo{
					UserBucketInfo: userBktInfo,
					Type:           INTraffic,
					Value:          bktStat.InTraffic,
				})
			}

			if bktStat.OutTraffic != 0 {
				result.Traffic = append(result.Traffic, UserTrafficMetricsInfo{
					UserBucketInfo: userBktInfo,
					Type:           OUTTraffic,
					Value:          bktStat.OutTraffic,
				})
			}

			for op, val := range bktStat.Operations {
				if val != 0 {
					result.Requests = append(result.Requests, UserMetricsInfo{
						UserBucketInfo: userBktInfo,
						Operation:      api.RequestType(op),
						Requests:       val,
					})
				}
			}
		}
	}

	u.users = make(map[string]*userAPIStats)

	return result
}

type billingMetrics struct {
	registry *prometheus.Registry

	desc    *prometheus.Desc
	apiStat UsersAPIStats
}

func newBillingMetrics() *billingMetrics {
	return &billingMetrics{
		registry: prometheus.NewRegistry(),
		desc:     prometheus.NewDesc("frostfs_s3_billing", "Billing statistics exposed by FrostFS S3 Gate instance", nil, nil),
		apiStat:  UsersAPIStats{},
	}
}

func (b *billingMetrics) register() {
	b.registry.MustRegister(b)
}

func (b *billingMetrics) unregister() {
	b.registry.Unregister(b)
}

func (b *billingMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.desc
}

func (b *billingMetrics) Collect(ch chan<- prometheus.Metric) {
	userMetrics := b.apiStat.DumpMetrics()

	for _, value := range userMetrics.Requests {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, billingSubsystem, "user_requests"),
				"",
				[]string{"user", "bucket", "cid", "operation"}, nil),
			prometheus.CounterValue,
			float64(value.Requests),
			value.User,
			value.Bucket,
			value.ContainerID,
			value.Operation.String(),
		)
	}

	for _, value := range userMetrics.Traffic {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, billingSubsystem, "user_traffic"),
				"",
				[]string{"user", "bucket", "cid", "direction"}, nil),
			prometheus.CounterValue,
			float64(value.Value),
			value.User,
			value.Bucket,
			value.ContainerID,
			value.Type.String(),
		)
	}
}
