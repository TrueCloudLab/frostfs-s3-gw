package layer

import (
	"context"
	errorsStd "errors"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
)

type GetObjectTaggingParams struct {
	ObjectVersion *ObjectVersion

	// NodeVersion can be nil. If not nil we save one request to tree service.
	NodeVersion *data.NodeVersion // optional
}

type PutObjectTaggingParams struct {
	ObjectVersion *ObjectVersion
	TagSet        map[string]string

	// NodeVersion can be nil. If not nil we save one request to tree service.
	NodeVersion *data.NodeVersion // optional
}

func (n *layer) GetObjectTagging(ctx context.Context, p *GetObjectTaggingParams) (string, map[string]string, error) {
	var err error
	owner := n.Owner(ctx)

	if len(p.ObjectVersion.VersionID) != 0 && p.ObjectVersion.VersionID != data.UnversionedObjectVersionID {
		if tags := n.cache.GetTagging(owner, objectTaggingCacheKey(p.ObjectVersion)); tags != nil {
			return p.ObjectVersion.VersionID, tags, nil
		}
	}

	nodeVersion := p.NodeVersion
	if nodeVersion == nil {
		nodeVersion, err = n.getNodeVersionFromCacheOrNeofs(ctx, p.ObjectVersion)
		if err != nil {
			return "", nil, err
		}
	}
	p.ObjectVersion.VersionID = nodeVersion.OID.EncodeToString()

	if tags := n.cache.GetTagging(owner, objectTaggingCacheKey(p.ObjectVersion)); tags != nil {
		return p.ObjectVersion.VersionID, tags, nil
	}

	tags, err := n.treeService.GetObjectTagging(ctx, p.ObjectVersion.BktInfo, nodeVersion)
	if err != nil {
		if errorsStd.Is(err, ErrNodeNotFound) {
			return "", nil, errors.GetAPIError(errors.ErrNoSuchKey)
		}
		return "", nil, err
	}

	n.cache.PutTagging(owner, objectTaggingCacheKey(p.ObjectVersion), tags)

	return p.ObjectVersion.VersionID, tags, nil
}

func (n *layer) PutObjectTagging(ctx context.Context, p *PutObjectTaggingParams) (nodeVersion *data.NodeVersion, err error) {
	nodeVersion = p.NodeVersion
	if nodeVersion == nil {
		nodeVersion, err = n.getNodeVersionFromCacheOrNeofs(ctx, p.ObjectVersion)
		if err != nil {
			return nil, err
		}
	}
	p.ObjectVersion.VersionID = nodeVersion.OID.EncodeToString()

	err = n.treeService.PutObjectTagging(ctx, p.ObjectVersion.BktInfo, nodeVersion, p.TagSet)
	if err != nil {
		if errorsStd.Is(err, ErrNodeNotFound) {
			return nil, errors.GetAPIError(errors.ErrNoSuchKey)
		}
		return nil, err
	}

	n.cache.PutTagging(n.Owner(ctx), objectTaggingCacheKey(p.ObjectVersion), p.TagSet)

	return nodeVersion, nil
}

func (n *layer) DeleteObjectTagging(ctx context.Context, p *ObjectVersion) (*data.NodeVersion, error) {
	version, err := n.getNodeVersion(ctx, p)
	if err != nil {
		return nil, err
	}

	err = n.treeService.DeleteObjectTagging(ctx, p.BktInfo, version)
	if err != nil {
		if errorsStd.Is(err, ErrNodeNotFound) {
			return nil, errors.GetAPIError(errors.ErrNoSuchKey)
		}
		return nil, err
	}

	p.VersionID = version.OID.EncodeToString()

	n.cache.DeleteTagging(objectTaggingCacheKey(p))

	return version, nil
}

func (n *layer) GetBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) (map[string]string, error) {
	owner := n.Owner(ctx)

	if tags := n.cache.GetTagging(owner, bucketTaggingCacheKey(bktInfo.CID)); tags != nil {
		return tags, nil
	}

	tags, err := n.treeService.GetBucketTagging(ctx, bktInfo)
	if err != nil && !errorsStd.Is(err, ErrNodeNotFound) {
		return nil, err
	}

	n.cache.PutTagging(owner, bucketTaggingCacheKey(bktInfo.CID), tags)

	return tags, nil
}

func (n *layer) PutBucketTagging(ctx context.Context, bktInfo *data.BucketInfo, tagSet map[string]string) error {
	if err := n.treeService.PutBucketTagging(ctx, bktInfo, tagSet); err != nil {
		return err
	}

	n.cache.PutTagging(n.Owner(ctx), bucketTaggingCacheKey(bktInfo.CID), tagSet)

	return nil
}

func (n *layer) DeleteBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) error {
	n.cache.DeleteTagging(bucketTaggingCacheKey(bktInfo.CID))

	return n.treeService.DeleteBucketTagging(ctx, bktInfo)
}

func objectTaggingCacheKey(p *ObjectVersion) string {
	return ".tagset." + p.BktInfo.CID.EncodeToString() + "." + p.ObjectName + "." + p.VersionID
}

func bucketTaggingCacheKey(cnrID cid.ID) string {
	return ".tagset." + cnrID.EncodeToString()
}

func (n *layer) getNodeVersion(ctx context.Context, objVersion *ObjectVersion) (*data.NodeVersion, error) {
	var err error
	var version *data.NodeVersion

	if objVersion.VersionID == data.UnversionedObjectVersionID {
		version, err = n.treeService.GetUnversioned(ctx, objVersion.BktInfo, objVersion.ObjectName)
	} else if len(objVersion.VersionID) == 0 {
		version, err = n.treeService.GetLatestVersion(ctx, objVersion.BktInfo, objVersion.ObjectName)
	} else {
		versions, err2 := n.treeService.GetVersions(ctx, objVersion.BktInfo, objVersion.ObjectName)
		if err2 != nil {
			return nil, err2
		}
		for _, v := range versions {
			if v.OID.EncodeToString() == objVersion.VersionID {
				version = v
				break
			}
		}
		if version == nil {
			err = errors.GetAPIError(errors.ErrNoSuchVersion)
		}
	}

	if err == nil && version.IsDeleteMarker() && !objVersion.NoErrorOnDeleteMarker || errorsStd.Is(err, ErrNodeNotFound) {
		return nil, errors.GetAPIError(errors.ErrNoSuchKey)
	}

	return version, err
}

func (n *layer) getNodeVersionFromCache(owner user.ID, o *ObjectVersion) *data.NodeVersion {
	if len(o.VersionID) == 0 || o.VersionID == data.UnversionedObjectVersionID {
		return nil
	}

	var objID oid.ID
	if objID.DecodeString(o.VersionID) != nil {
		return nil
	}

	var addr oid.Address
	addr.SetContainer(o.BktInfo.CID)
	addr.SetObject(objID)

	extObjectInfo := n.cache.GetObject(owner, addr)
	if extObjectInfo == nil {
		return nil
	}

	return extObjectInfo.NodeVersion
}
