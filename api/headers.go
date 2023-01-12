package api

// Standard S3 HTTP request/response constants.
const (
	MetadataPrefix              = "X-Amz-Meta-"
	FrostFSSystemMetadataPrefix = "S3-"
	AmzMetadataDirective        = "X-Amz-Metadata-Directive"
	AmzTaggingDirective         = "X-Amz-Tagging-Directive"
	AmzVersionID                = "X-Amz-Version-Id"
	AmzTaggingCount             = "X-Amz-Tagging-Count"
	AmzTagging                  = "X-Amz-Tagging"
	AmzDeleteMarker             = "X-Amz-Delete-Marker"
	AmzCopySource               = "X-Amz-Copy-Source"
	AmzCopySourceRange          = "X-Amz-Copy-Source-Range"
	AmzDate                     = "X-Amz-Date"

	LastModified       = "Last-Modified"
	Date               = "Date"
	ETag               = "ETag"
	ContentType        = "Content-Type"
	ContentMD5         = "Content-Md5"
	ContentEncoding    = "Content-Encoding"
	Expires            = "Expires"
	ContentLength      = "Content-Length"
	ContentLanguage    = "Content-Language"
	ContentRange       = "Content-Range"
	Connection         = "Connection"
	AcceptRanges       = "Accept-Ranges"
	AmzBucketRegion    = "X-Amz-Bucket-Region"
	ServerInfo         = "Server"
	RetryAfter         = "Retry-After"
	Location           = "Location"
	CacheControl       = "Cache-Control"
	ContentDisposition = "Content-Disposition"
	Authorization      = "Authorization"
	Action             = "Action"
	IfModifiedSince    = "If-Modified-Since"
	IfUnmodifiedSince  = "If-Unmodified-Since"
	IfMatch            = "If-Match"
	IfNoneMatch        = "If-None-Match"

	AmzCopyIfModifiedSince       = "X-Amz-Copy-Source-If-Modified-Since"
	AmzCopyIfUnmodifiedSince     = "X-Amz-Copy-Source-If-Unmodified-Since"
	AmzCopyIfMatch               = "X-Amz-Copy-Source-If-Match"
	AmzCopyIfNoneMatch           = "X-Amz-Copy-Source-If-None-Match"
	AmzACL                       = "X-Amz-Acl"
	AmzGrantFullControl          = "X-Amz-Grant-Full-Control"
	AmzGrantRead                 = "X-Amz-Grant-Read"
	AmzGrantWrite                = "X-Amz-Grant-Write"
	AmzExpectedBucketOwner       = "X-Amz-Expected-Bucket-Owner"
	AmzSourceExpectedBucketOwner = "X-Amz-Source-Expected-Bucket-Owner"
	AmzBucketObjectLockEnabled   = "X-Amz-Bucket-Object-Lock-Enabled"
	AmzObjectLockLegalHold       = "X-Amz-Object-Lock-Legal-Hold"
	AmzObjectLockMode            = "X-Amz-Object-Lock-Mode"
	AmzObjectLockRetainUntilDate = "X-Amz-Object-Lock-Retain-Until-Date"
	AmzBypassGovernanceRetention = "X-Amz-Bypass-Governance-Retention"
	AmzObjectAttributes          = "X-Amz-Object-Attributes"
	AmzMaxParts                  = "X-Amz-Max-Parts"
	AmzPartNumberMarker          = "X-Amz-Part-Number-Marker"
	AmzExpiration                = "X-Amz-Expiration"

	AmzServerSideEncryptionCustomerAlgorithm = "x-amz-server-side-encryption-customer-algorithm"
	AmzServerSideEncryptionCustomerKey       = "x-amz-server-side-encryption-customer-key"
	AmzServerSideEncryptionCustomerKeyMD5    = "x-amz-server-side-encryption-customer-key-MD5"

	ContainerID = "X-Container-Id"

	AccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	AccessControlAllowMethods     = "Access-Control-Allow-Methods"
	AccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	AccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	AccessControlMaxAge           = "Access-Control-Max-Age"
	AccessControlAllowCredentials = "Access-Control-Allow-Credentials"

	Origin                      = "Origin"
	AccessControlRequestMethod  = "Access-Control-Request-Method"
	AccessControlRequestHeaders = "Access-Control-Request-Headers"

	Vary = "Vary"

	DefaultLocationConstraint = "default"
)

// S3 request query params.
const (
	QueryVersionID = "versionId"
)

// ResponseModifiers maps response modifies headers to regular headers.
var ResponseModifiers = map[string]string{
	"response-content-type":        ContentType,
	"response-content-language":    ContentLanguage,
	"response-expires":             Expires,
	"response-cache-control":       CacheControl,
	"response-content-disposition": ContentDisposition,
	"response-content-encoding":    ContentEncoding,
}

var SystemMetadata = map[string]struct{}{
	Date:               {},
	CacheControl:       {},
	ContentDisposition: {},
	ContentLength:      {},
	ContentType:        {},
	LastModified:       {},
	ETag:               {},
}
