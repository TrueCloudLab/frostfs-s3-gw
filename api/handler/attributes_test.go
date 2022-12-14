package handler

import (
	"strings"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/stretchr/testify/require"
)

func TestGetObjectPartsAttributes(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName := "bucket-get-attributes"
	objName, objMultipartName := "object", "object-multipart"
	partSize := 8

	createTestBucket(hc, bktName)

	putObject(t, hc, bktName, objName)
	result := getObjectAttributes(hc, bktName, objName, objectParts)
	require.Nil(t, result.ObjectParts)

	multipartUpload := createMultipartUpload(hc, bktName, objMultipartName, map[string]string{})
	etag, _ := uploadPart(hc, bktName, objMultipartName, multipartUpload.UploadID, 1, partSize)
	completeMultipartUpload(hc, bktName, objMultipartName, multipartUpload.UploadID, []string{etag})

	result = getObjectAttributes(hc, bktName, objMultipartName, objectParts)
	require.NotNil(t, result.ObjectParts)
	require.Len(t, result.ObjectParts.Parts, 1)
	require.Equal(t, etag, result.ObjectParts.Parts[0].ChecksumSHA256)
	require.Equal(t, partSize, result.ObjectParts.Parts[0].Size)
	require.Equal(t, 1, result.ObjectParts.PartsCount)
}

func getObjectAttributes(hc *handlerContext, bktName, objName string, attrs ...string) *GetObjectAttributesResponse {
	w, r := prepareTestRequest(hc, bktName, objName, nil)
	r.Header.Set(api.AmzObjectAttributes, strings.Join(attrs, ","))
	hc.Handler().GetObjectAttributesHandler(w, r)
	result := &GetObjectAttributesResponse{}
	parseTestResponse(hc.t, w, result)

	return result
}
