package tokens

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/cache"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/user"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
)

type (
	// Credentials is a bearer token get/put interface.
	Credentials interface {
		GetBox(context.Context, oid.Address) (*accessbox.Box, error)
		Put(context.Context, cid.ID, user.ID, *accessbox.AccessBox, uint64, ...*keys.PublicKey) (oid.Address, error)
	}

	cred struct {
		key     *keys.PrivateKey
		frostFS FrostFS
		cache   *cache.AccessBoxCache
	}
)

// PrmObjectCreate groups parameters of objects created by credential tool.
type PrmObjectCreate struct {
	// FrostFS identifier of the object creator.
	Creator user.ID

	// FrostFS container to store the object.
	Container cid.ID

	// File path.
	Filepath string

	// Last FrostFS epoch of the object lifetime.
	ExpirationEpoch uint64

	// Object payload.
	Payload []byte
}

// FrostFS represents virtual connection to FrostFS network.
type FrostFS interface {
	// CreateObject creates and saves a parameterized object in the specified
	// FrostFS container from a specific user. It sets 'Timestamp' attribute to the current time.
	// It returns the ID of the saved object.
	//
	// It returns exactly one non-nil value. It returns any error encountered which
	// prevented the object from being created.
	CreateObject(context.Context, PrmObjectCreate) (oid.ID, error)

	// ReadObjectPayload reads payload of the object from FrostFS network by address
	// into memory.
	//
	// It returns exactly one non-nil value. It returns any error encountered which
	// prevented the object payload from being read.
	ReadObjectPayload(context.Context, oid.Address) ([]byte, error)
}

var (
	// ErrEmptyPublicKeys is returned when no HCS keys are provided.
	ErrEmptyPublicKeys = errors.New("HCS public keys could not be empty")
	// ErrEmptyBearerToken is returned when no bearer token is provided.
	ErrEmptyBearerToken = errors.New("Bearer token could not be empty")
)

var _ = New

// New creates a new Credentials instance using the given cli and key.
func New(frostFS FrostFS, key *keys.PrivateKey, config *cache.Config) Credentials {
	return &cred{frostFS: frostFS, key: key, cache: cache.NewAccessBoxCache(config)}
}

func (c *cred) GetBox(ctx context.Context, addr oid.Address) (*accessbox.Box, error) {
	cachedBox := c.cache.Get(addr)
	if cachedBox != nil {
		return cachedBox, nil
	}

	box, err := c.getAccessBox(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("get access box: %w", err)
	}

	cachedBox, err = box.GetBox(c.key)
	if err != nil {
		return nil, fmt.Errorf("get box: %w", err)
	}

	if err = c.cache.Put(addr, cachedBox); err != nil {
		return nil, fmt.Errorf("put box into cache: %w", err)
	}

	return cachedBox, nil
}

func (c *cred) getAccessBox(ctx context.Context, addr oid.Address) (*accessbox.AccessBox, error) {
	data, err := c.frostFS.ReadObjectPayload(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	// decode access box
	var box accessbox.AccessBox
	if err = box.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("unmarhal access box: %w", err)
	}

	return &box, nil
}

func (c *cred) Put(ctx context.Context, idCnr cid.ID, issuer user.ID, box *accessbox.AccessBox, expiration uint64, keys ...*keys.PublicKey) (oid.Address, error) {
	if len(keys) == 0 {
		return oid.Address{}, ErrEmptyPublicKeys
	} else if box == nil {
		return oid.Address{}, ErrEmptyBearerToken
	}
	data, err := box.Marshal()
	if err != nil {
		return oid.Address{}, fmt.Errorf("marshall box: %w", err)
	}

	idObj, err := c.frostFS.CreateObject(ctx, PrmObjectCreate{
		Creator:         issuer,
		Container:       idCnr,
		Filepath:        strconv.FormatInt(time.Now().Unix(), 10) + "_access.box",
		ExpirationEpoch: expiration,
		Payload:         data,
	})
	if err != nil {
		return oid.Address{}, fmt.Errorf("create object: %w", err)
	}

	var addr oid.Address
	addr.SetObject(idObj)
	addr.SetContainer(idCnr)

	return addr, nil
}
