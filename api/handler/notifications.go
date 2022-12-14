package handler

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/google/uuid"
)

type (
	SendNotificationParams struct {
		Event            string
		NotificationInfo *data.NotificationInfo
		BktInfo          *data.BucketInfo
		ReqInfo          *api.ReqInfo
		User             string
		Time             time.Time
	}

	NotificationConfiguration struct {
		XMLName                   xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ NotificationConfiguation"`
		NotificationConfiguration data.NotificationConfiguration
	}
)

const (
	filterRuleSuffixName = "suffix"
	filterRulePrefixName = "prefix"

	EventObjectCreated                                = "s3:ObjectCreated:*"
	EventObjectCreatedPut                             = "s3:ObjectCreated:Put"
	EventObjectCreatedPost                            = "s3:ObjectCreated:Post"
	EventObjectCreatedCopy                            = "s3:ObjectCreated:Copy"
	EventReducedRedundancyLostObject                  = "s3:ReducedRedundancyLostObject"
	EventObjectCreatedCompleteMultipartUpload         = "s3:ObjectCreated:CompleteMultipartUpload"
	EventObjectRemoved                                = "s3:ObjectRemoved:*"
	EventObjectRemovedDelete                          = "s3:ObjectRemoved:Delete"
	EventObjectRemovedDeleteMarkerCreated             = "s3:ObjectRemoved:DeleteMarkerCreated"
	EventObjectRestore                                = "s3:ObjectRestore:*"
	EventObjectRestorePost                            = "s3:ObjectRestore:Post"
	EventObjectRestoreCompleted                       = "s3:ObjectRestore:Completed"
	EventReplication                                  = "s3:Replication:*"
	EventReplicationOperationFailedReplication        = "s3:Replication:OperationFailedReplication"
	EventReplicationOperationNotTracked               = "s3:Replication:OperationNotTracked"
	EventReplicationOperationMissedThreshold          = "s3:Replication:OperationMissedThreshold"
	EventReplicationOperationReplicatedAfterThreshold = "s3:Replication:OperationReplicatedAfterThreshold"
	EventObjectRestoreDelete                          = "s3:ObjectRestore:Delete"
	EventLifecycleTransition                          = "s3:LifecycleTransition"
	EventIntelligentTiering                           = "s3:IntelligentTiering"
	EventObjectACLPut                                 = "s3:ObjectAcl:Put"
	EventLifecycleExpiration                          = "s3:LifecycleExpiration:*"
	EventLifecycleExpirationDelete                    = "s3:LifecycleExpiration:Delete"
	EventLifecycleExpirationDeleteMarkerCreated       = "s3:LifecycleExpiration:DeleteMarkerCreated"
	EventObjectTagging                                = "s3:ObjectTagging:*"
	EventObjectTaggingPut                             = "s3:ObjectTagging:Put"
	EventObjectTaggingDelete                          = "s3:ObjectTagging:Delete"
)

var validEvents = map[string]struct{}{
	EventReducedRedundancyLostObject:                  {},
	EventObjectCreated:                                {},
	EventObjectCreatedPut:                             {},
	EventObjectCreatedPost:                            {},
	EventObjectCreatedCopy:                            {},
	EventObjectCreatedCompleteMultipartUpload:         {},
	EventObjectRemoved:                                {},
	EventObjectRemovedDelete:                          {},
	EventObjectRemovedDeleteMarkerCreated:             {},
	EventObjectRestore:                                {},
	EventObjectRestorePost:                            {},
	EventObjectRestoreCompleted:                       {},
	EventReplication:                                  {},
	EventReplicationOperationFailedReplication:        {},
	EventReplicationOperationNotTracked:               {},
	EventReplicationOperationMissedThreshold:          {},
	EventReplicationOperationReplicatedAfterThreshold: {},
	EventObjectRestoreDelete:                          {},
	EventLifecycleTransition:                          {},
	EventIntelligentTiering:                           {},
	EventObjectACLPut:                                 {},
	EventLifecycleExpiration:                          {},
	EventLifecycleExpirationDelete:                    {},
	EventLifecycleExpirationDeleteMarkerCreated:       {},
	EventObjectTagging:                                {},
	EventObjectTaggingPut:                             {},
	EventObjectTaggingDelete:                          {},
}

func (h *handler) PutBucketNotificationHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	conf := &data.NotificationConfiguration{}
	if err = xml.NewDecoder(r.Body).Decode(conf); err != nil {
		h.logAndSendError(w, "couldn't decode notification configuration", reqInfo, errors.GetAPIError(errors.ErrMalformedXML))
		return
	}

	if _, err = h.checkBucketConfiguration(r.Context(), conf, reqInfo); err != nil {
		h.logAndSendError(w, "couldn't check bucket configuration", reqInfo, err)
		return
	}

	p := &layer.PutBucketNotificationConfigurationParams{
		RequestInfo:   reqInfo,
		BktInfo:       bktInfo,
		Configuration: conf,
		CopiesNumber:  h.cfg.CopiesNumber,
	}

	if err = h.obj.PutBucketNotificationConfiguration(r.Context(), p); err != nil {
		h.logAndSendError(w, "couldn't put bucket configuration", reqInfo, err)
		return
	}
}

func (h *handler) GetBucketNotificationHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	conf, err := h.obj.GetBucketNotificationConfiguration(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get bucket notification configuration", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, conf); err != nil {
		h.logAndSendError(w, "could not encode bucket notification configuration to response", reqInfo, err)
		return
	}
}

func (h *handler) sendNotifications(ctx context.Context, p *SendNotificationParams) error {
	if !h.cfg.NotificatorEnabled {
		return nil
	}

	conf, err := h.obj.GetBucketNotificationConfiguration(ctx, p.BktInfo)
	if err != nil {
		return fmt.Errorf("failed to get notification configuration: %w", err)
	}
	if conf.IsEmpty() {
		return nil
	}

	box, err := layer.GetBoxData(ctx)
	if err == nil && box.Gate.BearerToken != nil {
		p.User = bearer.ResolveIssuer(*box.Gate.BearerToken).EncodeToString()
	}

	p.Time = layer.TimeNow(ctx)

	topics := filterSubjects(conf, p.Event, p.NotificationInfo.Name)

	return h.notificator.SendNotifications(topics, p)
}

// checkBucketConfiguration checks notification configuration and generates an ID for configurations with empty ids.
func (h *handler) checkBucketConfiguration(ctx context.Context, conf *data.NotificationConfiguration, r *api.ReqInfo) (completed bool, err error) {
	if conf == nil {
		return
	}

	if conf.TopicConfigurations != nil || conf.LambdaFunctionConfigurations != nil {
		return completed, errors.GetAPIError(errors.ErrNotificationTopicNotSupported)
	}

	for i, q := range conf.QueueConfigurations {
		if err = checkEvents(q.Events); err != nil {
			return
		}

		if err = checkRules(q.Filter.Key.FilterRules); err != nil {
			return
		}

		if h.cfg.NotificatorEnabled {
			if err = h.notificator.SendTestNotification(q.QueueArn, r.BucketName, r.RequestID, r.Host, layer.TimeNow(ctx)); err != nil {
				return
			}
		} else {
			h.log.Warn("failed to send test event because notifications is disabled")
		}

		if q.ID == "" {
			completed = true
			conf.QueueConfigurations[i].ID = uuid.NewString()
		}
	}

	return
}

func checkRules(rules []data.FilterRule) error {
	names := make(map[string]struct{})

	for _, r := range rules {
		if r.Name != filterRuleSuffixName && r.Name != filterRulePrefixName {
			return errors.GetAPIError(errors.ErrFilterNameInvalid)
		}
		if _, ok := names[r.Name]; ok {
			if r.Name == filterRuleSuffixName {
				return errors.GetAPIError(errors.ErrFilterNameSuffix)
			}
			return errors.GetAPIError(errors.ErrFilterNamePrefix)
		}

		names[r.Name] = struct{}{}
	}

	return nil
}

func checkEvents(events []string) error {
	for _, e := range events {
		if _, ok := validEvents[e]; !ok {
			return errors.GetAPIError(errors.ErrEventNotification)
		}
	}

	return nil
}

func filterSubjects(conf *data.NotificationConfiguration, eventType, objName string) map[string]string {
	topics := make(map[string]string)

	for _, t := range conf.QueueConfigurations {
		event := false
		for _, e := range t.Events {
			// the second condition is comparison with the events ending with *:
			// s3:ObjectCreated:*, s3:ObjectRemoved:* etc without the last char
			if eventType == e || strings.HasPrefix(eventType, e[:len(e)-1]) {
				event = true
				break
			}
		}

		if !event {
			continue
		}

		filter := true
		for _, f := range t.Filter.Key.FilterRules {
			if f.Name == filterRulePrefixName && !strings.HasPrefix(objName, f.Value) ||
				f.Name == filterRuleSuffixName && !strings.HasSuffix(objName, f.Value) {
				filter = false
				break
			}
		}
		if filter {
			topics[t.ID] = t.QueueArn
		}
	}

	return topics
}
