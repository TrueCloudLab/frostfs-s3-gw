package layer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
)

type TreeServiceMock struct {
	settings   map[string]*data.BucketSettings
	versions   map[string]map[string][]*data.NodeVersion
	system     map[string]map[string]*data.BaseNodeVersion
	locks      map[string]map[uint64]*data.LockInfo
	tags       map[string]map[uint64]map[string]string
	multiparts map[string]map[string][]*data.MultipartInfo
	parts      map[string]map[int]*data.PartInfo
}

func (t *TreeServiceMock) GetObjectTaggingAndLock(ctx context.Context, bktInfo *data.BucketInfo, objVersion *data.NodeVersion) (map[string]string, *data.LockInfo, error) {
	// TODO implement object tagging
	lock, err := t.GetLock(ctx, bktInfo, objVersion.ID)
	return nil, lock, err
}

func (t *TreeServiceMock) GetObjectTagging(_ context.Context, bktInfo *data.BucketInfo, nodeVersion *data.NodeVersion) (map[string]string, error) {
	cnrTagsMap, ok := t.tags[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, nil
	}

	return cnrTagsMap[nodeVersion.ID], nil
}

func (t *TreeServiceMock) PutObjectTagging(_ context.Context, bktInfo *data.BucketInfo, nodeVersion *data.NodeVersion, tagSet map[string]string) error {
	cnrTagsMap, ok := t.tags[bktInfo.CID.EncodeToString()]
	if !ok {
		t.tags[bktInfo.CID.EncodeToString()] = map[uint64]map[string]string{
			nodeVersion.ID: tagSet,
		}
		return nil
	}

	cnrTagsMap[nodeVersion.ID] = tagSet

	return nil
}

func (t *TreeServiceMock) DeleteObjectTagging(_ context.Context, bktInfo *data.BucketInfo, objVersion *data.NodeVersion) error {
	cnrTagsMap, ok := t.tags[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil
	}

	delete(cnrTagsMap, objVersion.ID)
	return nil
}

func (t *TreeServiceMock) GetBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TreeServiceMock) PutBucketTagging(ctx context.Context, bktInfo *data.BucketInfo, tagSet map[string]string) error {
	// TODO implement me
	panic("implement me")
}

func (t *TreeServiceMock) DeleteBucketTagging(ctx context.Context, bktInfo *data.BucketInfo) error {
	// TODO implement me
	panic("implement me")
}

func NewTreeService() *TreeServiceMock {
	return &TreeServiceMock{
		settings:   make(map[string]*data.BucketSettings),
		versions:   make(map[string]map[string][]*data.NodeVersion),
		system:     make(map[string]map[string]*data.BaseNodeVersion),
		locks:      make(map[string]map[uint64]*data.LockInfo),
		tags:       make(map[string]map[uint64]map[string]string),
		multiparts: make(map[string]map[string][]*data.MultipartInfo),
		parts:      make(map[string]map[int]*data.PartInfo),
	}
}

func (t *TreeServiceMock) PutSettingsNode(_ context.Context, bktInfo *data.BucketInfo, settings *data.BucketSettings) error {
	t.settings[bktInfo.CID.EncodeToString()] = settings
	return nil
}

func (t *TreeServiceMock) GetSettingsNode(_ context.Context, bktInfo *data.BucketInfo) (*data.BucketSettings, error) {
	settings, ok := t.settings[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	return settings, nil
}

func (t *TreeServiceMock) GetNotificationConfigurationNode(ctx context.Context, bktInfo *data.BucketInfo) (oid.ID, error) {
	panic("implement me")
}

func (t *TreeServiceMock) PutNotificationConfigurationNode(ctx context.Context, bktInfo *data.BucketInfo, objID oid.ID) (oid.ID, error) {
	panic("implement me")
}

func (t *TreeServiceMock) GetBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) (oid.ID, error) {
	systemMap, ok := t.system[bktInfo.CID.EncodeToString()]
	if !ok {
		return oid.ID{}, nil
	}

	node, ok := systemMap["cors"]
	if !ok {
		return oid.ID{}, nil
	}

	return node.OID, nil
}

func (t *TreeServiceMock) PutBucketCORS(ctx context.Context, bktInfo *data.BucketInfo, objID oid.ID) (oid.ID, error) {
	systemMap, ok := t.system[bktInfo.CID.EncodeToString()]
	if !ok {
		systemMap = make(map[string]*data.BaseNodeVersion)
	}

	systemMap["cors"] = &data.BaseNodeVersion{
		OID: objID,
	}

	t.system[bktInfo.CID.EncodeToString()] = systemMap

	return oid.ID{}, ErrNoNodeToRemove
}

func (t *TreeServiceMock) DeleteBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) (oid.ID, error) {
	panic("implement me")
}

func (t *TreeServiceMock) GetVersions(_ context.Context, bktInfo *data.BucketInfo, objectName string) ([]*data.NodeVersion, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	versions, ok := cnrVersionsMap[objectName]
	if !ok {
		return nil, ErrNodeNotFound
	}

	return versions, nil
}

func (t *TreeServiceMock) GetLatestVersion(_ context.Context, bktInfo *data.BucketInfo, objectName string) (*data.NodeVersion, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	versions, ok := cnrVersionsMap[objectName]
	if !ok {
		return nil, ErrNodeNotFound
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ID < versions[j].ID
	})

	if len(versions) != 0 {
		return versions[len(versions)-1], nil
	}

	return nil, ErrNodeNotFound
}

func (t *TreeServiceMock) GetLatestVersionsByPrefix(_ context.Context, bktInfo *data.BucketInfo, prefix string) ([]*data.NodeVersion, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	var result []*data.NodeVersion

	for key, versions := range cnrVersionsMap {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		sort.Slice(versions, func(i, j int) bool {
			return versions[i].ID < versions[j].ID
		})

		if len(versions) != 0 {
			result = append(result, versions[len(versions)-1])
		}
	}

	return result, nil
}

func (t *TreeServiceMock) GetUnversioned(_ context.Context, bktInfo *data.BucketInfo, objectName string) (*data.NodeVersion, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	versions, ok := cnrVersionsMap[objectName]
	if !ok {
		return nil, ErrNodeNotFound
	}

	for _, version := range versions {
		if version.IsUnversioned {
			return version, nil
		}
	}

	return nil, ErrNodeNotFound
}

func (t *TreeServiceMock) AddVersion(_ context.Context, bktInfo *data.BucketInfo, newVersion *data.NodeVersion) (uint64, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		t.versions[bktInfo.CID.EncodeToString()] = map[string][]*data.NodeVersion{
			newVersion.FilePath: {newVersion},
		}
		return newVersion.ID, nil
	}

	versions, ok := cnrVersionsMap[newVersion.FilePath]
	if !ok {
		cnrVersionsMap[newVersion.FilePath] = []*data.NodeVersion{newVersion}
		return newVersion.ID, nil
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ID < versions[j].ID
	})

	if len(versions) != 0 {
		newVersion.ID = versions[len(versions)-1].ID + 1
		newVersion.Timestamp = versions[len(versions)-1].Timestamp + 1
	}

	result := versions

	if newVersion.IsUnversioned {
		result = make([]*data.NodeVersion, 0, len(versions))
		for _, node := range versions {
			if !node.IsUnversioned {
				result = append(result, node)
			}
		}
	}

	cnrVersionsMap[newVersion.FilePath] = append(result, newVersion)

	return newVersion.ID, nil
}

func (t *TreeServiceMock) RemoveVersion(_ context.Context, bktInfo *data.BucketInfo, nodeID uint64) error {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return ErrNodeNotFound
	}

	for key, versions := range cnrVersionsMap {
		for i, node := range versions {
			if node.ID == nodeID {
				cnrVersionsMap[key] = append(versions[:i], versions[i+1:]...)
				return nil
			}
		}
	}

	return ErrNodeNotFound
}

func (t *TreeServiceMock) GetAllVersionsByPrefix(_ context.Context, bktInfo *data.BucketInfo, prefix string) ([]*data.NodeVersion, error) {
	cnrVersionsMap, ok := t.versions[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, nil
	}

	var result []*data.NodeVersion
	for objName, versions := range cnrVersionsMap {
		if strings.HasPrefix(objName, prefix) {
			result = append(result, versions...)
		}
	}

	return result, nil
}

func (t *TreeServiceMock) CreateMultipartUpload(_ context.Context, bktInfo *data.BucketInfo, info *data.MultipartInfo) error {
	cnrMultipartsMap, ok := t.multiparts[bktInfo.CID.EncodeToString()]
	if !ok {
		t.multiparts[bktInfo.CID.EncodeToString()] = map[string][]*data.MultipartInfo{
			info.Key: {info},
		}
		return nil
	}

	multiparts := cnrMultipartsMap[info.Key]
	if len(multiparts) != 0 {
		info.ID = multiparts[len(multiparts)-1].ID + 1
	}
	cnrMultipartsMap[info.Key] = append(multiparts, info)

	return nil
}

func (t *TreeServiceMock) GetMultipartUploadsByPrefix(ctx context.Context, bktInfo *data.BucketInfo, prefix string) ([]*data.MultipartInfo, error) {
	panic("implement me")
}

func (t *TreeServiceMock) GetMultipartUpload(_ context.Context, bktInfo *data.BucketInfo, objectName, uploadID string) (*data.MultipartInfo, error) {
	cnrMultipartsMap, ok := t.multiparts[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, ErrNodeNotFound
	}

	multiparts := cnrMultipartsMap[objectName]
	for _, multipart := range multiparts {
		if multipart.UploadID == uploadID {
			return multipart, nil
		}
	}

	return nil, ErrNodeNotFound
}

func (t *TreeServiceMock) AddPart(ctx context.Context, bktInfo *data.BucketInfo, multipartNodeID uint64, info *data.PartInfo) (oldObjIDToDelete oid.ID, err error) {
	multipartInfo, err := t.GetMultipartUpload(ctx, bktInfo, info.Key, info.UploadID)
	if err != nil {
		return oid.ID{}, err
	}

	if multipartInfo.ID != multipartNodeID {
		return oid.ID{}, fmt.Errorf("invalid multipart info id")
	}

	partsMap, ok := t.parts[info.UploadID]
	if !ok {
		partsMap = make(map[int]*data.PartInfo)
	}

	partsMap[info.Number] = info

	t.parts[info.UploadID] = partsMap
	return oid.ID{}, nil
}

func (t *TreeServiceMock) GetParts(_ context.Context, bktInfo *data.BucketInfo, multipartNodeID uint64) ([]*data.PartInfo, error) {
	cnrMultipartsMap := t.multiparts[bktInfo.CID.EncodeToString()]

	var foundMultipart *data.MultipartInfo

LOOP:
	for _, multiparts := range cnrMultipartsMap {
		for _, multipart := range multiparts {
			if multipart.ID == multipartNodeID {
				foundMultipart = multipart
				break LOOP
			}
		}
	}

	if foundMultipart == nil {
		return nil, ErrNodeNotFound
	}

	partsMap := t.parts[foundMultipart.UploadID]
	result := make([]*data.PartInfo, 0, len(partsMap))
	for _, part := range partsMap {
		result = append(result, part)
	}

	return result, nil
}

func (t *TreeServiceMock) DeleteMultipartUpload(_ context.Context, bktInfo *data.BucketInfo, multipartNodeID uint64) error {
	cnrMultipartsMap := t.multiparts[bktInfo.CID.EncodeToString()]

	var uploadID string

LOOP:
	for key, multiparts := range cnrMultipartsMap {
		for i, multipart := range multiparts {
			if multipart.ID == multipartNodeID {
				uploadID = multipart.UploadID
				cnrMultipartsMap[key] = append(multiparts[:i], multiparts[i+1:]...)
				break LOOP
			}
		}
	}

	if uploadID == "" {
		return ErrNodeNotFound
	}

	delete(t.parts, uploadID)
	return nil
}

func (t *TreeServiceMock) PutLock(ctx context.Context, bktInfo *data.BucketInfo, nodeID uint64, lock *data.LockInfo) error {
	cnrLockMap, ok := t.locks[bktInfo.CID.EncodeToString()]
	if !ok {
		t.locks[bktInfo.CID.EncodeToString()] = map[uint64]*data.LockInfo{
			nodeID: lock,
		}
		return nil
	}

	cnrLockMap[nodeID] = lock

	return nil
}

func (t *TreeServiceMock) GetLock(ctx context.Context, bktInfo *data.BucketInfo, nodeID uint64) (*data.LockInfo, error) {
	cnrLockMap, ok := t.locks[bktInfo.CID.EncodeToString()]
	if !ok {
		return nil, nil
	}

	return cnrLockMap[nodeID], nil
}
