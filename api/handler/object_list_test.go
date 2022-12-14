package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/stretchr/testify/require"
)

func TestParseContinuationToken(t *testing.T) {
	var err error

	t.Run("empty token", func(t *testing.T) {
		var queryValues = map[string][]string{
			"continuation-token": {""},
		}
		_, err = parseContinuationToken(queryValues)
		require.Error(t, err)
	})

	t.Run("invalid not empty token", func(t *testing.T) {
		var queryValues = map[string][]string{
			"continuation-token": {"asd"},
		}
		_, err = parseContinuationToken(queryValues)
		require.Error(t, err)
	})

	t.Run("valid token", func(t *testing.T) {
		tokenStr := "75BTT5Z9o79XuKdUeGqvQbqDnxu6qWcR5EhxW8BXFf8t"
		var queryValues = map[string][]string{
			"continuation-token": {tokenStr},
		}
		token, err := parseContinuationToken(queryValues)
		require.NoError(t, err)
		require.Equal(t, tokenStr, token)
	})
}

func TestListObjectNullVersions(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName, objName := "bucket-versioning-enabled", "object"
	createTestBucket(hc, bktName)

	putObjectContent(hc, bktName, objName, "content")
	putBucketVersioning(t, hc, bktName, true)
	putObjectContent(hc, bktName, objName, "content2")

	result := listVersions(t, hc, bktName)

	require.Len(t, result.Version, 2)
	require.Equal(t, data.UnversionedObjectVersionID, result.Version[1].VersionID)
}

func TestS3CompatibilityBucketListV2BothContinuationTokenStartAfter(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName := "bucket-for-listing"
	objects := []string{"bar", "baz", "foo", "quxx"}
	bktInfo, _ := createBucketAndObject(tc, bktName, objects[0])

	for _, objName := range objects[1:] {
		createTestObject(tc, bktInfo, objName)
	}

	listV2Response1 := listObjectsV2(t, tc, bktName, "", "", "bar", "", 1)
	nextContinuationToken := listV2Response1.NextContinuationToken
	require.Equal(t, "baz", listV2Response1.Contents[0].Key)

	listV2Response2 := listObjectsV2(t, tc, bktName, "", "", "bar", nextContinuationToken, -1)

	require.Equal(t, nextContinuationToken, listV2Response2.ContinuationToken)
	require.Equal(t, "bar", listV2Response2.StartAfter)
	require.False(t, listV2Response2.IsTruncated)

	require.Equal(t, "foo", listV2Response2.Contents[0].Key)
	require.Equal(t, "quxx", listV2Response2.Contents[1].Key)
}

func TestS3BucketListDelimiterBasic(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName := "bucket-for-listing"
	objects := []string{"foo/bar", "foo/bar/xyzzy", "quux/thud", "asdf"}
	bktInfo, _ := createBucketAndObject(tc, bktName, objects[0])

	for _, objName := range objects[1:] {
		createTestObject(tc, bktInfo, objName)
	}

	listV1Response := listObjectsV1(t, tc, bktName, "", "/", "", -1)
	require.Equal(t, "/", listV1Response.Delimiter)
	require.Equal(t, "asdf", listV1Response.Contents[0].Key)
	require.Len(t, listV1Response.CommonPrefixes, 2)
	require.Equal(t, "foo/", listV1Response.CommonPrefixes[0].Prefix)
	require.Equal(t, "quux/", listV1Response.CommonPrefixes[1].Prefix)
}

func TestS3BucketListV2DelimiterPrefix(t *testing.T) {
	tc := prepareHandlerContext(t)

	bktName := "bucket-for-listingv2"
	objects := []string{"asdf", "boo/bar", "boo/baz/xyzzy", "cquux/thud", "cquux/bla"}
	bktInfo, _ := createBucketAndObject(tc, bktName, objects[0])

	for _, objName := range objects[1:] {
		createTestObject(tc, bktInfo, objName)
	}

	var empty []string
	delim := "/"
	prefix := ""

	continuationToken := validateListV2(t, tc, bktName, prefix, delim, "", 1, true, false, []string{"asdf"}, empty)
	continuationToken = validateListV2(t, tc, bktName, prefix, delim, continuationToken, 1, true, false, empty, []string{"boo/"})
	validateListV2(t, tc, bktName, prefix, delim, continuationToken, 1, false, true, empty, []string{"cquux/"})

	continuationToken = validateListV2(t, tc, bktName, prefix, delim, "", 2, true, false, []string{"asdf"}, []string{"boo/"})
	validateListV2(t, tc, bktName, prefix, delim, continuationToken, 2, false, true, empty, []string{"cquux/"})

	prefix = "boo/"
	continuationToken = validateListV2(t, tc, bktName, prefix, delim, "", 1, true, false, []string{"boo/bar"}, empty)
	validateListV2(t, tc, bktName, prefix, delim, continuationToken, 1, false, true, empty, []string{"boo/baz/"})

	validateListV2(t, tc, bktName, prefix, delim, "", 2, false, true, []string{"boo/bar"}, []string{"boo/baz/"})
}

func listObjectsV2(t *testing.T, tc *handlerContext, bktName, prefix, delimiter, startAfter, continuationToken string, maxKeys int) *ListObjectsV2Response {
	query := prepareCommonListObjectsQuery(prefix, delimiter, maxKeys)
	if len(startAfter) != 0 {
		query.Add("start-after", startAfter)
	}
	if len(continuationToken) != 0 {
		query.Add("continuation-token", continuationToken)
	}

	w, r := prepareTestFullRequest(tc, bktName, "", query, nil)
	tc.Handler().ListObjectsV2Handler(w, r)
	assertStatus(t, w, http.StatusOK)
	res := &ListObjectsV2Response{}
	parseTestResponse(t, w, res)
	return res
}

func validateListV2(t *testing.T, tc *handlerContext, bktName, prefix, delimiter, continuationToken string, maxKeys int,
	isTruncated, last bool, checkObjects, checkPrefixes []string) string {
	response := listObjectsV2(t, tc, bktName, prefix, delimiter, "", continuationToken, maxKeys)

	require.Equal(t, isTruncated, response.IsTruncated)
	require.Equal(t, last, len(response.NextContinuationToken) == 0)

	require.Len(t, response.Contents, len(checkObjects))
	for i := 0; i < len(checkObjects); i++ {
		require.Equal(t, checkObjects[i], response.Contents[i].Key)
	}

	require.Len(t, response.CommonPrefixes, len(checkPrefixes))
	for i := 0; i < len(checkPrefixes); i++ {
		require.Equal(t, checkPrefixes[i], response.CommonPrefixes[i].Prefix)
	}

	return response.NextContinuationToken
}

func prepareCommonListObjectsQuery(prefix, delimiter string, maxKeys int) url.Values {
	query := make(url.Values)

	if len(delimiter) != 0 {
		query.Add("delimiter", delimiter)
	}
	if len(prefix) != 0 {
		query.Add("prefix", prefix)
	}
	if maxKeys != -1 {
		query.Add("max-keys", strconv.Itoa(maxKeys))
	}

	return query
}

func listObjectsV1(t *testing.T, tc *handlerContext, bktName, prefix, delimiter, marker string, maxKeys int) *ListObjectsV1Response {
	query := prepareCommonListObjectsQuery(prefix, delimiter, maxKeys)
	if len(marker) != 0 {
		query.Add("marker", marker)
	}

	w, r := prepareTestFullRequest(tc, bktName, "", query, nil)
	tc.Handler().ListObjectsV1Handler(w, r)
	assertStatus(t, w, http.StatusOK)
	res := &ListObjectsV1Response{}
	parseTestResponse(t, w, res)
	return res
}
