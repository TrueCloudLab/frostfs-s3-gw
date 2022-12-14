package layer

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer/encryption"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
)

type (
	// ListObjectsInfo contains common fields of data for ListObjectsV1 and ListObjectsV2.
	ListObjectsInfo struct {
		Prefixes    []string
		Objects     []*data.ObjectInfo
		IsTruncated bool
	}

	// ListObjectsInfoV1 holds data which ListObjectsV1 returns.
	ListObjectsInfoV1 struct {
		ListObjectsInfo
		NextMarker string
	}

	// ListObjectsInfoV2 holds data which ListObjectsV2 returns.
	ListObjectsInfoV2 struct {
		ListObjectsInfo
		NextContinuationToken string
	}

	// ListObjectVersionsInfo stores info and list of objects versions.
	ListObjectVersionsInfo struct {
		CommonPrefixes      []string
		IsTruncated         bool
		KeyMarker           string
		NextKeyMarker       string
		NextVersionIDMarker string
		Version             []*data.ExtendedObjectInfo
		DeleteMarker        []*data.ExtendedObjectInfo
		VersionIDMarker     string
	}
)

// PathSeparator is a path components separator string.
const PathSeparator = string(os.PathSeparator)

func userHeaders(attrs []object.Attribute) map[string]string {
	result := make(map[string]string, len(attrs))

	for _, attr := range attrs {
		result[attr.Key()] = attr.Value()
	}

	return result
}

func objectInfoFromMeta(bkt *data.BucketInfo, meta *object.Object) *data.ObjectInfo {
	var (
		mimeType string
		creation time.Time
	)

	headers := userHeaders(meta.Attributes())
	delete(headers, object.AttributeFilePath)
	if contentType, ok := headers[object.AttributeContentType]; ok {
		mimeType = contentType
		delete(headers, object.AttributeContentType)
	}
	if val, ok := headers[object.AttributeTimestamp]; !ok {
		// ignore empty value
	} else if dt, err := strconv.ParseInt(val, 10, 64); err == nil {
		creation = time.Unix(dt, 0)
		delete(headers, object.AttributeTimestamp)
	}

	objID, _ := meta.ID()
	payloadChecksum, _ := meta.PayloadChecksum()
	return &data.ObjectInfo{
		ID:    objID,
		CID:   bkt.CID,
		IsDir: false,

		Bucket:      bkt.Name,
		Name:        filepathFromObject(meta),
		Created:     creation,
		ContentType: mimeType,
		Headers:     headers,
		Owner:       *meta.OwnerID(),
		Size:        int64(meta.PayloadSize()),
		HashSum:     hex.EncodeToString(payloadChecksum.Value()),
	}
}

func FormEncryptionInfo(headers map[string]string) encryption.ObjectEncryption {
	algorithm := headers[AttributeEncryptionAlgorithm]
	return encryption.ObjectEncryption{
		Enabled:   len(algorithm) > 0,
		Algorithm: algorithm,
		HMACKey:   headers[AttributeHMACKey],
		HMACSalt:  headers[AttributeHMACSalt],
	}
}

func addEncryptionHeaders(meta map[string]string, enc encryption.Params) error {
	meta[AttributeEncryptionAlgorithm] = AESEncryptionAlgorithm
	hmacKey, hmacSalt, err := enc.HMAC()
	if err != nil {
		return fmt.Errorf("get hmac: %w", err)
	}
	meta[AttributeHMACKey] = hex.EncodeToString(hmacKey)
	meta[AttributeHMACSalt] = hex.EncodeToString(hmacSalt)

	return nil
}

func filepathFromObject(o *object.Object) string {
	for _, attr := range o.Attributes() {
		if attr.Key() == object.AttributeFilePath {
			return attr.Value()
		}
	}
	objID, _ := o.ID()
	return objID.EncodeToString()
}

// NameFromString splits name into a base file name and a directory path.
func NameFromString(name string) (string, string) {
	ind := strings.LastIndex(name, PathSeparator)
	return name[ind+1:], name[:ind+1]
}

// GetBoxData  extracts accessbox.Box from context.
func GetBoxData(ctx context.Context) (*accessbox.Box, error) {
	var boxData *accessbox.Box
	data, ok := ctx.Value(api.BoxData).(*accessbox.Box)
	if !ok || data == nil {
		return nil, fmt.Errorf("couldn't get box data from context")
	}

	boxData = data
	if boxData.Gate == nil {
		boxData.Gate = &accessbox.GateData{}
	}
	return boxData, nil
}
