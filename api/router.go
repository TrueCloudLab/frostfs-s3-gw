package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/auth"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type (
	// Handler is an S3 API handler interface.
	Handler interface {
		HeadObjectHandler(http.ResponseWriter, *http.Request)
		GetObjectACLHandler(http.ResponseWriter, *http.Request)
		PutObjectACLHandler(http.ResponseWriter, *http.Request)
		GetObjectTaggingHandler(http.ResponseWriter, *http.Request)
		PutObjectTaggingHandler(http.ResponseWriter, *http.Request)
		DeleteObjectTaggingHandler(http.ResponseWriter, *http.Request)
		SelectObjectContentHandler(http.ResponseWriter, *http.Request)
		GetObjectRetentionHandler(http.ResponseWriter, *http.Request)
		GetObjectLegalHoldHandler(http.ResponseWriter, *http.Request)
		GetObjectHandler(http.ResponseWriter, *http.Request)
		GetObjectAttributesHandler(http.ResponseWriter, *http.Request)
		CopyObjectHandler(http.ResponseWriter, *http.Request)
		PutObjectRetentionHandler(http.ResponseWriter, *http.Request)
		PutObjectLegalHoldHandler(http.ResponseWriter, *http.Request)
		PutObjectHandler(http.ResponseWriter, *http.Request)
		DeleteObjectHandler(http.ResponseWriter, *http.Request)
		GetBucketLocationHandler(http.ResponseWriter, *http.Request)
		GetBucketPolicyHandler(http.ResponseWriter, *http.Request)
		GetBucketLifecycleHandler(http.ResponseWriter, *http.Request)
		GetBucketEncryptionHandler(http.ResponseWriter, *http.Request)
		GetBucketACLHandler(http.ResponseWriter, *http.Request)
		PutBucketACLHandler(http.ResponseWriter, *http.Request)
		GetBucketCorsHandler(http.ResponseWriter, *http.Request)
		PutBucketCorsHandler(http.ResponseWriter, *http.Request)
		DeleteBucketCorsHandler(http.ResponseWriter, *http.Request)
		GetBucketWebsiteHandler(http.ResponseWriter, *http.Request)
		GetBucketAccelerateHandler(http.ResponseWriter, *http.Request)
		GetBucketRequestPaymentHandler(http.ResponseWriter, *http.Request)
		GetBucketLoggingHandler(http.ResponseWriter, *http.Request)
		GetBucketReplicationHandler(http.ResponseWriter, *http.Request)
		GetBucketTaggingHandler(http.ResponseWriter, *http.Request)
		DeleteBucketWebsiteHandler(http.ResponseWriter, *http.Request)
		DeleteBucketTaggingHandler(http.ResponseWriter, *http.Request)
		GetBucketObjectLockConfigHandler(http.ResponseWriter, *http.Request)
		GetBucketVersioningHandler(http.ResponseWriter, *http.Request)
		GetBucketNotificationHandler(http.ResponseWriter, *http.Request)
		ListenBucketNotificationHandler(http.ResponseWriter, *http.Request)
		ListObjectsV2MHandler(http.ResponseWriter, *http.Request)
		ListObjectsV2Handler(http.ResponseWriter, *http.Request)
		ListBucketObjectVersionsHandler(http.ResponseWriter, *http.Request)
		ListObjectsV1Handler(http.ResponseWriter, *http.Request)
		PutBucketLifecycleHandler(http.ResponseWriter, *http.Request)
		PutBucketEncryptionHandler(http.ResponseWriter, *http.Request)
		PutBucketPolicyHandler(http.ResponseWriter, *http.Request)
		PutBucketObjectLockConfigHandler(http.ResponseWriter, *http.Request)
		PutBucketTaggingHandler(http.ResponseWriter, *http.Request)
		PutBucketVersioningHandler(http.ResponseWriter, *http.Request)
		PutBucketNotificationHandler(http.ResponseWriter, *http.Request)
		CreateBucketHandler(http.ResponseWriter, *http.Request)
		HeadBucketHandler(http.ResponseWriter, *http.Request)
		PostObject(http.ResponseWriter, *http.Request)
		DeleteMultipleObjectsHandler(http.ResponseWriter, *http.Request)
		DeleteBucketPolicyHandler(http.ResponseWriter, *http.Request)
		DeleteBucketLifecycleHandler(http.ResponseWriter, *http.Request)
		DeleteBucketEncryptionHandler(http.ResponseWriter, *http.Request)
		DeleteBucketHandler(http.ResponseWriter, *http.Request)
		ListBucketsHandler(http.ResponseWriter, *http.Request)
		Preflight(w http.ResponseWriter, r *http.Request)
		AppendCORSHeaders(w http.ResponseWriter, r *http.Request)
		CreateMultipartUploadHandler(http.ResponseWriter, *http.Request)
		UploadPartHandler(http.ResponseWriter, *http.Request)
		UploadPartCopy(w http.ResponseWriter, r *http.Request)
		CompleteMultipartUploadHandler(http.ResponseWriter, *http.Request)
		AbortMultipartUploadHandler(http.ResponseWriter, *http.Request)
		ListPartsHandler(w http.ResponseWriter, r *http.Request)
		ListMultipartUploadsHandler(http.ResponseWriter, *http.Request)
	}

	// mimeType represents various MIME types used in API responses.
	mimeType string

	logResponseWriter struct {
		sync.Once
		http.ResponseWriter

		statusCode int
	}
)

const (
	// SlashSeparator -- slash separator.
	SlashSeparator = "/"

	// MimeNone means no response type.
	MimeNone mimeType = ""

	// MimeXML means response type is XML.
	MimeXML mimeType = "application/xml"
)

var _ = logErrorResponse

func (lrw *logResponseWriter) WriteHeader(code int) {
	lrw.Do(func() {
		lrw.statusCode = code
		lrw.ResponseWriter.WriteHeader(code)
	})
}

func setRequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// generate random UUIDv4
		id, _ := uuid.NewRandom()

		// set request id into response header
		w.Header().Set(hdrAmzRequestID, id.String())

		// set request id into gRPC meta header
		r = r.WithContext(metadata.AppendToOutgoingContext(
			r.Context(), hdrAmzRequestID, id.String(),
		))

		// set request info into context
		// bucket name and object will be set in reqInfo later (limitation of go-chi)
		r = r.WithContext(SetReqInfo(r.Context(), NewReqInfo(w, r, ObjectRequest{})))

		// continue execution
		h.ServeHTTP(w, r)
	})
}

// addBucketName adds bucket name to ReqInfo from context.
func addBucketName(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqInfo := GetReqInfo(r.Context())
		reqInfo.BucketName = chi.URLParam(r, "bucket")
		h.ServeHTTP(w, r)
	})
}

// addObjectName adds objects name to ReqInfo from context.
func addObjectName(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		obj := chi.URLParam(r, "object")
		object, err := url.PathUnescape(obj)
		if err != nil {
			object = obj
		}
		prefix, err := url.QueryUnescape(chi.URLParam(r, "prefix"))
		if err != nil {
			prefix = chi.URLParam(r, "prefix")
		}
		if prefix != "" {
			object = prefix
		}

		reqInfo := GetReqInfo(r.Context())
		reqInfo.ObjectName = object

		h.ServeHTTP(w, r)
	})
}

func appendCORS(handler Handler) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.AppendCORSHeaders(w, r)
			h.ServeHTTP(w, r)
		})
	}
}

func logErrorResponse(l *zap.Logger) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &logResponseWriter{ResponseWriter: w}
			reqInfo := GetReqInfo(r.Context())

			// here reqInfo doesn't contain bucket name and object name

			// pass execution:
			h.ServeHTTP(lw, r)

			// here reqInfo contains bucket name and object name because of
			// addBucketName and addObjectName middlewares

			// Ignore >400 status codes
			if lw.statusCode >= http.StatusBadRequest {
				return
			}

			l.Info("call method",
				zap.Int("status", lw.statusCode),
				zap.String("host", r.Host),
				zap.String("request_id", GetRequestID(r.Context())),
				zap.String("method", reqInfo.API),
				zap.String("bucket", reqInfo.BucketName),
				zap.String("object", reqInfo.ObjectName),
				zap.String("description", http.StatusText(lw.statusCode)))
		})
	}
}

// GetRequestID returns the request ID from the response writer or the context.
func GetRequestID(v interface{}) string {
	switch t := v.(type) {
	case context.Context:
		return GetReqInfo(t).RequestID
	case http.ResponseWriter:
		return t.Header().Get(hdrAmzRequestID)
	default:
		panic("unknown type")
	}
}

func authMiddleware(center auth.Center, log *zap.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var ctx context.Context
			box, err := center.Authenticate(r)
			if err != nil {
				if err == auth.ErrNoAuthorizationHeader {
					log.Debug("couldn't receive access box for gate key, random key will be used")
					ctx = r.Context()
				} else {
					log.Error("failed to pass authentication", zap.Error(err))
					if _, ok := err.(errors.Error); !ok {
						err = errors.GetAPIError(errors.ErrAccessDenied)
					}
					WriteErrorResponse(w, GetReqInfo(r.Context()), err)
					return
				}
			} else {
				ctx = context.WithValue(r.Context(), BoxData, box.AccessBox)
				if !box.ClientTime.IsZero() {
					ctx = context.WithValue(ctx, ClientTime, box.ClientTime)
				}
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AttachChi(api *chi.Mux, domains []string, throttle middleware.ThrottleOpts, h Handler, center auth.Center, log *zap.Logger) {
	api.Use(
		middleware.CleanPath,
		setRequestID,
		logErrorResponse(log),
		middleware.ThrottleWithOpts(throttle),
		middleware.Recoverer,
		authMiddleware(center, log),
	)

	// todo reconsider host routing
	hr := hostrouter.New()
	for _, domain := range domains {
		hr.Map("*."+domain, bucketRouter(h, log))
	}

	api.Mount("/", hr)
	api.Mount("/{bucket}", bucketRouter(h, log))
	api.Get("/", h.ListBucketsHandler)

	// If none of the routes match, add default error handler routes
	api.NotFound(metrics.APIStats("notfound", errorResponseHandler))
	api.MethodNotAllowed(metrics.APIStats("methodnotallowed", errorResponseHandler))
}

func Named(name string, handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqInfo := GetReqInfo(r.Context())
		reqInfo.API = name
		handlerFunc.ServeHTTP(w, r)
	}
}

func bucketRouter(h Handler, log *zap.Logger) chi.Router {
	bktRouter := chi.NewRouter()
	bktRouter.Use(
		addBucketName,
		appendCORS(h),
	)

	bktRouter.Mount("/{object}", objectRouter(h, log))

	bktRouter.Options("/", h.Preflight)

	bktRouter.Head("/", Named("HeadBucket", h.HeadBucketHandler))

	// GET method handlers
	bktRouter.Group(func(r chi.Router) {
		r.Method(http.MethodGet, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("upload").
				Handler(Named("ListMultipartUploads", h.ListMultipartUploadsHandler))).
			Add(NewFilter().
				Queries("location").
				Handler(Named("GetBucketLocation", h.GetBucketLocationHandler))).
			Add(NewFilter().
				Queries("policy").
				Handler(Named("GetBucketPolicy", h.GetBucketPolicyHandler))).
			Add(NewFilter().
				Queries("lifecycle").
				Handler(Named("GetBucketLifecycle", h.GetBucketLifecycleHandler))).
			Add(NewFilter().
				Queries("encryption").
				Handler(Named("GetBucketEncryption", h.GetBucketEncryptionHandler))).
			Add(NewFilter().
				Queries("cors").
				Handler(Named("GetBucketCors", h.GetBucketCorsHandler))).
			Add(NewFilter().
				Queries("acl").
				Handler(Named("GetBucketACL", h.GetBucketACLHandler))).
			Add(NewFilter().
				Queries("website").
				Handler(Named("GetBucketWebsite", h.GetBucketWebsiteHandler))).
			Add(NewFilter().
				Queries("accelerate").
				Handler(Named("GetBucketAccelerate", h.GetBucketAccelerateHandler))).
			Add(NewFilter().
				Queries("requestPayment").
				Handler(Named("GetBucketRequestPayment", h.GetBucketRequestPaymentHandler))).
			Add(NewFilter().
				Queries("logging").
				Handler(Named("GetBucketLogging", h.GetBucketLoggingHandler))).
			Add(NewFilter().
				Queries("replication").
				Handler(Named("GetBucketReplication", h.GetBucketReplicationHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("GetBucketTagging", h.GetBucketTaggingHandler))).
			Add(NewFilter().
				Queries("object-lock").
				Handler(Named("GetBucketObjectLockConfig", h.GetBucketObjectLockConfigHandler))).
			Add(NewFilter().
				Queries("versioning").
				Handler(Named("GetBucketVersioning", h.GetBucketVersioningHandler))).
			Add(NewFilter().
				Queries("notification").
				Handler(Named("GetBucketNotification", h.GetBucketNotificationHandler))).
			Add(NewFilter().
				Queries("events").
				Handler(Named("ListenBucketNotification", h.ListenBucketNotificationHandler))).
			Add(NewFilter().
				QueriesMatch("list-type", "2", "metadata", "true").
				Handler(Named("ListObjectsV2M", h.ListObjectsV2MHandler))).
			Add(NewFilter().
				QueriesMatch("list-type", "2").
				Handler(Named("ListObjectsV2", h.ListObjectsV2Handler))).
			Add(NewFilter().
				Queries("versions").
				Handler(Named("ListBucketObjectVersions", h.ListBucketObjectVersionsHandler))).
			DefaultHandler(Named("ListObjectsV1", h.ListObjectsV1Handler)))
	})

	// PUT method handlers
	bktRouter.Group(func(r chi.Router) {
		r.Method(http.MethodPut, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("cors").
				Handler(Named("PutBucketCors", h.PutBucketCorsHandler))).
			Add(NewFilter().
				Queries("acl").
				Handler(Named("PutBucketACL", h.PutBucketACLHandler))).
			Add(NewFilter().
				Queries("lifecycle").
				Handler(Named("PutBucketLifecycle", h.PutBucketLifecycleHandler))).
			Add(NewFilter().
				Queries("encryption").
				Handler(Named("PutBucketEncryption", h.PutBucketEncryptionHandler))).
			Add(NewFilter().
				Queries("policy").
				Handler(Named("PutBucketPolicy", h.PutBucketPolicyHandler))).
			Add(NewFilter().
				Queries("object-lock").
				Handler(Named("PutBucketObjectLockConfig", h.PutBucketObjectLockConfigHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("PutBucketTagging", h.PutBucketTaggingHandler))).
			Add(NewFilter().
				Queries("versioning").
				Handler(Named("PutBucketVersioning", h.PutBucketVersioningHandler))).
			Add(NewFilter().
				Queries("notification").
				Handler(Named("PutBucketNotification", h.PutBucketNotificationHandler))).
			DefaultHandler(Named("CreateBucket", h.CreateBucketHandler)))
	})

	// POST method handlers
	bktRouter.Group(func(r chi.Router) {
		r.Method(http.MethodPost, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("delete").
				Handler(Named("DeleteMultipleObjects", h.DeleteMultipleObjectsHandler))).
			// todo consider add filter to match header for defaultHandler:  hdrContentType, "multipart/form-data*"
			DefaultHandler(Named("PostObject", h.PostObject)))
	})

	// DELETE method handlers
	bktRouter.Group(func(r chi.Router) {
		r.Method(http.MethodDelete, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("cors").
				Handler(Named("DeleteBucketCors", h.DeleteBucketCorsHandler))).
			Add(NewFilter().
				Queries("website").
				Handler(Named("DeleteBucketWebsite", h.DeleteBucketWebsiteHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("DeleteBucketTagging", h.DeleteBucketTaggingHandler))).
			Add(NewFilter().
				Queries("policy").
				Handler(Named("PutBucketPolicy", h.PutBucketPolicyHandler))).
			Add(NewFilter().
				Queries("lifecycle").
				Handler(Named("PutBucketLifecycle", h.PutBucketLifecycleHandler))).
			Add(NewFilter().
				Queries("encryption").
				Handler(Named("DeleteBucketEncryption", h.DeleteBucketEncryptionHandler))).
			DefaultHandler(Named("DeleteBucket", h.DeleteBucketHandler)))
	})

	return bktRouter
}

func objectRouter(h Handler, log *zap.Logger) chi.Router {
	objRouter := chi.NewRouter()
	objRouter.Use(addObjectName)

	objRouter.Head("/", Named("HeadObject", h.HeadObjectHandler))

	// GET method handlers
	objRouter.Group(func(r chi.Router) {
		r.Method(http.MethodGet, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("uploadId").
				Handler(Named("ListParts", h.ListPartsHandler))).
			Add(NewFilter().
				Queries("acl").
				Handler(Named("GetObjectACL", h.GetObjectACLHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("GetObjectTagging", h.GetObjectTaggingHandler))).
			Add(NewFilter().
				Queries("retention").
				Handler(Named("GetObjectRetention", h.GetObjectRetentionHandler))).
			Add(NewFilter().
				Queries("legal-hold").
				Handler(Named("GetObjectLegalHold", h.GetObjectLegalHoldHandler))).
			Add(NewFilter().
				Queries("attributes").
				Handler(Named("GetObjectAttributes", h.GetObjectAttributesHandler))).
			DefaultHandler(Named("GetObject", h.GetObjectHandler)))
	})

	// PUT method handlers
	objRouter.Group(func(r chi.Router) {
		r.Method(http.MethodPut, "/", NewHandlerFilter().
			Add(NewFilter().
				Headers(hdrAmzCopySource).
				Queries("partNumber", "uploadId").
				Handler(Named("UploadPartCopy", h.UploadPartCopy))).
			Add(NewFilter().
				Queries("partNumber", "uploadId").
				Handler(Named("UploadPart", h.UploadPartHandler))).
			Add(NewFilter().
				Queries("acl").
				Handler(Named("PutObjectACL", h.PutObjectACLHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("PutObjectTagging", h.PutObjectTaggingHandler))).
			Add(NewFilter().
				Headers(hdrAmzCopySource).
				Handler(Named("CopyObject", h.CopyObjectHandler))).
			Add(NewFilter().
				Queries("retention").
				Handler(Named("PutObjectRetention", h.PutObjectRetentionHandler))).
			Add(NewFilter().
				Queries("legal-hold").
				Handler(Named("PutObjectLegalHold", h.PutObjectLegalHoldHandler))).
			DefaultHandler(Named("PutObject", h.PutObjectHandler)))
	})

	// POST method handlers
	objRouter.Group(func(r chi.Router) {
		r.Method(http.MethodPost, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("uploadId").
				Handler(Named("CompleteMultipartUpload", h.CompleteMultipartUploadHandler))).
			Add(NewFilter().
				Queries("uploads").
				Handler(Named("CreateMultipartUpload", h.CreateMultipartUploadHandler))).
			DefaultHandler(Named("SelectObjectContent", h.SelectObjectContentHandler)))
	})

	// DELETE method handlers
	objRouter.Group(func(r chi.Router) {
		r.Method(http.MethodDelete, "/", NewHandlerFilter().
			Add(NewFilter().
				Queries("uploadId").
				Handler(Named("AbortMultipartUpload", h.AbortMultipartUploadHandler))).
			Add(NewFilter().
				Queries("tagging").
				Handler(Named("DeleteObjectTagging", h.DeleteObjectTaggingHandler))).
			DefaultHandler(Named("DeleteObject", h.DeleteObjectHandler)))
	})

	return objRouter
}

type HandlerFilters struct {
	filters        []Filter
	defaultHandler http.Handler
}

type Filter struct {
	queries []Pair
	headers []Pair
	h       http.Handler
}

type Pair struct {
	Key   string
	Value string
}

func NewHandlerFilter() *HandlerFilters {
	return &HandlerFilters{}
}

func NewFilter() *Filter {
	return &Filter{}
}

func (hf *HandlerFilters) Add(filter *Filter) *HandlerFilters {
	hf.filters = append(hf.filters, *filter)
	return hf
}

// HeadersMatch adds a matcher for header values.
// It accepts a sequence of key/value pairs. Values may define variables.
// Panics if number of parameters is not even.
// Supports only exact matching.
// If the value is an empty string, it will match any value if the key is set.
func (f *Filter) HeadersMatch(pairs ...string) *Filter {
	length := len(pairs)
	if length%2 != 0 {
		panic(fmt.Errorf("filter headers: number of parameters must be multiple of 2, got %v", pairs))
	}

	for i := 0; i < length; i += 2 {
		f.headers = append(f.headers, Pair{
			Key:   pairs[i],
			Value: pairs[i+1],
		})
	}

	return f
}

// Headers is similar to HeadersMatch but accept only header keys, set value to empty string internally.
func (f *Filter) Headers(headers ...string) *Filter {
	for _, header := range headers {
		f.headers = append(f.headers, Pair{
			Key:   header,
			Value: "",
		})
	}

	return f
}

func (f *Filter) Handler(handler http.HandlerFunc) *Filter {
	f.h = handler
	return f
}

// QueriesMatch adds a matcher for URL query values.
// It accepts a sequence of key/value pairs. Values may define variables.
// Panics if number of parameters is not even.
// Supports only exact matching.
// If the value is an empty string, it will match any value if the key is set.
func (f *Filter) QueriesMatch(pairs ...string) *Filter {
	length := len(pairs)
	if length%2 != 0 {
		panic(fmt.Errorf("filter headers: number of parameters must be multiple of 2, got %v", pairs))
	}

	for i := 0; i < length; i += 2 {
		f.queries = append(f.queries, Pair{
			Key:   pairs[i],
			Value: pairs[i+1],
		})
	}

	return f
}

// Queries is similar to QueriesMatch but accept only query keys, set value to empty string internally.
func (f *Filter) Queries(queries ...string) *Filter {
	for _, query := range queries {
		f.queries = append(f.queries, Pair{
			Key:   query,
			Value: "",
		})
	}

	return f
}

func (hf *HandlerFilters) DefaultHandler(handler http.HandlerFunc) *HandlerFilters {
	hf.defaultHandler = handler
	return hf
}

func (hf *HandlerFilters) ServeHTTP(w http.ResponseWriter, r *http.Request) {
LOOP:
	for _, filter := range hf.filters {
		for _, header := range filter.headers {
			hdrVals := r.Header.Values(header.Key)
			if len(hdrVals) == 0 || header.Value != "" && header.Value != hdrVals[0] {
				continue LOOP
			}
		}
		for _, query := range filter.queries {
			queryVal := r.URL.Query().Get(query.Key)
			if !r.URL.Query().Has(query.Key) || queryVal != "" && query.Value != queryVal {
				continue LOOP
			}
		}
		filter.h.ServeHTTP(w, r)
		return
	}

	hf.defaultHandler.ServeHTTP(w, r)
}
