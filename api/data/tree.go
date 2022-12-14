package data

import (
	"strconv"
	"time"

	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
)

const (
	UnversionedObjectVersionID = "null"
)

// NodeVersion represent node from tree service.
type NodeVersion struct {
	BaseNodeVersion
	DeleteMarker  *DeleteMarkerInfo
	IsUnversioned bool
}

func (v NodeVersion) IsDeleteMarker() bool {
	return v.DeleteMarker != nil
}

// DeleteMarkerInfo is used to save object info if node in the tree service is delete marker.
// We need this information because the "delete marker" object is no longer stored in NeoFS.
type DeleteMarkerInfo struct {
	Created time.Time
	Owner   user.ID
}

// ExtendedObjectInfo contains additional node info to be able to sort versions by timestamp.
type ExtendedObjectInfo struct {
	ObjectInfo  *ObjectInfo
	NodeVersion *NodeVersion
	IsLatest    bool
}

func (e ExtendedObjectInfo) Version() string {
	if e.NodeVersion.IsUnversioned {
		return UnversionedObjectVersionID
	}

	return e.ObjectInfo.ID.EncodeToString()
}

// BaseNodeVersion is minimal node info from tree service.
// Basically used for "system" object.
type BaseNodeVersion struct {
	ID        uint64
	ParenID   uint64
	OID       oid.ID
	Timestamp uint64
	Size      int64
	ETag      string
	FilePath  string
}

type ObjectTaggingInfo struct {
	CnrID     cid.ID
	ObjName   string
	VersionID string
}

// MultipartInfo is multipart upload information.
type MultipartInfo struct {
	// ID is node id in tree service.
	// It's ignored when creating a new multipart upload.
	ID           uint64
	Key          string
	UploadID     string
	Owner        user.ID
	Created      time.Time
	Meta         map[string]string
	CopiesNumber uint32
}

// PartInfo is upload information about part.
type PartInfo struct {
	Key      string
	UploadID string
	Number   int
	OID      oid.ID
	Size     int64
	ETag     string
	Created  time.Time
}

// ToHeaderString form short part representation to use in S3-Completed-Parts header.
func (p *PartInfo) ToHeaderString() string {
	return strconv.Itoa(p.Number) + "-" + strconv.FormatInt(p.Size, 10) + "-" + p.ETag
}

// LockInfo is lock information to create appropriate tree node.
type LockInfo struct {
	id uint64

	legalHoldOID oid.ID
	setLegalHold bool

	retentionOID oid.ID
	setRetention bool
	untilDate    string
	isCompliance bool
}

func NewLockInfo(id uint64) *LockInfo {
	return &LockInfo{id: id}
}

func (l LockInfo) ID() uint64 {
	return l.id
}

func (l *LockInfo) SetLegalHold(objID oid.ID) {
	l.legalHoldOID = objID
	l.setLegalHold = true
}

func (l *LockInfo) ResetLegalHold() {
	l.setLegalHold = false
}

func (l LockInfo) LegalHold() oid.ID {
	return l.legalHoldOID
}

func (l LockInfo) IsLegalHoldSet() bool {
	return l.setLegalHold
}

func (l *LockInfo) SetRetention(objID oid.ID, until string, isCompliance bool) {
	l.retentionOID = objID
	l.setRetention = true
	l.untilDate = until
	l.isCompliance = isCompliance
}

func (l LockInfo) IsRetentionSet() bool {
	return l.setRetention
}

func (l LockInfo) Retention() oid.ID {
	return l.retentionOID
}

func (l LockInfo) UntilDate() string {
	return l.untilDate
}

func (l LockInfo) IsCompliance() bool {
	return l.isCompliance
}
