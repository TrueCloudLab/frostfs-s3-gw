package layer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	objectv2 "github.com/TrueCloudLab/frostfs-api-go/v2/object"
	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/TrueCloudLab/frostfs-sdk-go/checksum"
	"github.com/TrueCloudLab/frostfs-sdk-go/container"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
)

type TestFrostFS struct {
	FrostFS

	objects      map[string]*object.Object
	containers   map[string]*container.Container
	eaclTables   map[string]*eacl.Table
	currentEpoch uint64
}

func NewTestFrostFS() *TestFrostFS {
	return &TestFrostFS{
		objects:    make(map[string]*object.Object),
		containers: make(map[string]*container.Container),
		eaclTables: make(map[string]*eacl.Table),
	}
}

func (t *TestFrostFS) CurrentEpoch() uint64 {
	return t.currentEpoch
}

func (t *TestFrostFS) Objects() []*object.Object {
	res := make([]*object.Object, 0, len(t.objects))

	for _, obj := range t.objects {
		res = append(res, obj)
	}

	return res
}

func (t *TestFrostFS) AddObject(key string, obj *object.Object) {
	t.objects[key] = obj
}

func (t *TestFrostFS) ContainerID(name string) (cid.ID, error) {
	for id, cnr := range t.containers {
		if container.Name(*cnr) == name {
			var cnrID cid.ID
			return cnrID, cnrID.DecodeString(id)
		}
	}
	return cid.ID{}, fmt.Errorf("not found")
}

func (t *TestFrostFS) CreateContainer(_ context.Context, prm PrmContainerCreate) (cid.ID, error) {
	var cnr container.Container
	cnr.Init()
	cnr.SetOwner(prm.Creator)
	cnr.SetPlacementPolicy(prm.Policy)
	cnr.SetBasicACL(prm.BasicACL)

	creationTime := prm.CreationTime
	if creationTime.IsZero() {
		creationTime = time.Now()
	}
	container.SetCreationTime(&cnr, creationTime)

	if prm.Name != "" {
		var d container.Domain
		d.SetName(prm.Name)

		container.WriteDomain(&cnr, d)
		container.SetName(&cnr, prm.Name)
	}

	for i := range prm.AdditionalAttributes {
		cnr.SetAttribute(prm.AdditionalAttributes[i][0], prm.AdditionalAttributes[i][1])
	}

	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return cid.ID{}, err
	}

	var id cid.ID
	id.SetSHA256(sha256.Sum256(b))
	t.containers[id.EncodeToString()] = &cnr

	return id, nil
}

func (t *TestFrostFS) DeleteContainer(_ context.Context, cnrID cid.ID, _ *session.Container) error {
	delete(t.containers, cnrID.EncodeToString())

	return nil
}

func (t *TestFrostFS) Container(_ context.Context, id cid.ID) (*container.Container, error) {
	for k, v := range t.containers {
		if k == id.EncodeToString() {
			return v, nil
		}
	}

	return nil, fmt.Errorf("container not found %s", id)
}

func (t *TestFrostFS) UserContainers(_ context.Context, _ user.ID) ([]cid.ID, error) {
	var res []cid.ID
	for k := range t.containers {
		var idCnr cid.ID
		if err := idCnr.DecodeString(k); err != nil {
			return nil, err
		}
		res = append(res, idCnr)
	}

	return res, nil
}

func (t *TestFrostFS) ReadObject(ctx context.Context, prm PrmObjectRead) (*ObjectPart, error) {
	var addr oid.Address
	addr.SetContainer(prm.Container)
	addr.SetObject(prm.Object)

	sAddr := addr.EncodeToString()

	if obj, ok := t.objects[sAddr]; ok {
		owner := getOwner(ctx)
		if !obj.OwnerID().Equals(owner) && !t.isPublicRead(prm.Container) {
			return nil, ErrAccessDenied
		}

		payload := obj.Payload()

		if prm.PayloadRange[0]+prm.PayloadRange[1] > 0 {
			off := prm.PayloadRange[0]
			payload = payload[off : off+prm.PayloadRange[1]]
		}

		return &ObjectPart{
			Head:    obj,
			Payload: io.NopCloser(bytes.NewReader(payload)),
		}, nil
	}

	return nil, fmt.Errorf("object not found %s", addr)
}

func (t *TestFrostFS) CreateObject(ctx context.Context, prm PrmObjectCreate) (oid.ID, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return oid.ID{}, err
	}
	var id oid.ID
	id.SetSHA256(sha256.Sum256(b))

	attrs := make([]object.Attribute, 0)

	if prm.Filepath != "" {
		a := object.NewAttribute()
		a.SetKey(object.AttributeFilePath)
		a.SetValue(prm.Filepath)
		attrs = append(attrs, *a)
	}

	for i := range prm.Attributes {
		a := object.NewAttribute()
		a.SetKey(prm.Attributes[i][0])
		a.SetValue(prm.Attributes[i][1])
		attrs = append(attrs, *a)
	}

	obj := object.New()
	obj.SetContainerID(prm.Container)
	obj.SetID(id)
	obj.SetPayloadSize(prm.PayloadSize)
	obj.SetAttributes(attrs...)
	obj.SetCreationEpoch(t.currentEpoch)
	obj.SetOwnerID(&prm.Creator)
	t.currentEpoch++

	if len(prm.Locks) > 0 {
		lock := new(object.Lock)
		lock.WriteMembers(prm.Locks)
		objectv2.WriteLock(obj.ToV2(), (objectv2.Lock)(*lock))
	}

	if prm.Payload != nil {
		all, err := io.ReadAll(prm.Payload)
		if err != nil {
			return oid.ID{}, err
		}
		obj.SetPayload(all)
		obj.SetPayloadSize(uint64(len(all)))
		var hash checksum.Checksum
		checksum.Calculate(&hash, checksum.SHA256, all)
		obj.SetPayloadChecksum(hash)
	}

	cnrID, _ := obj.ContainerID()
	objID, _ := obj.ID()

	addr := newAddress(cnrID, objID)
	t.objects[addr.EncodeToString()] = obj
	return objID, nil
}

func (t *TestFrostFS) DeleteObject(ctx context.Context, prm PrmObjectDelete) error {
	var addr oid.Address
	addr.SetContainer(prm.Container)
	addr.SetObject(prm.Object)

	if obj, ok := t.objects[addr.EncodeToString()]; ok {
		owner := getOwner(ctx)
		if !obj.OwnerID().Equals(owner) {
			return ErrAccessDenied
		}

		delete(t.objects, addr.EncodeToString())
	}

	return nil
}

func (t *TestFrostFS) TimeToEpoch(_ context.Context, now, futureTime time.Time) (uint64, uint64, error) {
	return t.currentEpoch, t.currentEpoch + uint64(futureTime.Sub(now).Seconds()), nil
}

func (t *TestFrostFS) AllObjects(cnrID cid.ID) []oid.ID {
	result := make([]oid.ID, 0, len(t.objects))

	for _, val := range t.objects {
		objCnrID, _ := val.ContainerID()
		objObjID, _ := val.ID()
		if cnrID.Equals(objCnrID) {
			result = append(result, objObjID)
		}
	}

	return result
}

func (t *TestFrostFS) SetContainerEACL(_ context.Context, table eacl.Table, _ *session.Container) error {
	cnrID, ok := table.CID()
	if !ok {
		return errors.New("invalid cid")
	}

	if _, ok = t.containers[cnrID.EncodeToString()]; !ok {
		return errors.New("not found")
	}

	t.eaclTables[cnrID.EncodeToString()] = &table

	return nil
}

func (t *TestFrostFS) ContainerEACL(_ context.Context, cnrID cid.ID) (*eacl.Table, error) {
	table, ok := t.eaclTables[cnrID.EncodeToString()]
	if !ok {
		return nil, errors.New("not found")
	}

	return table, nil
}

func (t *TestFrostFS) isPublicRead(cnrID cid.ID) bool {
	table, ok := t.eaclTables[cnrID.EncodeToString()]
	if !ok {
		return false
	}

	for _, rec := range table.Records() {
		if rec.Operation() == eacl.OperationGet && len(rec.Filters()) == 0 {
			for _, trgt := range rec.Targets() {
				if trgt.Role() == eacl.RoleOthers {
					return rec.Action() == eacl.ActionAllow
				}
			}
		}
	}

	return false
}

func getOwner(ctx context.Context) user.ID {
	if bd, ok := ctx.Value(api.BoxData).(*accessbox.Box); ok && bd != nil && bd.Gate != nil && bd.Gate.BearerToken != nil {
		return bearer.ResolveIssuer(*bd.Gate.BearerToken)
	}

	return user.ID{}
}
