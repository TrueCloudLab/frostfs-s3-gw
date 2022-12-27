package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/auth"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
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

		ResolveBucket(ctx context.Context, bucket string) (*data.BucketInfo, error)
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

var _ = logSuccessResponse

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
		r = r.WithContext(prepareContext(w, r))

		// continue execution
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

// BucketResolveFunc is a func to resolve bucket info by name.
type BucketResolveFunc func(ctx context.Context, bucket string) (*data.BucketInfo, error)

// metricsMiddleware wraps http handler for api with basic statistics collection.
func metricsMiddleware(log *zap.Logger, resolveBucket BucketResolveFunc) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return Stats(h.ServeHTTP, resolveCID(log, resolveBucket))
	}
}

// resolveCID forms CIDResolveFunc using BucketResolveFunc.
func resolveCID(log *zap.Logger, resolveBucket BucketResolveFunc) CIDResolveFunc {
	return func(ctx context.Context, reqInfo *ReqInfo) (cnrID string) {
		if reqInfo.BucketName == "" || reqInfo.API == "CreateBucket" || reqInfo.API == "" {
			return ""
		}

		bktInfo, err := resolveBucket(ctx, reqInfo.BucketName)
		if err != nil {
			log.Debug("failed to resolve CID",
				zap.String("request_id", reqInfo.RequestID), zap.String("method", reqInfo.API),
				zap.String("bucket", reqInfo.BucketName), zap.String("object", reqInfo.ObjectName),
				zap.Error(err))
			return ""
		}

		return bktInfo.CID.EncodeToString()
	}
}

func logSuccessResponse(l *zap.Logger) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &logResponseWriter{ResponseWriter: w}
			reqInfo := GetReqInfo(r.Context())

			// pass execution:
			h.ServeHTTP(lw, r)

			// Ignore >400 status codes
			if lw.statusCode >= http.StatusBadRequest {
				return
			}

			l.Info("call method",
				zap.Int("status", lw.statusCode),
				zap.String("host", r.Host),
				zap.String("request_id", GetRequestID(r.Context())),
				zap.String("method", mux.CurrentRoute(r).GetName()),
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

func setErrorAPI(apiName string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := SetReqInfo(r.Context(), &ReqInfo{API: apiName})
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// attachErrorHandler set NotFoundHandler and MethodNotAllowedHandler for mux.Router.
func attachErrorHandler(api *mux.Router, log *zap.Logger, h Handler, center auth.Center) {
	middlewares := []mux.MiddlewareFunc{
		AuthMiddleware(log, center),
		metricsMiddleware(log, h.ResolveBucket),
	}

	var errorHandler http.Handler = http.HandlerFunc(errorResponseHandler)
	for i := len(middlewares) - 1; i >= 0; i-- {
		errorHandler = middlewares[i](errorHandler)
	}

	// If none of the routes match, add default error handler routes
	api.NotFoundHandler = setErrorAPI("NotFound", errorHandler)
	api.MethodNotAllowedHandler = setErrorAPI("MethodNotAllowed", errorHandler)
}

// Attach adds S3 API handlers from h to r for domains with m client limit using
// center authentication and log logger.
func Attach(r *mux.Router, domains []string, m MaxClients, h Handler, center auth.Center, log *zap.Logger) {
	api := r.PathPrefix(SlashSeparator).Subrouter()

	api.Use(
		// -- prepare request
		setRequestID,

		// Attach user authentication for all S3 routes.
		AuthMiddleware(log, center),

		metricsMiddleware(log, h.ResolveBucket),

		// -- logging error requests
		logSuccessResponse(log),
	)

	attachErrorHandler(api, log, h, center)

	buckets := make([]*mux.Router, 0, len(domains)+1)
	buckets = append(buckets, api.PathPrefix("/{bucket}").Subrouter())

	for _, domain := range domains {
		buckets = append(buckets, api.Host("{bucket:.+}."+domain).Subrouter())
	}

	for _, bucket := range buckets {
		// Object operations
		// HeadObject
		bucket.Use(
			// -- append CORS headers to a response for
			appendCORS(h),
		)
		bucket.Methods(http.MethodOptions).HandlerFunc(
			m.Handle(h.Preflight)).
			Name("Options")
		bucket.Methods(http.MethodHead).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.HeadObjectHandler)).
			Name("HeadObject")
		// CopyObjectPart
		bucket.Methods(http.MethodPut).Path("/{object:.+}").Headers(hdrAmzCopySource, "").HandlerFunc(
			m.Handle(h.UploadPartCopy)).
			Queries("partNumber", "{partNumber:[0-9]+}", "uploadId", "{uploadId:.*}").
			Name("UploadPartCopy")
		// PutObjectPart
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.UploadPartHandler)).
			Queries("partNumber", "{partNumber:[0-9]+}", "uploadId", "{uploadId:.*}").
			Name("UploadPart")
		// ListParts
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.ListPartsHandler)).
			Queries("uploadId", "{uploadId:.*}").
			Name("ListObjectParts")
		// CompleteMultipartUpload
		bucket.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.CompleteMultipartUploadHandler)).
			Queries("uploadId", "{uploadId:.*}").
			Name("CompleteMultipartUpload")
		// CreateMultipartUpload
		bucket.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.CreateMultipartUploadHandler)).
			Queries("uploads", "").
			Name("CreateMultipartUpload")
		// AbortMultipartUpload
		bucket.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.AbortMultipartUploadHandler)).
			Queries("uploadId", "{uploadId:.*}").
			Name("AbortMultipartUpload")
		// ListMultipartUploads
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.ListMultipartUploadsHandler)).
			Queries("uploads", "").
			Name("ListMultipartUploads")
		// GetObjectACL -- this is a dummy call.
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectACLHandler)).
			Queries("acl", "").
			Name("GetObjectACL")
		// PutObjectACL -- this is a dummy call.
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.PutObjectACLHandler)).
			Queries("acl", "").
			Name("PutObjectACL")
		// GetObjectTagging
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectTaggingHandler)).
			Queries("tagging", "").
			Name("GetObjectTagging")
		// PutObjectTagging
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.PutObjectTaggingHandler)).
			Queries("tagging", "").
			Name("PutObjectTagging")
		// DeleteObjectTagging
		bucket.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.DeleteObjectTaggingHandler)).
			Queries("tagging", "").
			Name("DeleteObjectTagging")
		// SelectObjectContent
		bucket.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.SelectObjectContentHandler)).
			Queries("select", "").Queries("select-type", "2").
			Name("SelectObjectContent")
		// GetObjectRetention
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectRetentionHandler)).
			Queries("retention", "").
			Name("GetObjectRetention")
		// GetObjectLegalHold
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectLegalHoldHandler)).
			Queries("legal-hold", "").
			Name("GetObjectLegalHold")
		// GetObjectAttributes
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectAttributesHandler)).
			Queries("attributes", "").
			Name("GetObjectAttributes")
		// GetObject
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.GetObjectHandler)).
			Name("GetObject")
		// CopyObject
		bucket.Methods(http.MethodPut).Path("/{object:.+}").Headers(hdrAmzCopySource, "").HandlerFunc(
			m.Handle(h.CopyObjectHandler)).
			Name("CopyObject")
		// PutObjectRetention
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.PutObjectRetentionHandler)).
			Queries("retention", "").
			Name("PutObjectRetention")
		// PutObjectLegalHold
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.PutObjectLegalHoldHandler)).
			Queries("legal-hold", "").
			Name("PutObjectLegalHold")

		// PutObject
		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.PutObjectHandler)).
			Name("PutObject")
		// DeleteObject
		bucket.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(
			m.Handle(h.DeleteObjectHandler)).
			Name("DeleteObject")

		// Bucket operations
		// GetBucketLocation
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketLocationHandler)).
			Queries("location", "").
			Name("GetBucketLocation")
		// GetBucketPolicy
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketPolicyHandler)).
			Queries("policy", "").
			Name("GetBucketPolicy")
		// GetBucketLifecycle
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketLifecycleHandler)).
			Queries("lifecycle", "").
			Name("GetBucketLifecycle")
		// GetBucketEncryption
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketEncryptionHandler)).
			Queries("encryption", "").
			Name("GetBucketEncryption")
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketCorsHandler)).
			Queries("cors", "").
			Name("GetBucketCors")
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketCorsHandler)).
			Queries("cors", "").
			Name("PutBucketCors")
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketCorsHandler)).
			Queries("cors", "").
			Name("DeleteBucketCors")
		// Dummy Bucket Calls
		// GetBucketACL -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketACLHandler)).
			Queries("acl", "").
			Name("GetBucketACL")
		// PutBucketACL -- this is a dummy call.
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketACLHandler)).
			Queries("acl", "").
			Name("PutBucketACL")
		// GetBucketWebsiteHandler -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketWebsiteHandler)).
			Queries("website", "").
			Name("GetBucketWebsite")
		// GetBucketAccelerateHandler -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketAccelerateHandler)).
			Queries("accelerate", "").
			Name("GetBucketAccelerate")
		// GetBucketRequestPaymentHandler -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketRequestPaymentHandler)).
			Queries("requestPayment", "").
			Name("GetBucketRequestPayment")
		// GetBucketLoggingHandler -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketLoggingHandler)).
			Queries("logging", "").
			Name("GetBucketLogging")
		// GetBucketReplicationHandler -- this is a dummy call.
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketReplicationHandler)).
			Queries("replication", "").
			Name("GetBucketReplication")
		// GetBucketTaggingHandler
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketTaggingHandler)).
			Queries("tagging", "").
			Name("GetBucketTagging")
		// DeleteBucketWebsiteHandler
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketWebsiteHandler)).
			Queries("website", "").
			Name("DeleteBucketWebsite")
		// DeleteBucketTaggingHandler
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketTaggingHandler)).
			Queries("tagging", "").
			Name("DeleteBucketTagging")

		// GetBucketObjectLockConfig
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketObjectLockConfigHandler)).
			Queries("object-lock", "").
			Name("GetBucketObjectLockConfig")
		// GetBucketVersioning
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketVersioningHandler)).
			Queries("versioning", "").
			Name("GetBucketVersioning")
		// GetBucketNotification
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.GetBucketNotificationHandler)).
			Queries("notification", "").
			Name("GetBucketNotification")
		// ListenBucketNotification
		bucket.Methods(http.MethodGet).HandlerFunc(h.ListenBucketNotificationHandler).
			Queries("events", "{events:.*}").
			Name("ListenBucketNotification")
		// ListObjectsV2M
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.ListObjectsV2MHandler)).
			Queries("list-type", "2", "metadata", "true").
			Name("ListObjectsV2M")
		// ListObjectsV2
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.ListObjectsV2Handler)).
			Queries("list-type", "2").
			Name("ListObjectsV2")
		// ListBucketVersions
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.ListBucketObjectVersionsHandler)).
			Queries("versions", "").
			Name("ListBucketVersions")
		// ListObjectsV1 (Legacy)
		bucket.Methods(http.MethodGet).HandlerFunc(
			m.Handle(h.ListObjectsV1Handler)).
			Name("ListObjectsV1")
		// PutBucketLifecycle
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketLifecycleHandler)).
			Queries("lifecycle", "").
			Name("PutBucketLifecycle")
		// PutBucketEncryption
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketEncryptionHandler)).
			Queries("encryption", "").
			Name("PutBucketEncryption")

		// PutBucketPolicy
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketPolicyHandler)).
			Queries("policy", "").
			Name("PutBucketPolicy")

		// PutBucketObjectLockConfig
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketObjectLockConfigHandler)).
			Queries("object-lock", "").
			Name("PutBucketObjectLockConfig")
		// PutBucketTaggingHandler
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketTaggingHandler)).
			Queries("tagging", "").
			Name("PutBucketTagging")
		// PutBucketVersioning
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketVersioningHandler)).
			Queries("versioning", "").
			Name("PutBucketVersioning")
		// PutBucketNotification
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.PutBucketNotificationHandler)).
			Queries("notification", "").
			Name("PutBucketNotification")
		// CreateBucket
		bucket.Methods(http.MethodPut).HandlerFunc(
			m.Handle(h.CreateBucketHandler)).
			Name("CreateBucket")
		// HeadBucket
		bucket.Methods(http.MethodHead).HandlerFunc(
			m.Handle(h.HeadBucketHandler)).
			Name("HeadBucket")
		// PostPolicy
		bucket.Methods(http.MethodPost).HeadersRegexp(hdrContentType, "multipart/form-data*").HandlerFunc(
			m.Handle(h.PostObject)).
			Name("PostObject")
		// DeleteMultipleObjects
		bucket.Methods(http.MethodPost).HandlerFunc(
			m.Handle(h.DeleteMultipleObjectsHandler)).
			Queries("delete", "").
			Name("DeleteMultipleObjects")
		// DeleteBucketPolicy
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketPolicyHandler)).
			Queries("policy", "").
			Name("DeleteBucketPolicy")
		// DeleteBucketLifecycle
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketLifecycleHandler)).
			Queries("lifecycle", "").
			Name("DeleteBucketLifecycle")
		// DeleteBucketEncryption
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketEncryptionHandler)).
			Queries("encryption", "").
			Name("DeleteBucketEncryption")
		// DeleteBucket
		bucket.Methods(http.MethodDelete).HandlerFunc(
			m.Handle(h.DeleteBucketHandler)).
			Name("DeleteBucket")
	}
	// Root operation

	// ListBuckets
	api.Methods(http.MethodGet).Path(SlashSeparator).HandlerFunc(
		m.Handle(h.ListBucketsHandler)).
		Name("ListBuckets")

	// S3 browser with signature v4 adds '//' for ListBuckets request, so rather
	// than failing with UnknownAPIRequest we simply handle it for now.
	api.Methods(http.MethodGet).Path(SlashSeparator + SlashSeparator).HandlerFunc(
		m.Handle(h.ListBucketsHandler)).
		Name("ListBuckets")
}
