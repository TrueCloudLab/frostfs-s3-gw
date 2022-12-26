package api

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/prometheus/client_golang/prometheus"
)

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

type RequestType int

const (
	HEADRequest   RequestType = iota
	PUTRequest    RequestType = iota
	LISTRequest   RequestType = iota
	GETRequest    RequestType = iota
	DELETERequest RequestType = iota
)

func (t RequestType) String() string {
	switch t {
	case 0:
		return "HEAD"
	case 1:
		return "PUT"
	case 2:
		return "LIST"
	case 3:
		return "GET"
	case 4:
		return "DELETE"
	default:
		return "Unknown"
	}
}

func RequestTypeFromAPI(api string) RequestType {
	switch api {
	case "headobject", "headbucket":
		return HEADRequest
	case "createmultipartupload", "uploadpartcopy", "uploadpart", "completemutipartupload",
		"putobjectacl", "putobjecttagging", "copyobject", "putobjectretention", "putobjectlegalhold",
		"putobject", "putbucketcors", "putbucketacl", "putbucketlifecycle", "putbucketencryption",
		"putbucketpolicy", "putbucketobjectlockconfig", "putbuckettagging", "putbucketversioning",
		"putbucketnotification", "createbucket", "postobject":
		return PUTRequest
	case "listmultipartuploads", "listobjectsv2M", "listobjectsv2", "listbucketversions",
		"listobjectsv1", "listbuckets":
		return LISTRequest
	case "getobjectacl", "getobjecttagging", "getobjectretention", "getobjectlegalhold",
		"getobjectattributes", "getobject", "getbucketlocation", "getbucketpolicy",
		"getbucketlifecycle", "getbucketencryption", "getbucketcors", "getbucketacl",
		"getbucketwebsite", "getbucketaccelerate", "getbucketrequestpayment", "getbucketlogging",
		"getbucketreplication", "getbuckettagging", "selectobjectcontent",
		"getbucketobjectlockconfiguration", "getbucketversioning", "getbucketnotification",
		"listenbucketnotification":
		return GETRequest
	case "abortmultipartupload", "deleteobjecttagging", "deleteobject", "deletebucketcors",
		"deletebucketwebsite", "deletebuckettagging", "deletemultipleobjects", "deletebucketpolicy",
		"deletebucketlifecycle", "deletebucketencryption", "deletebucket":
		return DELETERequest
	default:
		return RequestType(-1)
	}
}

type OperationList [5]int

type (
	// HTTPAPIStats holds statistics information about
	// the API given in the requests.
	HTTPAPIStats struct {
		apiStats map[string]int
		sync.RWMutex
	}

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
		Operation RequestType
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

	// HTTPStats holds statistics information about
	// HTTP requests made by all clients.
	HTTPStats struct {
		usersS3Requests   UsersAPIStats
		currentS3Requests HTTPAPIStats
		totalS3Requests   HTTPAPIStats
		totalS3Errors     HTTPAPIStats

		totalInputBytes  uint64
		totalOutputBytes uint64
	}

	readCounter struct {
		io.ReadCloser
		countBytes uint64
	}

	writeCounter struct {
		http.ResponseWriter
		countBytes uint64
	}

	responseWrapper struct {
		sync.Once
		http.ResponseWriter

		statusCode int
		startTime  time.Time
	}
)

const systemPath = "/system"

var (
	httpStatsMetric      = new(HTTPStats)
	httpRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "frostfs_s3_request_seconds",
			Help:    "Time taken by requests served by current FrostFS S3 Gate instance",
			Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"api"},
	)
)

// Collects HTTP metrics for FrostFS S3 Gate in Prometheus specific format
// and sends to the given channel.
func collectHTTPMetrics(ch chan<- prometheus.Metric) {
	for api, value := range httpStatsMetric.currentS3Requests.Load() {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("frostfs_s3", "requests", "current"),
				"Total number of running s3 requests in current FrostFS S3 Gate instance",
				[]string{"api"}, nil),
			prometheus.CounterValue,
			float64(value),
			api,
		)
	}

	for api, value := range httpStatsMetric.totalS3Requests.Load() {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("frostfs_s3", "requests", "total"),
				"Total number of s3 requests in current FrostFS S3 Gate instance",
				[]string{"api"}, nil),
			prometheus.CounterValue,
			float64(value),
			api,
		)
	}

	for api, value := range httpStatsMetric.totalS3Errors.Load() {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("frostfs_s3", "errors", "total"),
				"Total number of s3 errors in current FrostFS S3 Gate instance",
				[]string{"api"}, nil),
			prometheus.CounterValue,
			float64(value),
			api,
		)
	}
}

func collectUserMetrics(ch chan<- prometheus.Metric) {
	userMetrics := httpStatsMetric.usersS3Requests.DumpMetrics()

	for _, value := range userMetrics.Requests {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("frostfs_s3", "user_requests", "count"),
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
				prometheus.BuildFQName("frostfs_s3", "user_traffic", "bytes"),
				"",
				[]string{"user", "bucket", "cid", "type"}, nil),
			prometheus.CounterValue,
			float64(value.Value),
			value.User,
			value.Bucket,
			value.ContainerID,
			value.Type.String(),
		)
	}
}

// APIStats wraps http handler for api with basic statistics collection.
func APIStats(api string, f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpStatsMetric.currentS3Requests.Inc(api)
		defer httpStatsMetric.currentS3Requests.Dec(api)

		in := &readCounter{ReadCloser: r.Body}
		out := &writeCounter{ResponseWriter: w}

		r.Body = in

		statsWriter := &responseWrapper{
			ResponseWriter: out,
			startTime:      time.Now(),
		}

		f.ServeHTTP(statsWriter, r)

		// Time duration in secs since the call started.
		// We don't need to do nanosecond precision here
		// simply for the fact that it is not human readable.
		durationSecs := time.Since(statsWriter.startTime).Seconds()

		httpStatsMetric.updateStats(api, statsWriter, r, durationSecs, in.countBytes, out.countBytes)

		atomic.AddUint64(&httpStatsMetric.totalInputBytes, in.countBytes)
		atomic.AddUint64(&httpStatsMetric.totalOutputBytes, out.countBytes)
	}
}

// Inc increments the api stats counter.
func (stats *HTTPAPIStats) Inc(api string) {
	if stats == nil {
		return
	}
	stats.Lock()
	defer stats.Unlock()
	if stats.apiStats == nil {
		stats.apiStats = make(map[string]int)
	}
	stats.apiStats[api]++
}

// Dec increments the api stats counter.
func (stats *HTTPAPIStats) Dec(api string) {
	if stats == nil {
		return
	}
	stats.Lock()
	defer stats.Unlock()
	if val, ok := stats.apiStats[api]; ok && val > 0 {
		stats.apiStats[api]--
	}
}

// Load returns the recorded stats.
func (stats *HTTPAPIStats) Load() map[string]int {
	stats.Lock()
	defer stats.Unlock()
	var apiStats = make(map[string]int, len(stats.apiStats))
	for k, v := range stats.apiStats {
		apiStats[k] = v
	}
	return apiStats
}

func (u *UsersAPIStats) Update(user, bucket, cnrID string, reqType RequestType, in, out uint64) {
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
	bktStat.Operations[reqType] += 1
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
						Operation:      RequestType(op),
						Requests:       val,
					})
				}
			}
		}
	}

	u.users = make(map[string]*userAPIStats)

	return result
}

func (st *HTTPStats) getInputBytes() uint64 {
	return atomic.LoadUint64(&st.totalInputBytes)
}

func (st *HTTPStats) getOutputBytes() uint64 {
	return atomic.LoadUint64(&st.totalOutputBytes)
}

// Update statistics from http request and response data.
func (st *HTTPStats) updateStats(apiOperation string, w http.ResponseWriter, r *http.Request, durationSecs float64, in, out uint64) {
	var code int

	if res, ok := w.(*responseWrapper); ok {
		code = res.statusCode
	}

	user := "anon"
	if bd, ok := r.Context().Value(BoxData).(*accessbox.Box); ok && bd != nil && bd.Gate != nil && bd.Gate.BearerToken != nil {
		user = bearer.ResolveIssuer(*bd.Gate.BearerToken).String()
	}

	reqInfo := GetReqInfo(r.Context())
	cnrID := GetCID(r.Context())

	st.usersS3Requests.Update(user, reqInfo.BucketName, cnrID, RequestTypeFromAPI(apiOperation), in, out)

	// A successful request has a 2xx response code
	successReq := code >= http.StatusOK && code < http.StatusMultipleChoices

	if !strings.HasSuffix(r.URL.Path, systemPath) {
		st.totalS3Requests.Inc(apiOperation)
		if !successReq && code != 0 {
			st.totalS3Errors.Inc(apiOperation)
		}
	}

	if r.Method == http.MethodGet {
		// Increment the prometheus http request response histogram with appropriate label
		httpRequestsDuration.With(prometheus.Labels{"api": apiOperation}).Observe(durationSecs)
	}
}

// WriteHeader -- writes http status code.
func (w *responseWrapper) WriteHeader(code int) {
	w.Do(func() {
		w.statusCode = code
		w.ResponseWriter.WriteHeader(code)
	})
}

// Flush -- calls the underlying Flush.
func (w *responseWrapper) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *writeCounter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	atomic.AddUint64(&w.countBytes, uint64(n))
	return n, err
}

func (r *readCounter) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	atomic.AddUint64(&r.countBytes, uint64(n))
	return n, err
}
