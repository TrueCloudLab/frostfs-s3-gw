package layer

import (
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/stretchr/testify/require"
)

func TestObjectLockAttributes(t *testing.T) {
	tc := prepareContext(t)
	err := tc.layer.PutBucketSettings(tc.ctx, &PutSettingsParams{
		BktInfo:  tc.bktInfo,
		Settings: &data.BucketSettings{Versioning: data.VersioningEnabled},
	})
	require.NoError(t, err)

	obj := tc.putObject([]byte("content obj1 v1"))

	p := &PutLockInfoParams{
		ObjVersion: &ObjectVersion{
			BktInfo:    tc.bktInfo,
			ObjectName: obj.Name,
			VersionID:  obj.VersionID(),
		},
		NewLock: &data.ObjectLock{
			Retention: &data.RetentionLock{
				Until: time.Now(),
			},
		},
		CopiesNumber: 0,
	}

	err = tc.layer.PutLockInfo(tc.ctx, p)
	require.NoError(t, err)

	foundLock, err := tc.layer.GetLockInfo(tc.ctx, p.ObjVersion)
	require.NoError(t, err)

	lockObj := tc.getObjectByID(foundLock.Retention())
	require.NotNil(t, lockObj)

	expEpoch := false
	for _, attr := range lockObj.Attributes() {
		if attr.Key() == AttributeExpirationEpoch {
			expEpoch = true
		}
	}

	require.Truef(t, expEpoch, "system header __NEOFS__EXPIRATION_EPOCH presence")
}
