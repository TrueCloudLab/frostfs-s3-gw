package handler

import (
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/auth"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"go.uber.org/zap"
)

type copyObjectArgs struct {
	Conditional       *conditionalArgs
	MetadataDirective string
	TaggingDirective  string
}

const (
	replaceDirective = "REPLACE"
	copyDirective    = "COPY"
)

var copySourceMatcher = auth.NewRegexpMatcher(regexp.MustCompile(`^/?(?P<bucket_name>[a-z0-9.\-]{3,63})/(?P<object_name>.+)$`))

// path2BucketObject returns a bucket and an object.
func path2BucketObject(path string) (string, string, error) {
	matches := copySourceMatcher.GetSubmatches(path)
	if len(matches) != 2 {
		return "", "", errors.GetAPIError(errors.ErrInvalidRequest)
	}

	return matches["bucket_name"], matches["object_name"], nil
}

func (h *handler) CopyObjectHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err              error
		versionID        string
		metadata         map[string]string
		tagSet           map[string]string
		sessionTokenEACL *session.Container

		reqInfo = api.GetReqInfo(r.Context())

		containsACL = containsACLHeaders(r)
	)

	src := r.Header.Get(api.AmzCopySource)
	// Check https://docs.aws.amazon.com/AmazonS3/latest/dev/ObjectVersioning.html
	// Regardless of whether you have enabled versioning, each object in your bucket
	// has a version ID. If you have not enabled versioning, Amazon S3 sets the value
	// of the version ID to null. If you have enabled versioning, Amazon S3 assigns a
	// unique version ID value for the object.
	if u, err := url.Parse(src); err == nil {
		versionID = u.Query().Get(api.QueryVersionID)
		src = u.Path
	}

	srcBucket, srcObject, err := path2BucketObject(src)
	if err != nil {
		h.logAndSendError(w, "invalid source copy", reqInfo, err)
		return
	}

	srcObjPrm := &layer.HeadObjectParams{
		Object:    srcObject,
		VersionID: versionID,
	}

	if srcObjPrm.BktInfo, err = h.getBucketAndCheckOwner(r, srcBucket, api.AmzSourceExpectedBucketOwner); err != nil {
		h.logAndSendError(w, "couldn't get source bucket", reqInfo, err)
		return
	}

	dstBktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "couldn't get target bucket", reqInfo, err)
		return
	}

	settings, err := h.obj.GetBucketSettings(r.Context(), dstBktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket settings", reqInfo, err)
		return
	}

	if containsACL {
		if sessionTokenEACL, err = getSessionTokenSetEACL(r.Context()); err != nil {
			h.logAndSendError(w, "could not get eacl session token from a box", reqInfo, err)
			return
		}
	}

	extendedSrcObjInfo, err := h.obj.GetExtendedObjectInfo(r.Context(), srcObjPrm)
	if err != nil {
		h.logAndSendError(w, "could not find object", reqInfo, err)
		return
	}
	srcObjInfo := extendedSrcObjInfo.ObjectInfo

	args, err := parseCopyObjectArgs(r.Header)
	if err != nil {
		h.logAndSendError(w, "could not parse request params", reqInfo, err)
		return
	}

	if isCopyingToItselfForbidden(reqInfo, srcBucket, srcObject, settings, args) {
		h.logAndSendError(w, "copying to itself without changing anything", reqInfo, errors.GetAPIError(errors.ErrInvalidCopyDest))
		return
	}

	if args.MetadataDirective == replaceDirective {
		metadata = parseMetadata(r)
	}

	if args.TaggingDirective == replaceDirective {
		tagSet, err = parseTaggingHeader(r.Header)
		if err != nil {
			h.logAndSendError(w, "could not parse tagging header", reqInfo, err)
			return
		}
	} else {
		tagPrm := &layer.GetObjectTaggingParams{
			ObjectVersion: &layer.ObjectVersion{
				BktInfo:    srcObjPrm.BktInfo,
				ObjectName: srcObject,
				VersionID:  srcObjInfo.VersionID(),
			},
			NodeVersion: extendedSrcObjInfo.NodeVersion,
		}

		_, tagSet, err = h.obj.GetObjectTagging(r.Context(), tagPrm)
		if err != nil {
			h.logAndSendError(w, "could not get object tagging", reqInfo, err)
			return
		}
	}

	encryptionParams, err := formEncryptionParams(r)
	if err != nil {
		h.logAndSendError(w, "invalid sse headers", reqInfo, err)
		return
	}

	if err = encryptionParams.MatchObjectEncryption(layer.FormEncryptionInfo(srcObjInfo.Headers)); err != nil {
		h.logAndSendError(w, "encryption doesn't match object", reqInfo, errors.GetAPIError(errors.ErrBadRequest), zap.Error(err))
		return
	}

	if err = checkPreconditions(srcObjInfo, args.Conditional); err != nil {
		h.logAndSendError(w, "precondition failed", reqInfo, errors.GetAPIError(errors.ErrPreconditionFailed))
		return
	}

	if metadata == nil {
		if len(srcObjInfo.ContentType) > 0 {
			srcObjInfo.Headers[api.ContentType] = srcObjInfo.ContentType
		}
		metadata = srcObjInfo.Headers
	} else if contentType := r.Header.Get(api.ContentType); len(contentType) > 0 {
		metadata[api.ContentType] = contentType
	}

	copiesNumber, err := getCopiesNumberOrDefault(metadata, h.cfg.CopiesNumber)
	if err != nil {
		h.logAndSendError(w, "invalid copies number", reqInfo, err)
		return
	}

	params := &layer.CopyObjectParams{
		SrcObject:   srcObjInfo,
		ScrBktInfo:  srcObjPrm.BktInfo,
		DstBktInfo:  dstBktInfo,
		DstObject:   reqInfo.ObjectName,
		SrcSize:     srcObjInfo.Size,
		Header:      metadata,
		Encryption:  encryptionParams,
		CopiesNuber: copiesNumber,
	}

	params.Lock, err = formObjectLock(r.Context(), dstBktInfo, settings.LockConfiguration, r.Header)
	if err != nil {
		h.logAndSendError(w, "could not form object lock", reqInfo, err)
		return
	}

	additional := []zap.Field{zap.String("src_bucket_name", srcBucket), zap.String("src_object_name", srcObject)}
	extendedDstObjInfo, err := h.obj.CopyObject(r.Context(), params)
	if err != nil {
		h.logAndSendError(w, "couldn't copy object", reqInfo, err, additional...)
		return
	}
	dstObjInfo := extendedDstObjInfo.ObjectInfo

	if err = api.EncodeToResponse(w, &CopyObjectResponse{LastModified: dstObjInfo.Created.UTC().Format(time.RFC3339), ETag: dstObjInfo.HashSum}); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err, additional...)
		return
	}

	if containsACL {
		newEaclTable, err := h.getNewEAclTable(r, dstBktInfo, dstObjInfo)
		if err != nil {
			h.logAndSendError(w, "could not get new eacl table", reqInfo, err)
			return
		}

		p := &layer.PutBucketACLParams{
			BktInfo:      dstBktInfo,
			EACL:         newEaclTable,
			SessionToken: sessionTokenEACL,
		}

		if err = h.obj.PutBucketACL(r.Context(), p); err != nil {
			h.logAndSendError(w, "could not put bucket acl", reqInfo, err)
			return
		}
	}

	if tagSet != nil {
		tagPrm := &layer.PutObjectTaggingParams{
			ObjectVersion: &layer.ObjectVersion{
				BktInfo:    dstBktInfo,
				ObjectName: reqInfo.ObjectName,
				VersionID:  dstObjInfo.VersionID(),
			},
			TagSet:      tagSet,
			NodeVersion: extendedDstObjInfo.NodeVersion,
		}
		if _, err = h.obj.PutObjectTagging(r.Context(), tagPrm); err != nil {
			h.logAndSendError(w, "could not upload object tagging", reqInfo, err)
			return
		}
	}

	h.log.Info("object is copied",
		zap.String("bucket", dstObjInfo.Bucket),
		zap.String("object", dstObjInfo.Name),
		zap.Stringer("object_id", dstObjInfo.ID))

	s := &SendNotificationParams{
		Event:            EventObjectCreatedCopy,
		NotificationInfo: data.NotificationInfoFromObject(dstObjInfo),
		BktInfo:          dstBktInfo,
		ReqInfo:          reqInfo,
	}
	if err = h.sendNotifications(r.Context(), s); err != nil {
		h.log.Error("couldn't send notification: %w", zap.Error(err))
	}

	if encryptionParams.Enabled() {
		addSSECHeaders(w.Header(), r.Header)
	}
}

func isCopyingToItselfForbidden(reqInfo *api.ReqInfo, srcBucket string, srcObject string, settings *data.BucketSettings, args *copyObjectArgs) bool {
	if reqInfo.BucketName != srcBucket || reqInfo.ObjectName != srcObject {
		return false
	}

	if !settings.Unversioned() {
		return false
	}

	return args.MetadataDirective != replaceDirective
}

func parseCopyObjectArgs(headers http.Header) (*copyObjectArgs, error) {
	var err error
	args := &conditionalArgs{
		IfMatch:     headers.Get(api.AmzCopyIfMatch),
		IfNoneMatch: headers.Get(api.AmzCopyIfNoneMatch),
	}

	if args.IfModifiedSince, err = parseHTTPTime(headers.Get(api.AmzCopyIfModifiedSince)); err != nil {
		return nil, err
	}
	if args.IfUnmodifiedSince, err = parseHTTPTime(headers.Get(api.AmzCopyIfUnmodifiedSince)); err != nil {
		return nil, err
	}

	copyArgs := &copyObjectArgs{Conditional: args}

	copyArgs.MetadataDirective = headers.Get(api.AmzMetadataDirective)
	if !isValidDirective(copyArgs.MetadataDirective) {
		return nil, errors.GetAPIError(errors.ErrInvalidMetadataDirective)
	}

	copyArgs.TaggingDirective = headers.Get(api.AmzTaggingDirective)
	if !isValidDirective(copyArgs.TaggingDirective) {
		return nil, errors.GetAPIError(errors.ErrInvalidTaggingDirective)
	}

	return copyArgs, nil
}

func isValidDirective(directive string) bool {
	return len(directive) == 0 ||
		directive == replaceDirective || directive == copyDirective
}
