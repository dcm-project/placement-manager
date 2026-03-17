package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/dcm-project/placement-manager/internal/store"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// VMEvent represents a VM status event payload.
// TODO: This must have somewhere defined common schema
type VMEvent struct {
	Id        string    `json:"id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// StatusConsumer subscribes to resource status events from the messaging system
// using JetStream and updates instance records in the database.
type StatusConsumer struct {
	conn         *nats.Conn
	js           jetstream.JetStream
	consumeCtx   jetstream.ConsumeContext
	store        store.Store
	subject      string
	streamName   string
	consumerName string
}

// New creates a new StatusConsumer connected to the given NATS URL.
func New(natsURL, subject string, st store.Store, opts ...Option) (*StatusConsumer, error) {
	conn, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Println("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	sc := &StatusConsumer{
		conn:         conn,
		js:           js,
		store:        st,
		subject:      subject,
		streamName:   "dcm-status",
		consumerName: "placement-manager",
	}
	for _, o := range opts {
		o(sc)
	}
	return sc, nil
}

// Option configures a StatusConsumer.
type Option func(*StatusConsumer)

// WithStreamName sets the JetStream stream name.
func WithStreamName(name string) Option {
	return func(c *StatusConsumer) { c.streamName = name }
}

// WithConsumerName sets the JetStream durable consumer name.
func WithConsumerName(name string) Option {
	return func(c *StatusConsumer) { c.consumerName = name }
}

// Start creates the JetStream stream and consumer, then begins processing messages.
func (c *StatusConsumer) Start(ctx context.Context) error {
	stream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     c.streamName,
		Subjects: []string{c.subject},
	})
	if err != nil {
		return fmt.Errorf("failed to create/update stream %s: %w", c.streamName, err)
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   c.consumerName,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update consumer %s: %w", c.consumerName, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		c.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}
	c.consumeCtx = cc

	log.Printf("StatusConsumer subscribed to %s (stream=%s, consumer=%s)", c.subject, c.streamName, c.consumerName)
	return nil
}

// Stop stops the consumer and closes the NATS connection.
func (c *StatusConsumer) Stop() {
	if c.consumeCtx != nil {
		c.consumeCtx.Stop()
	}
	c.conn.Close()
	log.Println("StatusConsumer stopped")
}

func (c *StatusConsumer) handleMessage(ctx context.Context, msg jetstream.Msg) {
	serviceType, err := parseSubject(msg.Subject())
	if err != nil {
		log.Printf("Failed to parse NATS subject: %v", err)
		// Ack to avoid redelivery of unparseable subjects
		_ = msg.Ack()
		return
	}

	var event cloudevents.Event
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("Failed to parse CloudEvent: %v", err)
		_ = msg.Ack()
		return
	}

	var instanceID, status, statusMessage string

	switch serviceType {
	case "vm":
		var payload VMEvent
		if err := json.Unmarshal(event.Data(), &payload); err != nil {
			log.Printf("Failed to deserialize %s event payload: %v", serviceType, err)
			_ = msg.Ack()
			return
		}
		instanceID = payload.Id
		status = payload.Status
		statusMessage = payload.Message
	default:
		// container, cluster — use the same event shape for now
		var payload VMEvent
		if err := json.Unmarshal(event.Data(), &payload); err != nil {
			log.Printf("Failed to deserialize %s event payload: %v", serviceType, err)
			_ = msg.Ack()
			return
		}
		instanceID = payload.Id
		status = payload.Status
		statusMessage = payload.Message
	}

	if instanceID == "" {
		log.Printf("Event missing instance ID, discarding")
		_ = msg.Ack()
		return
	}

	if err := c.store.Resource().UpdateStatus(ctx, instanceID, status, statusMessage); err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			log.Printf("No resource found for instance %s, skipping status update", instanceID)
			_ = msg.Ack()
			return
		}
		log.Printf("Failed to update status for instance %s: %v", instanceID, err)
		// Don't ack — allow redelivery on transient DB errors
		_ = msg.Nak()
		return
	}

	log.Printf("Updated instance %s status to %s", instanceID, status)
	_ = msg.Ack()
}

// parseSubject extracts serviceType from the NATS subject.
// Expected format: dcm.{serviceType}
func parseSubject(subject string) (serviceType string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected subject format: expected 2 parts, got %d", len(parts))
	}
	return parts[1], nil
}
