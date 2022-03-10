package layer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nspcc-dev/neofs-s3-gw/api/data"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	"go.uber.org/zap"
)

const (
	AttributeExpirationEpoch   = "__NEOFS__EXPIRATION_EPOCH"
	AttributeSysTickEpoch      = "__NEOFS__TICK_EPOCH"
	AttributeSysTickTopic      = "__NEOFS__TICK_TOPIC"
	AttributeParentObject      = ".s3-expire-parent-object"
	AttributeParentBucket      = ".s3-expire-parent-bucket"
	AttributeExpireDate        = ".s3-expire-date"
	AttributeExpireRuleID      = ".s3-expire-rule-id"
	AttributeLifecycleConfigID = ".s3-lifecycle-config"
	ExpireTopic                = "expire"
)

func (n *layer) handleExpireTick(ctx context.Context, msg *nats.Msg) error {
	var addr oid.Address
	if err := addr.DecodeString(string(msg.Data)); err != nil {
		return fmt.Errorf("invalid msg, address expected: %w", err)
	}

	n.log.Debug("handling expiration tick", zap.String("address", string(msg.Data)))

	// and make sure having right access

	//todo redo
	bktInfo := &data.BucketInfo{CID: addr.Container()}

	obj, err := n.objectHead(ctx, bktInfo, addr.Object())
	if err != nil {
		return fmt.Errorf("couldn't head expiration object: %w", err)
	}

	header := userHeaders(obj.Attributes())
	objName := header[AttributeParentObject]
	bktName := header[AttributeParentBucket]
	if objName == "" || bktName == "" {
		return fmt.Errorf("couldn't know bucket/object to expire")
	}

	p := &DeleteObjectParams{
		BktInfo: bktInfo,
		Objects: []*VersionedObject{{Name: objName}},
	}

	res := n.DeleteObjects(ctx, p)
	if res[0].Error != nil {
		return fmt.Errorf("couldn't delete expired object: %w", res[0].Error)
	}

	return n.objectDelete(ctx, bktInfo, addr.Object())
}

func (n *layer) ScheduleLifecycle(ctx context.Context, bktInfo *data.BucketInfo, newConf *data.LifecycleConfiguration) error {
	if newConf == nil {
		return nil
	}

	lifecycleID, err := computeLifecycleID(newConf)
	if err != nil {
		return fmt.Errorf("couldn't compute lifecycle id: %w", err)
	}

	// We want to be able to revert partly applied lifecycle if something goes wrong.
	if err = n.updateLifecycle(ctx, bktInfo, &data.LifecycleConfig{
		OldConfigurationID: lifecycleID,
	}); err != nil {
		return err
	}

	if err = n.applyLifecycle(ctx, bktInfo, lifecycleID, newConf); err != nil {
		return err
	}

	return n.updateLifecycle(ctx, bktInfo, &data.LifecycleConfig{
		OldConfigurationID:   lifecycleID,
		CurrentConfiguration: newConf,
	})
}

func (n *layer) updateLifecycle(ctx context.Context, bktInfo *data.BucketInfo, lifecycleConfig *data.LifecycleConfig) error {
	settings, err := n.GetBucketSettings(ctx, bktInfo)
	if err != nil {
		return fmt.Errorf("couldn't get bucket settings: %w", err)
	}

	settings.LifecycleConfig = lifecycleConfig
	sp := &PutSettingsParams{
		BktInfo:  bktInfo,
		Settings: settings,
	}

	if err = n.PutBucketSettings(ctx, sp); err != nil {
		return fmt.Errorf("couldn't put bucket settings: %w", err)
	}
	return nil
}

func (n *layer) applyLifecycle(ctx context.Context, bktInfo *data.BucketInfo, lifecycleID string, conf *data.LifecycleConfiguration) error {
	for _, rule := range conf.Rules {
		if rule.Status == "Disabled" {
			continue
		}

		listParam := allObjectParams{
			Bucket: bktInfo,
			Prefix: rule.RealPrefix(),
		}

		objects, _, err := n.getLatestObjectsVersions(ctx, listParam)
		if err != nil {
			return err
		}

		if err = n.applyLifecycleToObjects(ctx, bktInfo, lifecycleID, rule, objects); err != nil {
			return err
		}
	}

	return nil
}

func (n *layer) applyLifecycleToObjects(ctx context.Context, bktInfo *data.BucketInfo, lifecycleID string, rule data.Rule, objects []*data.ObjectInfo) error {
	var tags []map[string]string
	var err error
	if rule.NeedTags() {
		tags = make([]map[string]string, len(objects))
		p := &ObjectVersion{
			BktInfo: bktInfo,
		}
		for i, obj := range objects {
			p.ObjectName = obj.Name
			p.VersionID = obj.VersionID()
			if _, tags[i], err = n.GetObjectTagging(ctx, p); err != nil {
				return fmt.Errorf("couldn't get object tags: %w", err)
			}
		}
	}

	for i, obj := range objects {
		var objTags map[string]string
		if len(tags) != 0 {
			objTags = tags[i]
		}
		if !rule.MatchObject(obj, objTags) {
			continue
		}

		expObj := &data.ExpirationObject{
			Expiration:        rule.Expiration,
			RuleID:            rule.ID,
			LifecycleConfigID: lifecycleID,
		}

		if _, err = n.putExpirationObject(ctx, bktInfo, obj, expObj); err != nil {
			return fmt.Errorf("couldn't put expiration object: %w", err)
		}
	}

	return nil
}

func (n *layer) putLifecycleObjects(ctx context.Context, bktInfo *data.BucketInfo, obj *data.ObjectInfo, lifecycle *data.LifecycleConfig) error {
	if lifecycle == nil || lifecycle.CurrentConfiguration == nil {
		return nil
	}

	for _, rule := range lifecycle.CurrentConfiguration.Rules {
		if rule.Status == "Disabled" {
			continue
		}

		// at this time lifecycle.OldConfigurationID is the same as lifecycle.CurrentConfiguration id
		if err := n.applyLifecycleToObjects(ctx, bktInfo, lifecycle.OldConfigurationID, rule, []*data.ObjectInfo{obj}); err != nil {
			return err
		}
	}

	return nil
}

func computeLifecycleID(conf *data.LifecycleConfiguration) (string, error) {
	raw, err := xml.Marshal(conf)
	if err != nil {
		return "", fmt.Errorf("couldn't marshall new lifecycle configuration: %w", err)
	}

	sha := sha256.New()
	sha.Write(raw)
	sum := sha.Sum(nil)

	id := hex.EncodeToString(sum)

	if id == "" {
		return "", fmt.Errorf("computed id is empty")
	}

	return id, nil
}
