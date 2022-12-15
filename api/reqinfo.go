package api

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type (
	// KeyVal -- appended to ReqInfo.Tags.
	KeyVal struct {
		Key string
		Val string
	}

	// ReqInfo stores the request info.
	ReqInfo struct {
		sync.RWMutex
		RemoteHost   string   // Client Host/IP
		Host         string   // Node Host/IP
		UserAgent    string   // User Agent
		DeploymentID string   // random generated s3-deployment-id
		RequestID    string   // x-amz-request-id
		API          string   // API name -- GetObject PutObject NewMultipartUpload etc.
		BucketName   string   // Bucket name
		ObjectName   string   // Object name
		URL          *url.URL // Request url
		tags         []KeyVal // Any additional info not accommodated by above fields
	}

	// ObjectRequest represents object request data.
	ObjectRequest struct {
		Bucket string
		Object string
		Method string
	}
)

// Key used for Get/SetReqInfo.
type contextKeyType string

const ctxRequestInfo = contextKeyType("FrostFS-S3-GW")

var (
	// De-facto standard header keys.
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

var (
	// RFC7239 defines a new "Forwarded: " header designed to replace the
	// existing use of X-Forwarded-* headers.
	// e.g. Forwarded: for=192.0.2.60;proto=https;by=203.0.113.43.
	forwarded = http.CanonicalHeaderKey("Forwarded")
	// Allows for a sub-match of the first value after 'for=' to the next
	// comma, semi-colon or space. The match is case-insensitive.
	forRegex = regexp.MustCompile(`(?i)(?:for=)([^(;|, )]+)(.*)`)
)

// GetSourceIP retrieves the IP from the X-Forwarded-For, X-Real-IP and RFC7239
// Forwarded headers (in that order), falls back to r.RemoteAddr when everything
// else fails.
func GetSourceIP(r *http.Request) string {
	var addr string

	if fwd := r.Header.Get(xForwardedFor); fwd != "" {
		// Only grabs the first (client) address. Note that '192.168.0.1,
		// 10.1.1.1' is a valid key for X-Forwarded-For where addresses after
		// the first one may represent forwarding proxies earlier in the chain.
		s := strings.Index(fwd, ", ")
		if s == -1 {
			s = len(fwd)
		}
		addr = fwd[:s]
	} else if fwd := r.Header.Get(xRealIP); fwd != "" {
		// X-Real-IP should only contain one IP address (the client making the
		// request).
		addr = fwd
	} else if fwd := r.Header.Get(forwarded); fwd != "" {
		// match should contain at least two elements if the protocol was
		// specified in the Forwarded header. The first element will always be
		// the 'for=' capture, which we ignore. In the case of multiple IP
		// addresses (for=8.8.8.8, 8.8.4.4, 172.16.1.20 is valid) we only
		// extract the first, which should be the client IP.
		if match := forRegex.FindStringSubmatch(fwd); len(match) > 1 {
			// IPv6 addresses in Forwarded headers are quoted-strings. We strip
			// these quotes.
			addr = strings.Trim(match[1], `"`)
		}
	}

	if addr != "" {
		return addr
	}

	// Default to remote address if headers not set.
	addr, _, _ = net.SplitHostPort(r.RemoteAddr)
	return addr
}

func prepareContext(w http.ResponseWriter, r *http.Request) context.Context {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		object = vars["object"]
	}
	prefix, err := url.QueryUnescape(vars["prefix"])
	if err != nil {
		prefix = vars["prefix"]
	}
	if prefix != "" {
		object = prefix
	}
	return SetReqInfo(r.Context(),
		// prepare request info
		NewReqInfo(w, r, ObjectRequest{
			Bucket: bucket,
			Object: object,
			Method: mux.CurrentRoute(r).GetName(),
		}))
}

// NewReqInfo returns new ReqInfo based on parameters.
func NewReqInfo(w http.ResponseWriter, r *http.Request, req ObjectRequest) *ReqInfo {
	return &ReqInfo{
		API:          req.Method,
		BucketName:   req.Bucket,
		ObjectName:   req.Object,
		UserAgent:    r.UserAgent(),
		RemoteHost:   GetSourceIP(r),
		RequestID:    GetRequestID(w),
		DeploymentID: deploymentID.String(),
		URL:          r.URL,
	}
}

// AppendTags -- appends key/val to ReqInfo.tags.
func (r *ReqInfo) AppendTags(key string, val string) *ReqInfo {
	if r == nil {
		return nil
	}
	r.Lock()
	defer r.Unlock()
	r.tags = append(r.tags, KeyVal{key, val})
	return r
}

// SetTags -- sets key/val to ReqInfo.tags.
func (r *ReqInfo) SetTags(key string, val string) *ReqInfo {
	if r == nil {
		return nil
	}
	r.Lock()
	defer r.Unlock()
	// Search for a tag key already existing in tags
	var updated bool
	for _, tag := range r.tags {
		if tag.Key == key {
			tag.Val = val
			updated = true
			break
		}
	}
	if !updated {
		// Append to the end of tags list
		r.tags = append(r.tags, KeyVal{key, val})
	}
	return r
}

// GetTags -- returns the user defined tags.
func (r *ReqInfo) GetTags() []KeyVal {
	if r == nil {
		return nil
	}
	r.RLock()
	defer r.RUnlock()
	return append([]KeyVal(nil), r.tags...)
}

// SetReqInfo sets ReqInfo in the context.
func SetReqInfo(ctx context.Context, req *ReqInfo) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, ctxRequestInfo, req)
}

// GetReqInfo returns ReqInfo if set.
func GetReqInfo(ctx context.Context) *ReqInfo {
	if ctx == nil {
		return &ReqInfo{}
	} else if r, ok := ctx.Value(ctxRequestInfo).(*ReqInfo); ok {
		return r
	}
	return &ReqInfo{}
}
