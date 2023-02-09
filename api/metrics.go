package api

import (
	"context"
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

type RequestType int

const (
	UNKNOWNRequest RequestType = iota
	HEADRequest    RequestType = iota
	PUTRequest     RequestType = iota
	LISTRequest    RequestType = iota
	GETRequest     RequestType = iota
	DELETERequest  RequestType = iota
)

func (t RequestType) String() string {
	switch t {
	case 1:
		return "HEAD"
	case 2:
		return "PUT"
	case 3:
		return "LIST"
	case 4:
		return "GET"
	case 5:
		return "DELETE"
	default:
		return "Unknown"
	}
}

func RequestTypeFromAPI(api string) RequestType {
	switch api {
	case "Options", "HeadObject", "HeadBucket":
		return HEADRequest
	case "CreateMultipartUpload", "UploadPartCopy", "UploadPart", "CompleteMultipartUpload",
		"PutObjectACL", "PutObjectTagging", "CopyObject", "PutObjectRetention", "PutObjectLegalHold",
		"PutObject", "PutBucketCors", "PutBucketACL", "PutBucketLifecycle", "PutBucketEncryption",
		"PutBucketPolicy", "PutBucketObjectLockConfig", "PutBucketTagging", "PutBucketVersioning",
		"PutBucketNotification", "CreateBucket", "PostObject":
		return PUTRequest
	case "ListObjectParts", "ListMultipartUploads", "ListObjectsV2M", "ListObjectsV2", "ListBucketVersions",
		"ListObjectsV1", "ListBuckets":
		return LISTRequest
	case "GetObjectACL", "GetObjectTagging", "SelectObjectContent", "GetObjectRetention", "getobjectlegalhold",
		"GetObjectAttributes", "GetObject", "GetBucketLocation", "GetBucketPolicy",
		"GetBucketLifecycle", "GetBucketEncryption", "GetBucketCors", "GetBucketACL",
		"GetBucketWebsite", "GetBucketAccelerate", "GetBucketRequestPayment", "GetBucketLogging",
		"GetBucketReplication", "GetBucketTagging", "GetBucketObjectLockConfig",
		"GetBucketVersioning", "GetBucketNotification", "ListenBucketNotification":
		return GETRequest
	case "AbortMultipartUpload", "DeleteObjectTagging", "DeleteObject", "DeleteBucketCors",
		"DeleteBucketWebsite", "DeleteBucketTagging", "DeleteMultipleObjects", "DeleteBucketPolicy",
		"DeleteBucketLifecycle", "DeleteBucketEncryption", "DeleteBucket":
		return DELETERequest
	default:
		return UNKNOWNRequest
	}
}

type (
	// HTTPAPIStats holds statistics information about
	// the API given in the requests.
	HTTPAPIStats struct {
		apiStats map[string]int
		sync.RWMutex
	}

	UsersStat interface {
		Update(user, bucket, cnrID string, reqType RequestType, in, out uint64)
	}

	// HTTPStats holds statistics information about
	// HTTP requests made by all clients.
	HTTPStats struct {
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

// CIDResolveFunc is a func to resolve CID in Stats handler.
type CIDResolveFunc func(ctx context.Context, reqInfo *ReqInfo) (cnrID string)

// Stats is a handler that update metrics.
func Stats(f http.HandlerFunc, resolveCID CIDResolveFunc, usersStat UsersStat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqInfo := GetReqInfo(r.Context())

		httpStatsMetric.currentS3Requests.Inc(reqInfo.API)
		defer httpStatsMetric.currentS3Requests.Dec(reqInfo.API)

		in := &readCounter{ReadCloser: r.Body}
		out := &writeCounter{ResponseWriter: w}

		r.Body = in

		statsWriter := &responseWrapper{
			ResponseWriter: out,
			startTime:      time.Now(),
		}

		f(statsWriter, r)

		// Time duration in secs since the call started.
		// We don't need to do nanosecond precision here
		// simply for the fact that it is not human-readable.
		durationSecs := time.Since(statsWriter.startTime).Seconds()

		user := resolveUser(r.Context())
		cnrID := resolveCID(r.Context(), reqInfo)
		usersStat.Update(user, reqInfo.BucketName, cnrID, RequestTypeFromAPI(reqInfo.API), in.countBytes, out.countBytes)

		code := statsWriter.statusCode
		// A successful request has a 2xx response code
		successReq := code >= http.StatusOK && code < http.StatusMultipleChoices
		if !strings.HasSuffix(r.URL.Path, systemPath) {
			httpStatsMetric.totalS3Requests.Inc(reqInfo.API)
			if !successReq && code != 0 {
				httpStatsMetric.totalS3Errors.Inc(reqInfo.API)
			}
		}

		if r.Method == http.MethodGet {
			// Increment the prometheus http request response histogram with appropriate label
			httpRequestsDuration.With(prometheus.Labels{"api": reqInfo.API}).Observe(durationSecs)
		}

		atomic.AddUint64(&httpStatsMetric.totalInputBytes, in.countBytes)
		atomic.AddUint64(&httpStatsMetric.totalOutputBytes, out.countBytes)
	}
}

func resolveUser(ctx context.Context) string {
	user := "anon"
	if bd, ok := ctx.Value(BoxData).(*accessbox.Box); ok && bd != nil && bd.Gate != nil && bd.Gate.BearerToken != nil {
		user = bearer.ResolveIssuer(*bd.Gate.BearerToken).String()
	}
	return user
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

func (st *HTTPStats) getInputBytes() uint64 {
	return atomic.LoadUint64(&st.totalInputBytes)
}

func (st *HTTPStats) getOutputBytes() uint64 {
	return atomic.LoadUint64(&st.totalOutputBytes)
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
