package layer

import (
	"context"
	errorsStd "errors"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
)

func (n *layer) GetObjectTaggingAndLock(ctx context.Context, objVersion *ObjectVersion, nodeVersion *data.NodeVersion) (map[string]string, *data.LockInfo, error) {
	var err error
	owner := n.Owner(ctx)

	tags := n.cache.GetTagging(owner, objectTaggingCacheKey(objVersion))
	lockInfo := n.cache.GetLockInfo(owner, lockObjectKey(objVersion))

	if tags != nil && lockInfo != nil {
		return tags, lockInfo, nil
	}

	if nodeVersion == nil {
		nodeVersion, err = n.getNodeVersion(ctx, objVersion)
		if err != nil {
			return nil, nil, err
		}
	}

	tags, lockInfo, err = n.treeService.GetObjectTaggingAndLock(ctx, objVersion.BktInfo, nodeVersion)
	if err != nil {
		if errorsStd.Is(err, ErrNodeNotFound) {
			return nil, nil, errors.GetAPIError(errors.ErrNoSuchKey)
		}
		return nil, nil, err
	}

	n.cache.PutTagging(owner, objectTaggingCacheKey(objVersion), tags)
	n.cache.PutLockInfo(owner, lockObjectKey(objVersion), lockInfo)

	return tags, lockInfo, nil
}
