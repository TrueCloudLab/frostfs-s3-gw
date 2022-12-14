package layer

import (
	"context"
	"fmt"
	"strconv"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-sdk-go/client"
	"github.com/TrueCloudLab/frostfs-sdk-go/container"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"go.uber.org/zap"
)

type (
	// BucketACL extends BucketInfo by eacl.Table.
	BucketACL struct {
		Info *data.BucketInfo
		EACL *eacl.Table
	}
)

const (
	attributeLocationConstraint = ".s3-location-constraint"
	AttributeLockEnabled        = "LockEnabled"
)

func (n *layer) containerInfo(ctx context.Context, idCnr cid.ID) (*data.BucketInfo, error) {
	var (
		err error
		res *container.Container
		rid = api.GetRequestID(ctx)
		log = n.log.With(zap.Stringer("cid", idCnr), zap.String("request_id", rid))

		info = &data.BucketInfo{
			CID:  idCnr,
			Name: idCnr.EncodeToString(),
		}
	)
	res, err = n.neoFS.Container(ctx, idCnr)
	if err != nil {
		log.Error("could not fetch container", zap.Error(err))

		if client.IsErrContainerNotFound(err) {
			return nil, errors.GetAPIError(errors.ErrNoSuchBucket)
		}
		return nil, fmt.Errorf("get neofs container: %w", err)
	}

	cnr := *res

	info.Owner = cnr.Owner()
	if domain := container.ReadDomain(cnr); domain.Name() != "" {
		info.Name = domain.Name()
	}
	info.Created = container.CreatedAt(cnr)
	info.LocationConstraint = cnr.Attribute(attributeLocationConstraint)

	attrLockEnabled := cnr.Attribute(AttributeLockEnabled)
	if len(attrLockEnabled) > 0 {
		info.ObjectLockEnabled, err = strconv.ParseBool(attrLockEnabled)
		if err != nil {
			log.Error("could not parse container object lock enabled attribute",
				zap.String("lock_enabled", attrLockEnabled),
				zap.Error(err),
			)
		}
	}

	n.cache.PutBucket(info)

	return info, nil
}

func (n *layer) containerList(ctx context.Context) ([]*data.BucketInfo, error) {
	var (
		err error
		own = n.Owner(ctx)
		res []cid.ID
		rid = api.GetRequestID(ctx)
	)
	res, err = n.neoFS.UserContainers(ctx, own)
	if err != nil {
		n.log.Error("could not list user containers",
			zap.String("request_id", rid),
			zap.Error(err))
		return nil, err
	}

	list := make([]*data.BucketInfo, 0, len(res))
	for i := range res {
		info, err := n.containerInfo(ctx, res[i])
		if err != nil {
			n.log.Error("could not fetch container info",
				zap.String("request_id", rid),
				zap.Error(err))
			continue
		}

		list = append(list, info)
	}

	return list, nil
}

func (n *layer) createContainer(ctx context.Context, p *CreateBucketParams) (*data.BucketInfo, error) {
	ownerID := n.Owner(ctx)
	if p.LocationConstraint == "" {
		p.LocationConstraint = api.DefaultLocationConstraint // s3tests_boto3.functional.test_s3:test_bucket_get_location
	}
	bktInfo := &data.BucketInfo{
		Name:               p.Name,
		Owner:              ownerID,
		Created:            TimeNow(ctx),
		LocationConstraint: p.LocationConstraint,
		ObjectLockEnabled:  p.ObjectLockEnabled,
	}

	var attributes [][2]string

	attributes = append(attributes, [2]string{
		attributeLocationConstraint, p.LocationConstraint,
	})

	if p.ObjectLockEnabled {
		attributes = append(attributes, [2]string{
			AttributeLockEnabled, "true",
		})
	}

	idCnr, err := n.neoFS.CreateContainer(ctx, PrmContainerCreate{
		Creator:              bktInfo.Owner,
		Policy:               p.Policy,
		Name:                 p.Name,
		SessionToken:         p.SessionContainerCreation,
		CreationTime:         bktInfo.Created,
		AdditionalAttributes: attributes,
	})
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	bktInfo.CID = idCnr

	if err = n.setContainerEACLTable(ctx, bktInfo.CID, p.EACL, p.SessionEACL); err != nil {
		return nil, fmt.Errorf("set container eacl: %w", err)
	}

	n.cache.PutBucket(bktInfo)

	return bktInfo, nil
}

func (n *layer) setContainerEACLTable(ctx context.Context, idCnr cid.ID, table *eacl.Table, sessionToken *session.Container) error {
	table.SetCID(idCnr)

	return n.neoFS.SetContainerEACL(ctx, *table, sessionToken)
}

func (n *layer) GetContainerEACL(ctx context.Context, idCnr cid.ID) (*eacl.Table, error) {
	return n.neoFS.ContainerEACL(ctx, idCnr)
}
