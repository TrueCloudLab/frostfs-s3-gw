package layer

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer/encryption"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/TrueCloudLab/frostfs-sdk-go/netmap"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/nats-io/nats.go"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"go.uber.org/zap"
)

type (
	EventListener interface {
		Subscribe(context.Context, string, MsgHandler) error
		Listen(context.Context)
	}

	MsgHandler interface {
		HandleMessage(context.Context, *nats.Msg) error
	}

	MsgHandlerFunc func(context.Context, *nats.Msg) error

	BucketResolver interface {
		Resolve(ctx context.Context, name string) (cid.ID, error)
	}

	layer struct {
		frostFS     FrostFS
		log         *zap.Logger
		anonKey     AnonymousKey
		resolver    BucketResolver
		ncontroller EventListener
		cache       *Cache
		treeService TreeService
	}

	Config struct {
		ChainAddress string
		Caches       *CachesConfig
		AnonKey      AnonymousKey
		Resolver     BucketResolver
		TreeService  TreeService
	}

	// AnonymousKey contains data for anonymous requests.
	AnonymousKey struct {
		Key *keys.PrivateKey
	}

	// GetObjectParams stores object get request parameters.
	GetObjectParams struct {
		Range      *RangeParams
		ObjectInfo *data.ObjectInfo
		BucketInfo *data.BucketInfo
		Writer     io.Writer
		Encryption encryption.Params
	}

	// HeadObjectParams stores object head request parameters.
	HeadObjectParams struct {
		BktInfo   *data.BucketInfo
		Object    string
		VersionID string
	}

	// ObjectVersion stores object version info.
	ObjectVersion struct {
		BktInfo               *data.BucketInfo
		ObjectName            string
		VersionID             string
		NoErrorOnDeleteMarker bool
	}

	// RangeParams stores range header request parameters.
	RangeParams struct {
		Start uint64
		End   uint64
	}

	// PutObjectParams stores object put request parameters.
	PutObjectParams struct {
		BktInfo      *data.BucketInfo
		Object       string
		Size         int64
		Reader       io.Reader
		Header       map[string]string
		Lock         *data.ObjectLock
		Encryption   encryption.Params
		CopiesNumber uint32
	}

	DeleteObjectParams struct {
		BktInfo  *data.BucketInfo
		Objects  []*VersionedObject
		Settings *data.BucketSettings
	}

	// PutSettingsParams stores object copy request parameters.
	PutSettingsParams struct {
		BktInfo  *data.BucketInfo
		Settings *data.BucketSettings
	}

	// PutCORSParams stores PutCORS request parameters.
	PutCORSParams struct {
		BktInfo      *data.BucketInfo
		Reader       io.Reader
		CopiesNumber uint32
	}

	// CopyObjectParams stores object copy request parameters.
	CopyObjectParams struct {
		SrcObject   *data.ObjectInfo
		ScrBktInfo  *data.BucketInfo
		DstBktInfo  *data.BucketInfo
		DstObject   string
		SrcSize     int64
		Header      map[string]string
		Range       *RangeParams
		Lock        *data.ObjectLock
		Encryption  encryption.Params
		CopiesNuber uint32
	}
	// CreateBucketParams stores bucket create request parameters.
	CreateBucketParams struct {
		Name                     string
		Policy                   netmap.PlacementPolicy
		EACL                     *eacl.Table
		SessionContainerCreation *session.Container
		SessionEACL              *session.Container
		LocationConstraint       string
		ObjectLockEnabled        bool
	}
	// PutBucketACLParams stores put bucket acl request parameters.
	PutBucketACLParams struct {
		BktInfo      *data.BucketInfo
		EACL         *eacl.Table
		SessionToken *session.Container
	}
	// DeleteBucketParams stores delete bucket request parameters.
	DeleteBucketParams struct {
		BktInfo      *data.BucketInfo
		SessionToken *session.Container
	}

	// ListObjectVersionsParams stores list objects versions parameters.
	ListObjectVersionsParams struct {
		BktInfo         *data.BucketInfo
		Delimiter       string
		KeyMarker       string
		MaxKeys         int
		Prefix          string
		VersionIDMarker string
		Encode          string
	}

	// VersionedObject stores info about objects to delete.
	VersionedObject struct {
		Name              string
		VersionID         string
		DeleteMarkVersion string
		DeleteMarkerEtag  string
		Error             error
	}

	// Client provides S3 API client interface.
	Client interface {
		Initialize(ctx context.Context, c EventListener) error
		EphemeralKey() *keys.PublicKey

		GetBucketSettings(ctx context.Context, bktInfo *data.BucketInfo) (*data.BucketSettings, error)
		PutBucketSettings(ctx context.Context, p *PutSettingsParams) error

		PutBucketCORS(ctx context.Context, p *PutCORSParams) error
		GetBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) (*data.CORSConfiguration, error)
		DeleteBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) error

		ListBuckets(ctx context.Context) ([]*data.BucketInfo, error)
		GetBucketInfo(ctx context.Context, name string) (*data.BucketInfo, error)
		GetBucketACL(ctx context.Context, bktInfo *data.BucketInfo) (*BucketACL, error)
		PutBucketACL(ctx context.Context, p *PutBucketACLParams) error
		CreateBucket(ctx context.Context, p *CreateBucketParams) (*data.BucketInfo, error)
		DeleteBucket(ctx context.Context, p *DeleteBucketParams) error

		GetObject(ctx context.Context, p *GetObjectParams) error
		GetObjectInfo(ctx context.Context, p *HeadObjectParams) (*data.ObjectInfo, error)
		GetExtendedObjectInfo(ctx context.Context, p *HeadObjectParams) (*data.ExtendedObjectInfo, error)

		GetLockInfo(ctx context.Context, obj *ObjectVersion) (*data.LockInfo, error)
		PutLockInfo(ctx context.Context, p *PutLockInfoParams) error

		GetBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) (map[string]string, error)
		PutBucketTagging(ctx context.Context, bktInfo *data.BucketInfo, tagSet map[string]string) error
		DeleteBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) error

		GetObjectTagging(ctx context.Context, p *GetObjectTaggingParams) (string, map[string]string, error)
		PutObjectTagging(ctx context.Context, p *PutObjectTaggingParams) (*data.NodeVersion, error)
		DeleteObjectTagging(ctx context.Context, p *ObjectVersion) (*data.NodeVersion, error)

		PutObject(ctx context.Context, p *PutObjectParams) (*data.ExtendedObjectInfo, error)

		CopyObject(ctx context.Context, p *CopyObjectParams) (*data.ExtendedObjectInfo, error)

		ListObjectsV1(ctx context.Context, p *ListObjectsParamsV1) (*ListObjectsInfoV1, error)
		ListObjectsV2(ctx context.Context, p *ListObjectsParamsV2) (*ListObjectsInfoV2, error)
		ListObjectVersions(ctx context.Context, p *ListObjectVersionsParams) (*ListObjectVersionsInfo, error)

		DeleteObjects(ctx context.Context, p *DeleteObjectParams) []*VersionedObject

		CreateMultipartUpload(ctx context.Context, p *CreateMultipartParams) error
		CompleteMultipartUpload(ctx context.Context, p *CompleteMultipartParams) (*UploadData, *data.ExtendedObjectInfo, error)
		UploadPart(ctx context.Context, p *UploadPartParams) (string, error)
		UploadPartCopy(ctx context.Context, p *UploadCopyParams) (*data.ObjectInfo, error)
		ListMultipartUploads(ctx context.Context, p *ListMultipartUploadsParams) (*ListMultipartUploadsInfo, error)
		AbortMultipartUpload(ctx context.Context, p *UploadInfoParams) error
		ListParts(ctx context.Context, p *ListPartsParams) (*ListPartsInfo, error)

		PutBucketNotificationConfiguration(ctx context.Context, p *PutBucketNotificationConfigurationParams) error
		GetBucketNotificationConfiguration(ctx context.Context, bktInfo *data.BucketInfo) (*data.NotificationConfiguration, error)

		// Compound methods for optimizations

		// GetObjectTaggingAndLock unifies GetObjectTagging and GetLock methods in single tree service invocation.
		GetObjectTaggingAndLock(ctx context.Context, p *ObjectVersion, nodeVersion *data.NodeVersion) (map[string]string, *data.LockInfo, error)
	}
)

const (
	tagPrefix = "S3-Tag-"

	AESEncryptionAlgorithm       = "AES256"
	AESKeySize                   = 32
	AttributeEncryptionAlgorithm = api.FrostFSSystemMetadataPrefix + "Algorithm"
	AttributeDecryptedSize       = api.FrostFSSystemMetadataPrefix + "Decrypted-Size"
	AttributeHMACSalt            = api.FrostFSSystemMetadataPrefix + "HMAC-Salt"
	AttributeHMACKey             = api.FrostFSSystemMetadataPrefix + "HMAC-Key"

	AttributeFrostfsCopiesNumber = "frostfs-copies-number" // such format to match X-Amz-Meta-Frostfs-Copies-Number header
)

func (t *VersionedObject) String() string {
	return t.Name + ":" + t.VersionID
}

func (f MsgHandlerFunc) HandleMessage(ctx context.Context, msg *nats.Msg) error {
	return f(ctx, msg)
}

// NewLayer creates an instance of a layer. It checks credentials
// and establishes gRPC connection with the node.
func NewLayer(log *zap.Logger, frostFS FrostFS, config *Config) Client {
	return &layer{
		frostFS:     frostFS,
		log:         log,
		anonKey:     config.AnonKey,
		resolver:    config.Resolver,
		cache:       NewCache(config.Caches),
		treeService: config.TreeService,
	}
}

func (n *layer) EphemeralKey() *keys.PublicKey {
	return n.anonKey.Key.PublicKey()
}

func (n *layer) Initialize(ctx context.Context, c EventListener) error {
	if n.IsNotificationEnabled() {
		return fmt.Errorf("already initialized")
	}

	// todo add notification handlers (e.g. for lifecycles)

	c.Listen(ctx)

	n.ncontroller = c
	return nil
}

func (n *layer) IsNotificationEnabled() bool {
	return n.ncontroller != nil
}

// IsAuthenticatedRequest checks if access box exists in the current request.
func IsAuthenticatedRequest(ctx context.Context) bool {
	_, ok := ctx.Value(api.BoxData).(*accessbox.Box)
	return ok
}

// TimeNow returns client time from request or time.Now().
func TimeNow(ctx context.Context) time.Time {
	if now, ok := ctx.Value(api.ClientTime).(time.Time); ok {
		return now
	}

	return time.Now()
}

// Owner returns owner id from BearerToken (context) or from client owner.
func (n *layer) Owner(ctx context.Context) user.ID {
	if bd, ok := ctx.Value(api.BoxData).(*accessbox.Box); ok && bd != nil && bd.Gate != nil && bd.Gate.BearerToken != nil {
		return bearer.ResolveIssuer(*bd.Gate.BearerToken)
	}

	var ownerID user.ID
	user.IDFromKey(&ownerID, (ecdsa.PublicKey)(*n.EphemeralKey()))

	return ownerID
}

func (n *layer) prepareAuthParameters(ctx context.Context, prm *PrmAuth, bktOwner user.ID) {
	if bd, ok := ctx.Value(api.BoxData).(*accessbox.Box); ok && bd != nil && bd.Gate != nil && bd.Gate.BearerToken != nil {
		if bktOwner.Equals(bearer.ResolveIssuer(*bd.Gate.BearerToken)) {
			prm.BearerToken = bd.Gate.BearerToken
			return
		}
	}

	prm.PrivateKey = &n.anonKey.Key.PrivateKey
}

// GetBucketInfo returns bucket info by name.
func (n *layer) GetBucketInfo(ctx context.Context, name string) (*data.BucketInfo, error) {
	name, err := url.QueryUnescape(name)
	if err != nil {
		return nil, fmt.Errorf("unescape bucket name: %w", err)
	}

	if bktInfo := n.cache.GetBucket(name); bktInfo != nil {
		return bktInfo, nil
	}

	containerID, err := n.ResolveBucket(ctx, name)
	if err != nil {
		n.log.Debug("bucket not found", zap.Error(err))
		return nil, errors.GetAPIError(errors.ErrNoSuchBucket)
	}

	return n.containerInfo(ctx, containerID)
}

// GetBucketACL returns bucket acl info by name.
func (n *layer) GetBucketACL(ctx context.Context, bktInfo *data.BucketInfo) (*BucketACL, error) {
	eACL, err := n.GetContainerEACL(ctx, bktInfo.CID)
	if err != nil {
		return nil, fmt.Errorf("get container eacl: %w", err)
	}

	return &BucketACL{
		Info: bktInfo,
		EACL: eACL,
	}, nil
}

// PutBucketACL puts bucket acl by name.
func (n *layer) PutBucketACL(ctx context.Context, param *PutBucketACLParams) error {
	return n.setContainerEACLTable(ctx, param.BktInfo.CID, param.EACL, param.SessionToken)
}

// ListBuckets returns all user containers. The name of the bucket is a container
// id. Timestamp is omitted since it is not saved in frostfs container.
func (n *layer) ListBuckets(ctx context.Context) ([]*data.BucketInfo, error) {
	return n.containerList(ctx)
}

// GetObject from storage.
func (n *layer) GetObject(ctx context.Context, p *GetObjectParams) error {
	var params getParams

	params.oid = p.ObjectInfo.ID
	params.bktInfo = p.BucketInfo

	var decReader *encryption.Decrypter
	if p.Encryption.Enabled() {
		var err error
		decReader, err = getDecrypter(p)
		if err != nil {
			return fmt.Errorf("creating decrypter: %w", err)
		}
		params.off = decReader.EncryptedOffset()
		params.ln = decReader.EncryptedLength()
	} else {
		if p.Range != nil {
			if p.Range.Start > p.Range.End {
				panic("invalid range")
			}
			params.ln = p.Range.End - p.Range.Start + 1
			params.off = p.Range.Start
		}
	}

	payload, err := n.initObjectPayloadReader(ctx, params)
	if err != nil {
		return fmt.Errorf("init object payload reader: %w", err)
	}

	bufSize := uint64(32 * 1024) // configure?
	if params.ln != 0 && params.ln < bufSize {
		bufSize = params.ln
	}

	// alloc buffer for copying
	buf := make([]byte, bufSize) // sync-pool it?

	r := payload
	if decReader != nil {
		if err = decReader.SetReader(payload); err != nil {
			return fmt.Errorf("set reader to decrypter: %w", err)
		}
		r = io.LimitReader(decReader, int64(decReader.DecryptedLength()))
	}

	// copy full payload
	written, err := io.CopyBuffer(p.Writer, r, buf)
	if err != nil {
		if decReader != nil {
			return fmt.Errorf("copy object payload written: '%d', decLength: '%d', params.ln: '%d' : %w", written, decReader.DecryptedLength(), params.ln, err)
		}
		return fmt.Errorf("copy object payload written: '%d': %w", written, err)
	}

	return nil
}

func getDecrypter(p *GetObjectParams) (*encryption.Decrypter, error) {
	var encRange *encryption.Range
	if p.Range != nil {
		encRange = &encryption.Range{Start: p.Range.Start, End: p.Range.End}
	}

	header := p.ObjectInfo.Headers[UploadCompletedParts]
	if len(header) == 0 {
		return encryption.NewDecrypter(p.Encryption, uint64(p.ObjectInfo.Size), encRange)
	}

	decryptedObjectSize, err := strconv.ParseUint(p.ObjectInfo.Headers[AttributeDecryptedSize], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse decrypted size: %w", err)
	}

	splits := strings.Split(header, ",")
	sizes := make([]uint64, len(splits))
	for i, splitInfo := range splits {
		part, err := ParseCompletedPartHeader(splitInfo)
		if err != nil {
			return nil, fmt.Errorf("parse completed part: %w", err)
		}
		sizes[i] = uint64(part.Size)
	}

	return encryption.NewMultipartDecrypter(p.Encryption, decryptedObjectSize, sizes, encRange)
}

// GetObjectInfo returns meta information about the object.
func (n *layer) GetObjectInfo(ctx context.Context, p *HeadObjectParams) (*data.ObjectInfo, error) {
	extendedObjectInfo, err := n.GetExtendedObjectInfo(ctx, p)
	if err != nil {
		return nil, err
	}

	return extendedObjectInfo.ObjectInfo, nil
}

// GetExtendedObjectInfo returns meta information and corresponding info from the tree service about the object.
func (n *layer) GetExtendedObjectInfo(ctx context.Context, p *HeadObjectParams) (*data.ExtendedObjectInfo, error) {
	var objInfo *data.ExtendedObjectInfo
	var err error

	if len(p.VersionID) == 0 {
		objInfo, err = n.headLastVersionIfNotDeleted(ctx, p.BktInfo, p.Object)
	} else {
		objInfo, err = n.headVersion(ctx, p.BktInfo, p)
	}
	if err != nil {
		return nil, err
	}

	reqInfo := api.GetReqInfo(ctx)
	n.log.Debug("get object",
		zap.String("reqId", reqInfo.RequestID),
		zap.String("bucket", p.BktInfo.Name),
		zap.Stringer("cid", p.BktInfo.CID),
		zap.String("object", objInfo.ObjectInfo.Name),
		zap.Stringer("oid", objInfo.ObjectInfo.ID))

	return objInfo, nil
}

// CopyObject from one bucket into another bucket.
func (n *layer) CopyObject(ctx context.Context, p *CopyObjectParams) (*data.ExtendedObjectInfo, error) {
	pr, pw := io.Pipe()

	go func() {
		err := n.GetObject(ctx, &GetObjectParams{
			ObjectInfo: p.SrcObject,
			Writer:     pw,
			Range:      p.Range,
			BucketInfo: p.ScrBktInfo,
			Encryption: p.Encryption,
		})

		if err = pw.CloseWithError(err); err != nil {
			n.log.Error("could not get object", zap.Error(err))
		}
	}()

	return n.PutObject(ctx, &PutObjectParams{
		BktInfo:      p.DstBktInfo,
		Object:       p.DstObject,
		Size:         p.SrcSize,
		Reader:       pr,
		Header:       p.Header,
		Encryption:   p.Encryption,
		CopiesNumber: p.CopiesNuber,
	})
}

func getRandomOID() (oid.ID, error) {
	b := [32]byte{}
	if _, err := rand.Read(b[:]); err != nil {
		return oid.ID{}, err
	}

	var objID oid.ID
	objID.SetSHA256(b)
	return objID, nil
}

func (n *layer) deleteObject(ctx context.Context, bkt *data.BucketInfo, settings *data.BucketSettings, obj *VersionedObject) *VersionedObject {
	if len(obj.VersionID) != 0 || settings.Unversioned() {
		var nodeVersion *data.NodeVersion
		if nodeVersion, obj.Error = n.getNodeVersionToDelete(ctx, bkt, obj); obj.Error != nil {
			return dismissNotFoundError(obj)
		}

		if obj.DeleteMarkVersion, obj.Error = n.removeOldVersion(ctx, bkt, nodeVersion, obj); obj.Error != nil {
			return obj
		}

		obj.Error = n.treeService.RemoveVersion(ctx, bkt, nodeVersion.ID)
		n.cache.CleanListCacheEntriesContainingObject(obj.Name, bkt.CID)
		return obj
	}

	var newVersion *data.NodeVersion

	if settings.VersioningSuspended() {
		obj.VersionID = data.UnversionedObjectVersionID

		var nodeVersion *data.NodeVersion
		if nodeVersion, obj.Error = n.getNodeVersionToDelete(ctx, bkt, obj); obj.Error != nil {
			return dismissNotFoundError(obj)
		}

		if obj.DeleteMarkVersion, obj.Error = n.removeOldVersion(ctx, bkt, nodeVersion, obj); obj.Error != nil {
			return obj
		}
	}

	randOID, err := getRandomOID()
	if err != nil {
		obj.Error = fmt.Errorf("couldn't get random oid: %w", err)
		return obj
	}

	obj.DeleteMarkVersion = randOID.EncodeToString()

	newVersion = &data.NodeVersion{
		BaseNodeVersion: data.BaseNodeVersion{
			OID:      randOID,
			FilePath: obj.Name,
		},
		DeleteMarker: &data.DeleteMarkerInfo{
			Created: TimeNow(ctx),
			Owner:   n.Owner(ctx),
		},
		IsUnversioned: settings.VersioningSuspended(),
	}

	if _, obj.Error = n.treeService.AddVersion(ctx, bkt, newVersion); obj.Error != nil {
		return obj
	}

	n.cache.DeleteObjectName(bkt.CID, bkt.Name, obj.Name)

	return obj
}

func dismissNotFoundError(obj *VersionedObject) *VersionedObject {
	if errors.IsS3Error(obj.Error, errors.ErrNoSuchKey) ||
		errors.IsS3Error(obj.Error, errors.ErrNoSuchVersion) {
		obj.Error = nil
	}

	return obj
}

func (n *layer) getNodeVersionToDelete(ctx context.Context, bkt *data.BucketInfo, obj *VersionedObject) (*data.NodeVersion, error) {
	objVersion := &ObjectVersion{
		BktInfo:               bkt,
		ObjectName:            obj.Name,
		VersionID:             obj.VersionID,
		NoErrorOnDeleteMarker: true,
	}

	return n.getNodeVersion(ctx, objVersion)
}

func (n *layer) removeOldVersion(ctx context.Context, bkt *data.BucketInfo, nodeVersion *data.NodeVersion, obj *VersionedObject) (string, error) {
	if nodeVersion.IsDeleteMarker() {
		return obj.VersionID, nil
	}

	return "", n.objectDelete(ctx, bkt, nodeVersion.OID)
}

// DeleteObjects from the storage.
func (n *layer) DeleteObjects(ctx context.Context, p *DeleteObjectParams) []*VersionedObject {
	for i, obj := range p.Objects {
		p.Objects[i] = n.deleteObject(ctx, p.BktInfo, p.Settings, obj)
	}

	return p.Objects
}

func (n *layer) CreateBucket(ctx context.Context, p *CreateBucketParams) (*data.BucketInfo, error) {
	bktInfo, err := n.GetBucketInfo(ctx, p.Name)
	if err != nil {
		if errors.IsS3Error(err, errors.ErrNoSuchBucket) {
			return n.createContainer(ctx, p)
		}
		return nil, err
	}

	if p.SessionContainerCreation != nil && session.IssuedBy(*p.SessionContainerCreation, bktInfo.Owner) {
		return nil, errors.GetAPIError(errors.ErrBucketAlreadyOwnedByYou)
	}

	return nil, errors.GetAPIError(errors.ErrBucketAlreadyExists)
}

func (n *layer) ResolveBucket(ctx context.Context, name string) (cid.ID, error) {
	var cnrID cid.ID
	if err := cnrID.DecodeString(name); err != nil {
		if cnrID, err = n.resolver.Resolve(ctx, name); err != nil {
			return cid.ID{}, err
		}

		reqInfo := api.GetReqInfo(ctx)
		n.log.Info("resolve bucket", zap.String("reqId", reqInfo.RequestID), zap.String("bucket", name), zap.Stringer("cid", cnrID))
	}

	return cnrID, nil
}

func (n *layer) DeleteBucket(ctx context.Context, p *DeleteBucketParams) error {
	nodeVersions, err := n.bucketNodeVersions(ctx, p.BktInfo, "")
	if err != nil {
		return err
	}
	if len(nodeVersions) != 0 {
		return errors.GetAPIError(errors.ErrBucketNotEmpty)
	}

	n.cache.DeleteBucket(p.BktInfo.Name)
	return n.frostFS.DeleteContainer(ctx, p.BktInfo.CID, p.SessionToken)
}
