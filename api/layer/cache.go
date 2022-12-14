package layer

import (
	"github.com/TrueCloudLab/frostfs-s3-gw/api/cache"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"go.uber.org/zap"
)

type Cache struct {
	logger      *zap.Logger
	listsCache  *cache.ObjectsListCache
	objCache    *cache.ObjectsCache
	namesCache  *cache.ObjectsNameCache
	bucketCache *cache.BucketCache
	systemCache *cache.SystemCache
	accessCache *cache.AccessControlCache
}

// CachesConfig contains params for caches.
type CachesConfig struct {
	Logger        *zap.Logger
	Objects       *cache.Config
	ObjectsList   *cache.Config
	Names         *cache.Config
	Buckets       *cache.Config
	System        *cache.Config
	AccessControl *cache.Config
}

// DefaultCachesConfigs returns filled configs.
func DefaultCachesConfigs(logger *zap.Logger) *CachesConfig {
	return &CachesConfig{
		Logger:        logger,
		Objects:       cache.DefaultObjectsConfig(logger),
		ObjectsList:   cache.DefaultObjectsListConfig(logger),
		Names:         cache.DefaultObjectsNameConfig(logger),
		Buckets:       cache.DefaultBucketConfig(logger),
		System:        cache.DefaultSystemConfig(logger),
		AccessControl: cache.DefaultAccessControlConfig(logger),
	}
}

func NewCache(cfg *CachesConfig) *Cache {
	return &Cache{
		logger:      cfg.Logger,
		listsCache:  cache.NewObjectsListCache(cfg.ObjectsList),
		objCache:    cache.New(cfg.Objects),
		namesCache:  cache.NewObjectsNameCache(cfg.Names),
		bucketCache: cache.NewBucketCache(cfg.Buckets),
		systemCache: cache.NewSystemCache(cfg.System),
		accessCache: cache.NewAccessControlCache(cfg.AccessControl),
	}
}

func (c *Cache) GetBucket(name string) *data.BucketInfo {
	return c.bucketCache.Get(name)
}

func (c *Cache) PutBucket(bktInfo *data.BucketInfo) {
	if err := c.bucketCache.Put(bktInfo); err != nil {
		c.logger.Warn("couldn't put bucket info into cache",
			zap.String("bucket name", bktInfo.Name),
			zap.Stringer("bucket cid", bktInfo.CID),
			zap.Error(err))
	}
}

func (c *Cache) DeleteBucket(name string) {
	c.bucketCache.Delete(name)
}

func (c *Cache) CleanListCacheEntriesContainingObject(objectName string, cnrID cid.ID) {
	c.listsCache.CleanCacheEntriesContainingObject(objectName, cnrID)
}

func (c *Cache) DeleteObjectName(cnrID cid.ID, bktName, objName string) {
	c.namesCache.Delete(bktName + "/" + objName)
	c.listsCache.CleanCacheEntriesContainingObject(objName, cnrID)
}

func (c *Cache) DeleteObject(addr oid.Address) {
	c.objCache.Delete(addr)
}

func (c *Cache) GetObject(owner user.ID, addr oid.Address) *data.ExtendedObjectInfo {
	if !c.accessCache.Get(owner, addr.String()) {
		return nil
	}

	return c.objCache.GetObject(addr)
}

func (c *Cache) GetLastObject(owner user.ID, bktName, objName string) *data.ExtendedObjectInfo {
	addr := c.namesCache.Get(bktName + "/" + objName)
	if addr == nil {
		return nil
	}

	return c.GetObject(owner, *addr)
}

func (c *Cache) PutObject(owner user.ID, extObjInfo *data.ExtendedObjectInfo) {
	if err := c.objCache.PutObject(extObjInfo); err != nil {
		c.logger.Warn("couldn't add object to cache", zap.Error(err),
			zap.String("object_name", extObjInfo.ObjectInfo.Name), zap.String("bucket_name", extObjInfo.ObjectInfo.Bucket),
			zap.String("cid", extObjInfo.ObjectInfo.CID.EncodeToString()), zap.String("oid", extObjInfo.ObjectInfo.ID.EncodeToString()))
	}

	if err := c.accessCache.Put(owner, extObjInfo.ObjectInfo.Address().EncodeToString()); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) PutObjectWithName(owner user.ID, extObjInfo *data.ExtendedObjectInfo) {
	c.PutObject(owner, extObjInfo)

	if err := c.namesCache.Put(extObjInfo.ObjectInfo.NiceName(), extObjInfo.ObjectInfo.Address()); err != nil {
		c.logger.Warn("couldn't put obj address to name cache",
			zap.String("obj nice name", extObjInfo.ObjectInfo.NiceName()),
			zap.Error(err))
	}
}

func (c *Cache) GetList(owner user.ID, key cache.ObjectsListKey) []*data.NodeVersion {
	if !c.accessCache.Get(owner, key.String()) {
		return nil
	}

	return c.listsCache.GetVersions(key)
}

func (c *Cache) PutList(owner user.ID, key cache.ObjectsListKey, list []*data.NodeVersion) {
	if err := c.listsCache.PutVersions(key, list); err != nil {
		c.logger.Warn("couldn't cache list of objects", zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key.String()); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) GetTagging(owner user.ID, key string) map[string]string {
	if !c.accessCache.Get(owner, key) {
		return nil
	}

	return c.systemCache.GetTagging(key)
}

func (c *Cache) PutTagging(owner user.ID, key string, tags map[string]string) {
	if err := c.systemCache.PutTagging(key, tags); err != nil {
		c.logger.Error("couldn't cache tags", zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) DeleteTagging(key string) {
	c.systemCache.Delete(key)
}

func (c *Cache) GetLockInfo(owner user.ID, key string) *data.LockInfo {
	if !c.accessCache.Get(owner, key) {
		return nil
	}

	return c.systemCache.GetLockInfo(key)
}

func (c *Cache) PutLockInfo(owner user.ID, key string, lockInfo *data.LockInfo) {
	if err := c.systemCache.PutLockInfo(key, lockInfo); err != nil {
		c.logger.Error("couldn't cache lock info", zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) GetSettings(owner user.ID, bktInfo *data.BucketInfo) *data.BucketSettings {
	key := bktInfo.Name + bktInfo.SettingsObjectName()

	if !c.accessCache.Get(owner, key) {
		return nil
	}

	return c.systemCache.GetSettings(key)
}

func (c *Cache) PutSettings(owner user.ID, bktInfo *data.BucketInfo, settings *data.BucketSettings) {
	key := bktInfo.Name + bktInfo.SettingsObjectName()
	if err := c.systemCache.PutSettings(key, settings); err != nil {
		c.logger.Warn("couldn't cache bucket settings", zap.String("bucket", bktInfo.Name), zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) GetCORS(owner user.ID, bkt *data.BucketInfo) *data.CORSConfiguration {
	key := bkt.Name + bkt.CORSObjectName()

	if !c.accessCache.Get(owner, key) {
		return nil
	}

	return c.systemCache.GetCORS(key)
}

func (c *Cache) PutCORS(owner user.ID, bkt *data.BucketInfo, cors *data.CORSConfiguration) {
	key := bkt.Name + bkt.CORSObjectName()

	if err := c.systemCache.PutCORS(key, cors); err != nil {
		c.logger.Warn("couldn't cache cors", zap.String("bucket", bkt.Name), zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}

func (c *Cache) DeleteCORS(bktInfo *data.BucketInfo) {
	c.systemCache.Delete(bktInfo.Name + bktInfo.CORSObjectName())
}

func (c *Cache) GetNotificationConfiguration(owner user.ID, bktInfo *data.BucketInfo) *data.NotificationConfiguration {
	key := bktInfo.Name + bktInfo.NotificationConfigurationObjectName()

	if !c.accessCache.Get(owner, key) {
		return nil
	}

	return c.systemCache.GetNotificationConfiguration(key)
}

func (c *Cache) PutNotificationConfiguration(owner user.ID, bktInfo *data.BucketInfo, configuration *data.NotificationConfiguration) {
	key := bktInfo.Name + bktInfo.NotificationConfigurationObjectName()
	if err := c.systemCache.PutNotificationConfiguration(key, configuration); err != nil {
		c.logger.Warn("couldn't cache notification configuration", zap.String("bucket", bktInfo.Name), zap.Error(err))
	}

	if err := c.accessCache.Put(owner, key); err != nil {
		c.logger.Warn("couldn't cache access control operation", zap.Error(err))
	}
}
