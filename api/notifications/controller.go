package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/handler"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	DefaultTimeout = 30 * time.Second

	// EventVersion23 is used for lifecycle, tiering, objectACL, objectTagging, object restoration notifications.
	EventVersion23 = "2.3"
	// EventVersion22 is used for replication notifications.
	EventVersion22 = "2.2"
	// EventVersion21 is used for all other notification types.
	EventVersion21 = "2.1"
)

type (
	Options struct {
		URL                       string
		TLSCertFilepath           string
		TLSAuthPrivateKeyFilePath string
		Timeout                   time.Duration
		RootCAFiles               []string
	}

	Controller struct {
		logger              *zap.Logger
		taskQueueConnection *nats.Conn
		jsClient            nats.JetStreamContext
		handlers            map[string]Stream
		mu                  sync.RWMutex
	}

	Stream struct {
		h  layer.MsgHandler
		ch chan *nats.Msg
	}

	TestEvent struct {
		Service   string
		Event     string
		Time      time.Time
		Bucket    string
		RequestID string
		HostID    string
	}

	Event struct {
		Records []EventRecord `json:"Records"`
	}

	EventRecord struct {
		EventVersion      string            `json:"eventVersion"`
		EventSource       string            `json:"eventSource"`         // frostfs:s3
		AWSRegion         string            `json:"awsRegion,omitempty"` // empty
		EventTime         time.Time         `json:"eventTime"`
		EventName         string            `json:"eventName"`
		UserIdentity      UserIdentity      `json:"userIdentity"`
		RequestParameters RequestParameters `json:"requestParameters"`
		ResponseElements  map[string]string `json:"responseElements"`
		S3                S3Entity          `json:"s3"`
	}

	UserIdentity struct {
		PrincipalID string `json:"principalId"`
	}

	RequestParameters struct {
		SourceIPAddress string `json:"sourceIPAddress"`
	}

	S3Entity struct {
		SchemaVersion   string `json:"s3SchemaVersion"`
		ConfigurationID string `json:"configurationId,omitempty"`
		Bucket          Bucket `json:"bucket"`
		Object          Object `json:"object"`
	}

	Bucket struct {
		Name          string       `json:"name"`
		OwnerIdentity UserIdentity `json:"ownerIdentity,omitempty"`
		Arn           string       `json:"arn,omitempty"`
	}

	Object struct {
		Key       string `json:"key"`
		Size      int64  `json:"size,omitempty"`
		VersionID string `json:"versionId,omitempty"`
		ETag      string `json:"eTag,omitempty"`
		Sequencer string `json:"sequencer,omitempty"`
	}
)

func NewController(p *Options, l *zap.Logger) (*Controller, error) {
	ncopts := []nats.Option{
		nats.Timeout(p.Timeout),
	}

	if len(p.TLSCertFilepath) != 0 && len(p.TLSAuthPrivateKeyFilePath) != 0 {
		ncopts = append(ncopts, nats.ClientCert(p.TLSCertFilepath, p.TLSAuthPrivateKeyFilePath))
	}
	if len(p.RootCAFiles) != 0 {
		ncopts = append(ncopts, nats.RootCAs(p.RootCAFiles...))
	}

	nc, err := nats.Connect(p.URL, ncopts...)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("get jet stream: %w", err)
	}

	return &Controller{
		logger:              l,
		taskQueueConnection: nc,
		jsClient:            js,
		handlers:            make(map[string]Stream),
	}, nil
}

func (c *Controller) Subscribe(ctx context.Context, topic string, handler layer.MsgHandler) error {
	ch := make(chan *nats.Msg, 1)

	c.mu.RLock()
	if _, ok := c.handlers[topic]; ok {
		return fmt.Errorf("already subscribed to topic '%s'", topic)
	}
	c.mu.RUnlock()

	if _, err := c.jsClient.AddStream(&nats.StreamConfig{Name: topic}); err != nil {
		return fmt.Errorf("add stream: %w", err)
	}

	if _, err := c.jsClient.ChanSubscribe(topic, ch); err != nil {
		return fmt.Errorf("could not subscribe: %w", err)
	}

	c.mu.Lock()
	c.handlers[topic] = Stream{
		h:  handler,
		ch: ch,
	}
	c.mu.Unlock()

	return nil
}

func (c *Controller) Listen(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, stream := range c.handlers {
		go func(stream Stream) {
			for {
				select {
				case msg := <-stream.ch:
					if err := stream.h.HandleMessage(ctx, msg); err != nil {
						c.logger.Error("could not handle message", zap.Error(err))
					} else if err = msg.Ack(); err != nil {
						c.logger.Error("could not ACK message", zap.Error(err))
					}
				case <-ctx.Done():
					return
				}
			}
		}(stream)
	}
}

func (c *Controller) SendNotifications(topics map[string]string, p *handler.SendNotificationParams) error {
	event := prepareEvent(p)

	for id, topic := range topics {
		event.Records[0].S3.ConfigurationID = id
		msg, err := json.Marshal(event)
		if err != nil {
			c.logger.Error("couldn't marshal an event", zap.String("subject", topic), zap.Error(err))
		}
		if err = c.publish(topic, msg); err != nil {
			c.logger.Error("couldn't send an event to topic", zap.String("subject", topic), zap.Error(err))
		}
	}

	return nil
}

func (c *Controller) SendTestNotification(topic, bucketName, requestID, HostID string, now time.Time) error {
	event := &TestEvent{
		Service:   "FrostFS S3",
		Event:     "s3:TestEvent",
		Time:      now,
		Bucket:    bucketName,
		RequestID: requestID,
		HostID:    HostID,
	}

	msg, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("couldn't marshal test event: %w", err)
	}

	return c.publish(topic, msg)
}

func prepareEvent(p *handler.SendNotificationParams) *Event {
	return &Event{
		Records: []EventRecord{
			{
				EventVersion: EventVersion21,
				EventSource:  "frostfs:s3",
				AWSRegion:    "",
				EventTime:    p.Time,
				EventName:    p.Event,
				UserIdentity: UserIdentity{
					PrincipalID: p.User,
				},
				RequestParameters: RequestParameters{
					SourceIPAddress: p.ReqInfo.RemoteHost,
				},
				ResponseElements: nil,
				S3: S3Entity{
					SchemaVersion: "1.0",
					// ConfigurationID is skipped and will be placed later
					Bucket: Bucket{
						Name:          p.BktInfo.Name,
						OwnerIdentity: UserIdentity{PrincipalID: p.BktInfo.Owner.String()},
						Arn:           p.BktInfo.Name,
					},
					Object: Object{
						Key:       p.NotificationInfo.Name,
						Size:      p.NotificationInfo.Size,
						VersionID: p.NotificationInfo.Version,
						ETag:      p.NotificationInfo.HashSum,
						Sequencer: "",
					},
				},
			},
		},
	}
}

func (c *Controller) publish(topic string, msg []byte) error {
	if _, err := c.jsClient.Publish(topic, msg); err != nil {
		return fmt.Errorf("couldn't send  event: %w", err)
	}

	return nil
}
