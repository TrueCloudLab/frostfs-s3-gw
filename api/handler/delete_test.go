package handler

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/stretchr/testify/require"
)

const (
	emptyVersion = ""
)

func TestDeleteBucket(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	_, objInfo := createVersionedBucketAndObject(t, tc, bktName, objName)

	deleteMarkerVersion, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
	require.True(t, isDeleteMarker)

	deleteBucket(t, tc, bktName, http.StatusConflict)
	deleteObject(t, tc, bktName, objName, objInfo.VersionID())
	deleteBucket(t, tc, bktName, http.StatusConflict)
	deleteObject(t, tc, bktName, objName, deleteMarkerVersion)
	deleteBucket(t, tc, bktName, http.StatusNoContent)
}

func TestDeleteObject(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createBucketAndObject(tc, bktName, objName)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	require.False(t, existInMockedFrostFS(tc, bktInfo, objInfo))
}

func TestDeleteObjectFromSuspended(t *testing.T) {
	tc := prepareHandlerContext(t)
	bktName, objName := "bucket-versioned-for-removal", "object-to-delete"

	createSuspendedBucket(t, tc, bktName)
	putObject(t, tc, bktName, objName)

	versionID, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
	require.True(t, isDeleteMarker)
	require.Equal(t, data.UnversionedObjectVersionID, versionID)
}

func TestDeleteDeletedObject(t *testing.T) {
	tc := prepareHandlerContext(t)

	t.Run("unversioned bucket", func(t *testing.T) {
		bktName, objName := "bucket-unversioned-removal", "object-to-delete"
		createBucketAndObject(tc, bktName, objName)

		versionID, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
		require.Empty(t, versionID)
		require.False(t, isDeleteMarker)
		versionID, isDeleteMarker = deleteObject(t, tc, bktName, objName, emptyVersion)
		require.Empty(t, versionID)
		require.False(t, isDeleteMarker)
	})

	t.Run("versioned bucket", func(t *testing.T) {
		bktName, objName := "bucket-versioned-for-removal", "object-to-delete"
		createVersionedBucketAndObject(t, tc, bktName, objName)

		_, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
		require.True(t, isDeleteMarker)
		_, isDeleteMarker = deleteObject(t, tc, bktName, objName, emptyVersion)
		require.True(t, isDeleteMarker)
	})

	t.Run("versioned bucket not found obj", func(t *testing.T) {
		bktName, objName := "bucket-versioned-for-removal", "object-to-delete"
		_, objInfo := createVersionedBucketAndObject(t, tc, bktName, objName)

		versionID, isDeleteMarker := deleteObject(t, tc, bktName, objName, objInfo.VersionID())
		require.False(t, isDeleteMarker)
		require.Equal(t, objInfo.VersionID(), versionID)

		versionID2, isDeleteMarker := deleteObject(t, tc, bktName, objName, versionID)
		require.False(t, isDeleteMarker)
		require.Equal(t, objInfo.VersionID(), versionID2)
	})
}

func TestDeleteObjectVersioned(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createVersionedBucketAndObject(t, tc, bktName, objName)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	checkFound(t, tc, bktName, objName, objInfo.VersionID())
	deleteObject(t, tc, bktName, objName, objInfo.VersionID())
	checkNotFound(t, tc, bktName, objName, objInfo.VersionID())

	require.False(t, existInMockedFrostFS(tc, bktInfo, objInfo), "object exists but shouldn't")
}

func TestDeleteObjectUnversioned(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal-unversioned", "object-to-delete-unversioned"
	bktInfo, objInfo := createBucketAndObject(tc, bktName, objName)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	versions := listVersions(t, tc, bktName)
	require.Len(t, versions.DeleteMarker, 0, "delete markers must be empty")
	require.Len(t, versions.Version, 0, "versions must be empty")

	require.False(t, existInMockedFrostFS(tc, bktInfo, objInfo), "object exists but shouldn't")
}

func TestRemoveDeleteMarker(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createVersionedBucketAndObject(t, tc, bktName, objName)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteMarkerVersion, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
	require.True(t, isDeleteMarker)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	checkFound(t, tc, bktName, objName, objInfo.VersionID())
	deleteObject(t, tc, bktName, objName, deleteMarkerVersion)
	checkFound(t, tc, bktName, objName, emptyVersion)

	require.True(t, existInMockedFrostFS(tc, bktInfo, objInfo), "object doesn't exist but should")
}

func TestDeleteObjectCombined(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createBucketAndObject(tc, bktName, objName)

	putBucketVersioning(t, tc, bktName, true)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	checkFound(t, tc, bktName, objName, objInfo.VersionID())

	require.True(t, existInMockedFrostFS(tc, bktInfo, objInfo), "object doesn't exist but should")
}

func TestDeleteObjectSuspended(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createBucketAndObject(tc, bktName, objName)

	putBucketVersioning(t, tc, bktName, true)

	checkFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, emptyVersion)

	putBucketVersioning(t, tc, bktName, false)

	deleteObject(t, tc, bktName, objName, emptyVersion)
	checkNotFound(t, tc, bktName, objName, objInfo.VersionID())

	require.False(t, existInMockedFrostFS(tc, bktInfo, objInfo), "object exists but shouldn't")
}

func TestDeleteMarkers(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	createTestBucket(tc, bktName)
	putBucketVersioning(t, tc, bktName, true)

	checkNotFound(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)
	deleteObject(t, tc, bktName, objName, emptyVersion)

	versions := listVersions(t, tc, bktName)
	require.Len(t, versions.DeleteMarker, 3, "invalid delete markers length")
	require.Len(t, versions.Version, 0, "versions must be empty")

	require.Len(t, listOIDsFromMockedFrostFS(t, tc, bktName), 0, "shouldn't be any object in frostfs")
}

func TestDeleteObjectFromListCache(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	bktInfo, objInfo := createVersionedBucketAndObject(t, tc, bktName, objName)

	versions := listObjectsV1(t, tc, bktName, "", "", "", -1)
	require.Len(t, versions.Contents, 1)

	checkFound(t, tc, bktName, objName, objInfo.VersionID())
	deleteObject(t, tc, bktName, objName, objInfo.VersionID())
	checkNotFound(t, tc, bktName, objName, objInfo.VersionID())

	// check cache is clean after object removal
	versions = listObjectsV1(t, tc, bktName, "", "", "", -1)
	require.Len(t, versions.Contents, 0)

	require.False(t, existInMockedFrostFS(tc, bktInfo, objInfo))
}

func TestDeleteObjectCheckMarkerReturn(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName, objName := "bucket-for-removal", "object-to-delete"
	createVersionedBucketAndObject(t, tc, bktName, objName)

	deleteMarkerVersion, isDeleteMarker := deleteObject(t, tc, bktName, objName, emptyVersion)
	require.True(t, isDeleteMarker)

	versions := listVersions(t, tc, bktName)
	require.Len(t, versions.DeleteMarker, 1)
	require.Equal(t, deleteMarkerVersion, versions.DeleteMarker[0].VersionID)

	deleteMarkerVersion2, isDeleteMarker2 := deleteObject(t, tc, bktName, objName, deleteMarkerVersion)
	require.True(t, isDeleteMarker2)
	versions = listVersions(t, tc, bktName)
	require.Len(t, versions.DeleteMarker, 0)
	require.Equal(t, deleteMarkerVersion, deleteMarkerVersion2)
}

func createBucketAndObject(tc *handlerContext, bktName, objName string) (*data.BucketInfo, *data.ObjectInfo) {
	bktInfo := createTestBucket(tc, bktName)

	objInfo := createTestObject(tc, bktInfo, objName)

	return bktInfo, objInfo
}

func createVersionedBucketAndObject(t *testing.T, tc *handlerContext, bktName, objName string) (*data.BucketInfo, *data.ObjectInfo) {
	createTestBucket(tc, bktName)
	bktInfo, err := tc.Layer().GetBucketInfo(tc.Context(), bktName)
	require.NoError(t, err)
	putBucketVersioning(t, tc, bktName, true)

	objInfo := createTestObject(tc, bktInfo, objName)

	return bktInfo, objInfo
}

func putBucketVersioning(t *testing.T, tc *handlerContext, bktName string, enabled bool) {
	cfg := &VersioningConfiguration{Status: "Suspended"}
	if enabled {
		cfg.Status = "Enabled"
	}
	w, r := prepareTestRequest(tc, bktName, "", cfg)
	tc.Handler().PutBucketVersioningHandler(w, r)
	assertStatus(t, w, http.StatusOK)
}

func deleteObject(t *testing.T, tc *handlerContext, bktName, objName, version string) (string, bool) {
	query := make(url.Values)
	query.Add(api.QueryVersionID, version)

	w, r := prepareTestFullRequest(tc, bktName, objName, query, nil)
	tc.Handler().DeleteObjectHandler(w, r)
	assertStatus(t, w, http.StatusNoContent)

	return w.Header().Get(api.AmzVersionID), w.Header().Get(api.AmzDeleteMarker) != ""
}

func deleteBucket(t *testing.T, tc *handlerContext, bktName string, code int) {
	w, r := prepareTestRequest(tc, bktName, "", nil)
	tc.Handler().DeleteBucketHandler(w, r)
	assertStatus(t, w, code)
}

func checkNotFound(t *testing.T, tc *handlerContext, bktName, objName, version string) {
	query := make(url.Values)
	query.Add(api.QueryVersionID, version)

	w, r := prepareTestFullRequest(tc, bktName, objName, query, nil)
	tc.Handler().HeadObjectHandler(w, r)
	assertStatus(t, w, http.StatusNotFound)
}

func checkFound(t *testing.T, tc *handlerContext, bktName, objName, version string) {
	query := make(url.Values)
	query.Add(api.QueryVersionID, version)

	w, r := prepareTestFullRequest(tc, bktName, objName, query, nil)
	tc.Handler().HeadObjectHandler(w, r)
	assertStatus(t, w, http.StatusOK)
}

func listVersions(t *testing.T, tc *handlerContext, bktName string) *ListObjectsVersionsResponse {
	w, r := prepareTestRequest(tc, bktName, "", nil)
	tc.Handler().ListBucketObjectVersionsHandler(w, r)
	assertStatus(t, w, http.StatusOK)
	res := &ListObjectsVersionsResponse{}
	parseTestResponse(t, w, res)
	return res
}

func putObject(t *testing.T, tc *handlerContext, bktName, objName string) {
	body := bytes.NewReader([]byte("content"))
	w, r := prepareTestPayloadRequest(tc, bktName, objName, body)
	tc.Handler().PutObjectHandler(w, r)
	assertStatus(t, w, http.StatusOK)
}

func createSuspendedBucket(t *testing.T, tc *handlerContext, bktName string) *data.BucketInfo {
	createTestBucket(tc, bktName)
	bktInfo, err := tc.Layer().GetBucketInfo(tc.Context(), bktName)
	require.NoError(t, err)
	putBucketVersioning(t, tc, bktName, false)
	return bktInfo
}
