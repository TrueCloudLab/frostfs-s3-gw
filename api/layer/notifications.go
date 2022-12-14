package layer

import (
	"bytes"
	"context"
	"encoding/xml"
	errorsStd "errors"
	"fmt"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"go.uber.org/zap"
)

type PutBucketNotificationConfigurationParams struct {
	RequestInfo   *api.ReqInfo
	BktInfo       *data.BucketInfo
	Configuration *data.NotificationConfiguration
	CopiesNumber  uint32
}

func (n *layer) PutBucketNotificationConfiguration(ctx context.Context, p *PutBucketNotificationConfigurationParams) error {
	confXML, err := xml.Marshal(p.Configuration)
	if err != nil {
		return fmt.Errorf("marshal notify configuration: %w", err)
	}

	prm := PrmObjectCreate{
		Container:    p.BktInfo.CID,
		Creator:      p.BktInfo.Owner,
		Payload:      bytes.NewReader(confXML),
		Filepath:     p.BktInfo.NotificationConfigurationObjectName(),
		CreationTime: TimeNow(ctx),
		CopiesNumber: p.CopiesNumber,
	}

	objID, _, err := n.objectPutAndHash(ctx, prm, p.BktInfo)
	if err != nil {
		return err
	}

	objIDToDelete, err := n.treeService.PutNotificationConfigurationNode(ctx, p.BktInfo, objID)
	objIDToDeleteNotFound := errorsStd.Is(err, ErrNoNodeToRemove)
	if err != nil && !objIDToDeleteNotFound {
		return err
	}

	if !objIDToDeleteNotFound {
		if err = n.objectDelete(ctx, p.BktInfo, objIDToDelete); err != nil {
			n.log.Error("couldn't delete notification configuration object", zap.Error(err),
				zap.String("cnrID", p.BktInfo.CID.EncodeToString()),
				zap.String("bucket name", p.BktInfo.Name),
				zap.String("objID", objIDToDelete.EncodeToString()))
		}
	}

	n.cache.PutNotificationConfiguration(n.Owner(ctx), p.BktInfo, p.Configuration)

	return nil
}

func (n *layer) GetBucketNotificationConfiguration(ctx context.Context, bktInfo *data.BucketInfo) (*data.NotificationConfiguration, error) {
	owner := n.Owner(ctx)
	if conf := n.cache.GetNotificationConfiguration(owner, bktInfo); conf != nil {
		return conf, nil
	}

	objID, err := n.treeService.GetNotificationConfigurationNode(ctx, bktInfo)
	objIDNotFound := errorsStd.Is(err, ErrNodeNotFound)
	if err != nil && !objIDNotFound {
		return nil, err
	}

	conf := &data.NotificationConfiguration{}

	if !objIDNotFound {
		obj, err := n.objectGet(ctx, bktInfo, objID)
		if err != nil {
			return nil, err
		}

		if err = xml.Unmarshal(obj.Payload(), &conf); err != nil {
			return nil, fmt.Errorf("unmarshal notify configuration: %w", err)
		}
	}

	n.cache.PutNotificationConfiguration(owner, bktInfo, conf)

	return conf, nil
}
