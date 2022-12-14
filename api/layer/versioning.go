package layer

import (
	"context"
	"sort"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
)

func (n *layer) ListObjectVersions(ctx context.Context, p *ListObjectVersionsParams) (*ListObjectVersionsInfo, error) {
	var (
		allObjects = make([]*data.ExtendedObjectInfo, 0, p.MaxKeys)
		res        = &ListObjectVersionsInfo{}
	)

	versions, err := n.getAllObjectsVersions(ctx, p.BktInfo, p.Prefix, p.Delimiter)
	if err != nil {
		return nil, err
	}

	sortedNames := make([]string, 0, len(versions))
	for k := range versions {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		sortedVersions := versions[name]
		sort.Slice(sortedVersions, func(i, j int) bool {
			return sortedVersions[j].NodeVersion.Timestamp < sortedVersions[i].NodeVersion.Timestamp // sort in reverse order
		})

		for i, version := range sortedVersions {
			version.IsLatest = i == 0
			allObjects = append(allObjects, version)
		}
	}

	for i, obj := range allObjects {
		if obj.ObjectInfo.Name >= p.KeyMarker && obj.ObjectInfo.VersionID() >= p.VersionIDMarker {
			allObjects = allObjects[i:]
			break
		}
	}

	res.CommonPrefixes, allObjects = triageExtendedObjects(allObjects)

	if len(allObjects) > p.MaxKeys {
		res.IsTruncated = true
		res.NextKeyMarker = allObjects[p.MaxKeys].ObjectInfo.Name
		res.NextVersionIDMarker = allObjects[p.MaxKeys].ObjectInfo.VersionID()

		allObjects = allObjects[:p.MaxKeys]
		res.KeyMarker = allObjects[p.MaxKeys-1].ObjectInfo.Name
		res.VersionIDMarker = allObjects[p.MaxKeys-1].ObjectInfo.VersionID()
	}

	res.Version, res.DeleteMarker = triageVersions(allObjects)
	return res, nil
}

func triageVersions(objVersions []*data.ExtendedObjectInfo) ([]*data.ExtendedObjectInfo, []*data.ExtendedObjectInfo) {
	if len(objVersions) == 0 {
		return nil, nil
	}

	var resVersion []*data.ExtendedObjectInfo
	var resDelMarkVersions []*data.ExtendedObjectInfo

	for _, version := range objVersions {
		if version.NodeVersion.IsDeleteMarker() {
			resDelMarkVersions = append(resDelMarkVersions, version)
		} else {
			resVersion = append(resVersion, version)
		}
	}

	return resVersion, resDelMarkVersions
}
