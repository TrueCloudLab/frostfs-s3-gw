package cache

import (
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	objecttest "github.com/TrueCloudLab/frostfs-sdk-go/object/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func getTestConfig() *Config {
	return &Config{
		Size:     10,
		Lifetime: 5 * time.Second,
		Logger:   zap.NewExample(),
	}
}

func TestCache(t *testing.T) {
	obj := objecttest.Object()
	objID, _ := obj.ID()
	cnrID, _ := obj.ContainerID()

	var addr oid.Address
	addr.SetContainer(cnrID)
	addr.SetObject(objID)

	extObjInfo := &data.ExtendedObjectInfo{
		ObjectInfo: &data.ObjectInfo{
			ID:  addr.Object(),
			CID: addr.Container(),
		},
		NodeVersion: &data.NodeVersion{
			BaseNodeVersion: data.BaseNodeVersion{
				FilePath: "obj",
				Size:     50,
			},
			IsUnversioned: true,
		},
		IsLatest: true,
	}

	t.Run("check get", func(t *testing.T) {
		cache := New(getTestConfig())
		err := cache.PutObject(extObjInfo)
		require.NoError(t, err)

		actual := cache.GetObject(addr)
		require.Equal(t, extObjInfo, actual)
	})

	t.Run("check delete", func(t *testing.T) {
		cache := New(getTestConfig())
		err := cache.PutObject(extObjInfo)
		require.NoError(t, err)

		cache.Delete(addr)
		actual := cache.GetObject(addr)
		require.Nil(t, actual)
	})
}
