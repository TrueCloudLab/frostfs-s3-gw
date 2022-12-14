package layer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/cache"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	apiErrors "github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-sdk-go/client"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/minio/sio"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type (
	getParams struct {
		// payload range
		off, ln uint64

		oid     oid.ID
		bktInfo *data.BucketInfo
	}

	// ListObjectsParamsCommon contains common parameters for ListObjectsV1 and ListObjectsV2.
	ListObjectsParamsCommon struct {
		BktInfo   *data.BucketInfo
		Delimiter string
		Encode    string
		MaxKeys   int
		Prefix    string
	}

	// ListObjectsParamsV1 contains params for ListObjectsV1.
	ListObjectsParamsV1 struct {
		ListObjectsParamsCommon
		Marker string
	}

	// ListObjectsParamsV2 contains params for ListObjectsV2.
	ListObjectsParamsV2 struct {
		ListObjectsParamsCommon
		ContinuationToken string
		StartAfter        string
		FetchOwner        bool
	}

	allObjectParams struct {
		Bucket            *data.BucketInfo
		Delimiter         string
		Prefix            string
		MaxKeys           int
		Marker            string
		ContinuationToken string
	}
)

const (
	continuationToken = "<continuation-token>"
)

func newAddress(cnr cid.ID, obj oid.ID) oid.Address {
	var addr oid.Address
	addr.SetContainer(cnr)
	addr.SetObject(obj)
	return addr
}

// objectHead returns all object's headers.
func (n *layer) objectHead(ctx context.Context, bktInfo *data.BucketInfo, idObj oid.ID) (*object.Object, error) {
	prm := PrmObjectRead{
		Container:  bktInfo.CID,
		Object:     idObj,
		WithHeader: true,
	}

	n.prepareAuthParameters(ctx, &prm.PrmAuth, bktInfo.Owner)

	res, err := n.neoFS.ReadObject(ctx, prm)
	if err != nil {
		return nil, err
	}

	return res.Head, nil
}

// initializes payload reader of the NeoFS object.
// Zero range corresponds to full payload (panics if only offset is set).
func (n *layer) initObjectPayloadReader(ctx context.Context, p getParams) (io.Reader, error) {
	prm := PrmObjectRead{
		Container:    p.bktInfo.CID,
		Object:       p.oid,
		WithPayload:  true,
		PayloadRange: [2]uint64{p.off, p.ln},
	}

	n.prepareAuthParameters(ctx, &prm.PrmAuth, p.bktInfo.Owner)

	res, err := n.neoFS.ReadObject(ctx, prm)
	if err != nil {
		return nil, err
	}

	return res.Payload, nil
}

// objectGet returns an object with payload in the object.
func (n *layer) objectGet(ctx context.Context, bktInfo *data.BucketInfo, objID oid.ID) (*object.Object, error) {
	prm := PrmObjectRead{
		Container:   bktInfo.CID,
		Object:      objID,
		WithHeader:  true,
		WithPayload: true,
	}

	n.prepareAuthParameters(ctx, &prm.PrmAuth, bktInfo.Owner)

	res, err := n.neoFS.ReadObject(ctx, prm)
	if err != nil {
		return nil, err
	}

	return res.Head, nil
}

// MimeByFilePath detect mime type by file path extension.
func MimeByFilePath(path string) string {
	ext := filepath.Ext(path)
	if len(ext) == 0 {
		return ""
	}
	return mime.TypeByExtension(ext)
}

func encryptionReader(r io.Reader, size uint64, key []byte) (io.Reader, uint64, error) {
	encSize, err := sio.EncryptedSize(size)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compute enc size: %w", err)
	}

	r, err = sio.EncryptReader(r, sio.Config{MinVersion: sio.Version20, MaxVersion: sio.Version20, Key: key, CipherSuites: []byte{sio.AES_256_GCM}})
	if err != nil {
		return nil, 0, fmt.Errorf("couldn't create encrypter: %w", err)
	}

	return r, encSize, nil
}

func ParseCompletedPartHeader(hdr string) (*Part, error) {
	// partInfo[0] -- part number, partInfo[1] -- part size, partInfo[2] -- checksum
	partInfo := strings.Split(hdr, "-")
	if len(partInfo) != 3 {
		return nil, fmt.Errorf("invalid completed part header")
	}
	num, err := strconv.Atoi(partInfo[0])
	if err != nil {
		return nil, fmt.Errorf("invalid completed part number '%s': %w", partInfo[0], err)
	}
	size, err := strconv.Atoi(partInfo[1])
	if err != nil {
		return nil, fmt.Errorf("invalid completed part size '%s': %w", partInfo[1], err)
	}

	return &Part{
		ETag:       partInfo[2],
		PartNumber: num,
		Size:       int64(size),
	}, nil
}

// PutObject stores object into NeoFS, took payload from io.Reader.
func (n *layer) PutObject(ctx context.Context, p *PutObjectParams) (*data.ExtendedObjectInfo, error) {
	owner := n.Owner(ctx)

	bktSettings, err := n.GetBucketSettings(ctx, p.BktInfo)
	if err != nil {
		return nil, fmt.Errorf("couldn't get versioning settings object: %w", err)
	}

	newVersion := &data.NodeVersion{
		BaseNodeVersion: data.BaseNodeVersion{
			FilePath: p.Object,
			Size:     p.Size,
		},
		IsUnversioned: !bktSettings.VersioningEnabled(),
	}

	r := p.Reader
	if p.Encryption.Enabled() {
		p.Header[AttributeDecryptedSize] = strconv.FormatInt(p.Size, 10)
		if err = addEncryptionHeaders(p.Header, p.Encryption); err != nil {
			return nil, fmt.Errorf("add encryption header: %w", err)
		}

		var encSize uint64
		if r, encSize, err = encryptionReader(p.Reader, uint64(p.Size), p.Encryption.Key()); err != nil {
			return nil, fmt.Errorf("create encrypter: %w", err)
		}
		p.Size = int64(encSize)
	}

	if r != nil {
		if len(p.Header[api.ContentType]) == 0 {
			if contentType := MimeByFilePath(p.Object); len(contentType) == 0 {
				d := newDetector(r)
				if contentType, err := d.Detect(); err == nil {
					p.Header[api.ContentType] = contentType
				}
				r = d.MultiReader()
			} else {
				p.Header[api.ContentType] = contentType
			}
		}
	}

	prm := PrmObjectCreate{
		Container:    p.BktInfo.CID,
		Creator:      owner,
		PayloadSize:  uint64(p.Size),
		Filepath:     p.Object,
		Payload:      r,
		CreationTime: TimeNow(ctx),
		CopiesNumber: p.CopiesNumber,
	}

	prm.Attributes = make([][2]string, 0, len(p.Header))

	for k, v := range p.Header {
		prm.Attributes = append(prm.Attributes, [2]string{k, v})
	}

	id, hash, err := n.objectPutAndHash(ctx, prm, p.BktInfo)
	if err != nil {
		return nil, err
	}

	newVersion.OID = id
	newVersion.ETag = hex.EncodeToString(hash)
	if newVersion.ID, err = n.treeService.AddVersion(ctx, p.BktInfo, newVersion); err != nil {
		return nil, fmt.Errorf("couldn't add new verion to tree service: %w", err)
	}

	if p.Lock != nil && (p.Lock.Retention != nil || p.Lock.LegalHold != nil) {
		putLockInfoPrms := &PutLockInfoParams{
			ObjVersion: &ObjectVersion{
				BktInfo:    p.BktInfo,
				ObjectName: p.Object,
				VersionID:  id.EncodeToString(),
			},
			NewLock:      p.Lock,
			CopiesNumber: p.CopiesNumber,
			NodeVersion:  newVersion, // provide new version to make one less tree service call in PutLockInfo
		}

		if err = n.PutLockInfo(ctx, putLockInfoPrms); err != nil {
			return nil, err
		}
	}

	n.cache.CleanListCacheEntriesContainingObject(p.Object, p.BktInfo.CID)

	objInfo := &data.ObjectInfo{
		ID:  id,
		CID: p.BktInfo.CID,

		Owner:       owner,
		Bucket:      p.BktInfo.Name,
		Name:        p.Object,
		Size:        p.Size,
		Created:     prm.CreationTime,
		Headers:     p.Header,
		ContentType: p.Header[api.ContentType],
		HashSum:     newVersion.ETag,
	}

	extendedObjInfo := &data.ExtendedObjectInfo{
		ObjectInfo:  objInfo,
		NodeVersion: newVersion,
	}

	n.cache.PutObjectWithName(owner, extendedObjInfo)

	return extendedObjInfo, nil
}

func (n *layer) headLastVersionIfNotDeleted(ctx context.Context, bkt *data.BucketInfo, objectName string) (*data.ExtendedObjectInfo, error) {
	owner := n.Owner(ctx)
	if extObjInfo := n.cache.GetLastObject(owner, bkt.Name, objectName); extObjInfo != nil {
		return extObjInfo, nil
	}

	node, err := n.treeService.GetLatestVersion(ctx, bkt, objectName)
	if err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			return nil, apiErrors.GetAPIError(apiErrors.ErrNoSuchKey)
		}
		return nil, err
	}

	if node.IsDeleteMarker() {
		return nil, apiErrors.GetAPIError(apiErrors.ErrNoSuchKey)
	}

	meta, err := n.objectHead(ctx, bkt, node.OID)
	if err != nil {
		return nil, err
	}
	objInfo := objectInfoFromMeta(bkt, meta)

	extObjInfo := &data.ExtendedObjectInfo{
		ObjectInfo:  objInfo,
		NodeVersion: node,
	}

	n.cache.PutObjectWithName(owner, extObjInfo)

	return extObjInfo, nil
}

func (n *layer) headVersion(ctx context.Context, bkt *data.BucketInfo, p *HeadObjectParams) (*data.ExtendedObjectInfo, error) {
	var err error
	var foundVersion *data.NodeVersion
	if p.VersionID == data.UnversionedObjectVersionID {
		foundVersion, err = n.treeService.GetUnversioned(ctx, bkt, p.Object)
		if err != nil {
			if errors.Is(err, ErrNodeNotFound) {
				return nil, apiErrors.GetAPIError(apiErrors.ErrNoSuchVersion)
			}
			return nil, err
		}
	} else {
		versions, err := n.treeService.GetVersions(ctx, bkt, p.Object)
		if err != nil {
			return nil, fmt.Errorf("couldn't get versions: %w", err)
		}

		for _, version := range versions {
			if version.OID.EncodeToString() == p.VersionID {
				foundVersion = version
				break
			}
		}
		if foundVersion == nil {
			return nil, apiErrors.GetAPIError(apiErrors.ErrNoSuchVersion)
		}
	}

	owner := n.Owner(ctx)
	if extObjInfo := n.cache.GetObject(owner, newAddress(bkt.CID, foundVersion.OID)); extObjInfo != nil {
		return extObjInfo, nil
	}

	meta, err := n.objectHead(ctx, bkt, foundVersion.OID)
	if err != nil {
		if client.IsErrObjectNotFound(err) {
			return nil, apiErrors.GetAPIError(apiErrors.ErrNoSuchVersion)
		}
		return nil, err
	}
	objInfo := objectInfoFromMeta(bkt, meta)

	extObjInfo := &data.ExtendedObjectInfo{
		ObjectInfo:  objInfo,
		NodeVersion: foundVersion,
	}

	n.cache.PutObject(owner, extObjInfo)

	return extObjInfo, nil
}

// objectDelete puts tombstone object into neofs.
func (n *layer) objectDelete(ctx context.Context, bktInfo *data.BucketInfo, idObj oid.ID) error {
	prm := PrmObjectDelete{
		Container: bktInfo.CID,
		Object:    idObj,
	}

	n.prepareAuthParameters(ctx, &prm.PrmAuth, bktInfo.Owner)

	n.cache.DeleteObject(newAddress(bktInfo.CID, idObj))

	return n.neoFS.DeleteObject(ctx, prm)
}

// objectPutAndHash prepare auth parameters and invoke neofs.CreateObject.
// Returns object ID and payload sha256 hash.
func (n *layer) objectPutAndHash(ctx context.Context, prm PrmObjectCreate, bktInfo *data.BucketInfo) (oid.ID, []byte, error) {
	n.prepareAuthParameters(ctx, &prm.PrmAuth, bktInfo.Owner)
	hash := sha256.New()
	prm.Payload = wrapReader(prm.Payload, 64*1024, func(buf []byte) {
		hash.Write(buf)
	})
	id, err := n.neoFS.CreateObject(ctx, prm)
	if err != nil {
		return oid.ID{}, nil, err
	}
	return id, hash.Sum(nil), nil
}

// ListObjectsV1 returns objects in a bucket for requests of Version 1.
func (n *layer) ListObjectsV1(ctx context.Context, p *ListObjectsParamsV1) (*ListObjectsInfoV1, error) {
	var result ListObjectsInfoV1

	prm := allObjectParams{
		Bucket:    p.BktInfo,
		Delimiter: p.Delimiter,
		Prefix:    p.Prefix,
		MaxKeys:   p.MaxKeys,
		Marker:    p.Marker,
	}

	objects, next, err := n.getLatestObjectsVersions(ctx, prm)
	if err != nil {
		return nil, err
	}

	if next != nil {
		result.IsTruncated = true
		result.NextMarker = objects[len(objects)-1].Name
	}

	result.Prefixes, result.Objects = triageObjects(objects)

	return &result, nil
}

// ListObjectsV2 returns objects in a bucket for requests of Version 2.
func (n *layer) ListObjectsV2(ctx context.Context, p *ListObjectsParamsV2) (*ListObjectsInfoV2, error) {
	var result ListObjectsInfoV2

	prm := allObjectParams{
		Bucket:            p.BktInfo,
		Delimiter:         p.Delimiter,
		Prefix:            p.Prefix,
		MaxKeys:           p.MaxKeys,
		Marker:            p.StartAfter,
		ContinuationToken: p.ContinuationToken,
	}

	objects, next, err := n.getLatestObjectsVersions(ctx, prm)
	if err != nil {
		return nil, err
	}

	if next != nil {
		result.IsTruncated = true
		result.NextContinuationToken = next.ID.EncodeToString()
	}

	result.Prefixes, result.Objects = triageObjects(objects)

	return &result, nil
}

type logWrapper struct {
	log *zap.Logger
}

func (l *logWrapper) Printf(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (n *layer) getLatestObjectsVersions(ctx context.Context, p allObjectParams) (objects []*data.ObjectInfo, next *data.ObjectInfo, err error) {
	if p.MaxKeys == 0 {
		return nil, nil, nil
	}

	owner := n.Owner(ctx)
	cacheKey := cache.CreateObjectsListCacheKey(p.Bucket.CID, p.Prefix, true)
	nodeVersions := n.cache.GetList(owner, cacheKey)

	if nodeVersions == nil {
		nodeVersions, err = n.treeService.GetLatestVersionsByPrefix(ctx, p.Bucket, p.Prefix)
		if err != nil {
			return nil, nil, err
		}
		n.cache.PutList(owner, cacheKey, nodeVersions)
	}

	if len(nodeVersions) == 0 {
		return nil, nil, nil
	}

	sort.Slice(nodeVersions, func(i, j int) bool {
		return nodeVersions[i].FilePath < nodeVersions[j].FilePath
	})

	poolCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	objOutCh, err := n.initWorkerPool(poolCtx, 2, p, nodesGenerator(poolCtx, p, nodeVersions))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init worker pool: %w", err)
	}

	objects = make([]*data.ObjectInfo, 0, p.MaxKeys)

	for obj := range objOutCh {
		objects = append(objects, obj)
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name < objects[j].Name
	})

	if len(objects) > p.MaxKeys {
		next = objects[p.MaxKeys]
		objects = objects[:p.MaxKeys]
	}

	return
}

func nodesGenerator(ctx context.Context, p allObjectParams, nodeVersions []*data.NodeVersion) <-chan *data.NodeVersion {
	nodeCh := make(chan *data.NodeVersion)
	existed := make(map[string]struct{}, len(nodeVersions)) // to squash the same directories

	go func() {
		var generated int
	LOOP:
		for _, node := range nodeVersions {
			if shouldSkip(node, p, existed) {
				continue
			}

			select {
			case <-ctx.Done():
				break LOOP
			case nodeCh <- node:
				generated++
				if generated == p.MaxKeys+1 { // we use maxKeys+1 to be able to know nextMarker/nextContinuationToken
					break LOOP
				}
			}
		}
		close(nodeCh)
	}()

	return nodeCh
}

func (n *layer) initWorkerPool(ctx context.Context, size int, p allObjectParams, input <-chan *data.NodeVersion) (<-chan *data.ObjectInfo, error) {
	pool, err := ants.NewPool(size, ants.WithLogger(&logWrapper{n.log}))
	if err != nil {
		return nil, fmt.Errorf("coudln't init go pool for listing: %w", err)
	}
	objCh := make(chan *data.ObjectInfo)

	go func() {
		var wg sync.WaitGroup

	LOOP:
		for node := range input {
			select {
			case <-ctx.Done():
				break LOOP
			default:
			}

			// We have to make a copy of pointer to data.NodeVersion
			// to get correct value in submitted task function.
			func(node *data.NodeVersion) {
				wg.Add(1)
				err = pool.Submit(func() {
					defer wg.Done()
					oi := n.objectInfoFromObjectsCacheOrNeoFS(ctx, p.Bucket, node, p.Prefix, p.Delimiter)
					if oi == nil {
						// try to get object again
						if oi = n.objectInfoFromObjectsCacheOrNeoFS(ctx, p.Bucket, node, p.Prefix, p.Delimiter); oi == nil {
							// form object info with data that the tree node contains
							oi = getPartialObjectInfo(p.Bucket, node)
						}
					}
					select {
					case <-ctx.Done():
					case objCh <- oi:
					}
				})
				if err != nil {
					wg.Done()
					n.log.Warn("failed to submit task to pool", zap.Error(err))
				}
			}(node)
		}
		wg.Wait()
		close(objCh)
		pool.Release()
	}()

	return objCh, nil
}

// getPartialObjectInfo form data.ObjectInfo using data available in data.NodeVersion.
func getPartialObjectInfo(bktInfo *data.BucketInfo, node *data.NodeVersion) *data.ObjectInfo {
	return &data.ObjectInfo{
		ID:      node.OID,
		CID:     bktInfo.CID,
		Bucket:  bktInfo.Name,
		Name:    node.FilePath,
		Size:    node.Size,
		HashSum: node.ETag,
	}
}

func (n *layer) bucketNodeVersions(ctx context.Context, bkt *data.BucketInfo, prefix string) ([]*data.NodeVersion, error) {
	var err error

	owner := n.Owner(ctx)
	cacheKey := cache.CreateObjectsListCacheKey(bkt.CID, prefix, false)
	nodeVersions := n.cache.GetList(owner, cacheKey)

	if nodeVersions == nil {
		nodeVersions, err = n.treeService.GetAllVersionsByPrefix(ctx, bkt, prefix)
		if err != nil {
			return nil, fmt.Errorf("get all versions from tree service: %w", err)
		}

		n.cache.PutList(owner, cacheKey, nodeVersions)
	}

	return nodeVersions, nil
}

func (n *layer) getAllObjectsVersions(ctx context.Context, bkt *data.BucketInfo, prefix, delimiter string) (map[string][]*data.ExtendedObjectInfo, error) {
	nodeVersions, err := n.bucketNodeVersions(ctx, bkt, prefix)
	if err != nil {
		return nil, err
	}

	versions := make(map[string][]*data.ExtendedObjectInfo, len(nodeVersions))

	for _, nodeVersion := range nodeVersions {
		oi := &data.ObjectInfo{}

		if nodeVersion.IsDeleteMarker() { // delete marker does not match any object in NeoFS
			oi.ID = nodeVersion.OID
			oi.Name = nodeVersion.FilePath
			oi.Owner = nodeVersion.DeleteMarker.Owner
			oi.Created = nodeVersion.DeleteMarker.Created
			oi.IsDeleteMarker = true
		} else {
			if oi = n.objectInfoFromObjectsCacheOrNeoFS(ctx, bkt, nodeVersion, prefix, delimiter); oi == nil {
				continue
			}
		}

		eoi := &data.ExtendedObjectInfo{
			ObjectInfo:  oi,
			NodeVersion: nodeVersion,
		}

		objVersions, ok := versions[oi.Name]
		if !ok {
			objVersions = []*data.ExtendedObjectInfo{eoi}
		} else if !oi.IsDir {
			objVersions = append(objVersions, eoi)
		}
		versions[oi.Name] = objVersions
	}

	return versions, nil
}

func IsSystemHeader(key string) bool {
	_, ok := api.SystemMetadata[key]
	return ok || strings.HasPrefix(key, api.NeoFSSystemMetadataPrefix)
}

func shouldSkip(node *data.NodeVersion, p allObjectParams, existed map[string]struct{}) bool {
	if node.IsDeleteMarker() {
		return true
	}

	filePath := node.FilePath
	if dirName := tryDirectoryName(node, p.Prefix, p.Delimiter); len(dirName) != 0 {
		filePath = dirName
	}
	if _, ok := existed[filePath]; ok {
		return true
	}

	if filePath <= p.Marker {
		return true
	}

	if p.ContinuationToken != "" {
		if _, ok := existed[continuationToken]; !ok {
			if p.ContinuationToken != node.OID.EncodeToString() {
				return true
			}
			existed[continuationToken] = struct{}{}
		}
	}

	existed[filePath] = struct{}{}
	return false
}

func triageObjects(allObjects []*data.ObjectInfo) (prefixes []string, objects []*data.ObjectInfo) {
	for _, ov := range allObjects {
		if ov.IsDir {
			prefixes = append(prefixes, ov.Name)
		} else {
			objects = append(objects, ov)
		}
	}

	return
}

func triageExtendedObjects(allObjects []*data.ExtendedObjectInfo) (prefixes []string, objects []*data.ExtendedObjectInfo) {
	for _, ov := range allObjects {
		if ov.ObjectInfo.IsDir {
			prefixes = append(prefixes, ov.ObjectInfo.Name)
		} else {
			objects = append(objects, ov)
		}
	}

	return
}

func (n *layer) objectInfoFromObjectsCacheOrNeoFS(ctx context.Context, bktInfo *data.BucketInfo, node *data.NodeVersion, prefix, delimiter string) (oi *data.ObjectInfo) {
	if oiDir := tryDirectory(bktInfo, node, prefix, delimiter); oiDir != nil {
		return oiDir
	}

	owner := n.Owner(ctx)
	if extInfo := n.cache.GetObject(owner, newAddress(bktInfo.CID, node.OID)); extInfo != nil {
		return extInfo.ObjectInfo
	}

	meta, err := n.objectHead(ctx, bktInfo, node.OID)
	if err != nil {
		n.log.Warn("could not fetch object meta", zap.Error(err))
		return nil
	}

	oi = objectInfoFromMeta(bktInfo, meta)
	n.cache.PutObject(owner, &data.ExtendedObjectInfo{ObjectInfo: oi, NodeVersion: node})

	return oi
}

func tryDirectory(bktInfo *data.BucketInfo, node *data.NodeVersion, prefix, delimiter string) *data.ObjectInfo {
	dirName := tryDirectoryName(node, prefix, delimiter)
	if len(dirName) == 0 {
		return nil
	}

	return &data.ObjectInfo{
		ID:             node.OID, // to use it as continuation token
		CID:            bktInfo.CID,
		IsDir:          true,
		IsDeleteMarker: node.IsDeleteMarker(),
		Bucket:         bktInfo.Name,
		Name:           dirName,
	}
}

// tryDirectoryName forms directory name by prefix and delimiter.
// If node isn't a directory empty string is returned.
// This function doesn't check if node has a prefix. It must do a caller.
func tryDirectoryName(node *data.NodeVersion, prefix, delimiter string) string {
	if len(delimiter) == 0 {
		return ""
	}

	tail := strings.TrimPrefix(node.FilePath, prefix)
	index := strings.Index(tail, delimiter)
	if index >= 0 {
		return prefix + tail[:index+1]
	}

	return ""
}

func wrapReader(input io.Reader, bufSize int, f func(buf []byte)) io.Reader {
	if input == nil {
		return nil
	}

	r, w := io.Pipe()
	go func() {
		var buf = make([]byte, bufSize)
		for {
			n, err := input.Read(buf)
			if n > 0 {
				f(buf[:n])
				_, _ = w.Write(buf[:n]) // ignore error, input is not ReadCloser
			}
			if err != nil {
				_ = w.CloseWithError(err)
				break
			}
		}
	}()
	return r
}
