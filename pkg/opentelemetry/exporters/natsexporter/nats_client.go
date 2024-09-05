package natsexporter

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// JetStreamConnection represents a connection to NATS JetStream.
type JetStreamConnection struct {
	NATSConn  *nats.Conn            // The NATS connection.
	JSContext nats.JetStreamContext // The JetStream context.
}

// JetStream represents a message queue using NATS JetStream.
type JetStream struct {
	URL          string              // The NATS server URL.
	Options      []nats.Option       // NATS options.
	StreamName   string              // The name of the JetStream stream.
	Conn         JetStreamConnection // The JetStream connection.
	Subscription *nats.Subscription  // NATS subscription for consuming messages.
	logger       *zap.Logger         // A logger for logging messages and errors.
}

// NewJetstream creates and returns a new JetStream instance.
func NewJetstream(logger *zap.Logger, url string, options []nats.Option, streamName string) *JetStream {
	return &JetStream{
		URL:        url,
		Options:    options,
		StreamName: streamName,
		logger:     logger,
	}
}

// Connect establishes a connection to NATS JetStream.
func (j *JetStream) Connect() error {
	nc, err := nats.Connect(j.URL, j.Options...)
	if err != nil {
		return fmt.Errorf("nats: jetstream connect: failed to connect to NATS server: %w", err)
	}

	j.logger.Info("successfully connected to NATS server")

	jetStreamContext, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("nats: jetstream connect: failed to get JetStream context: %w", err)
	}

	j.logger.Info("successfully got JetStream context")
	j.Conn = JetStreamConnection{NATSConn: nc, JSContext: jetStreamContext}

	return nil
}

// Init initializes the JetStream queue and consumer.
//
// Parameters:
//   - streamConfig: Configuration for creating the JetStream stream.
//
// Returns:
//   - error: An error if there is any issue initializing the JetStream queue and consumer.
func (j *JetStream) Init(streamConfig nats.StreamConfig) error {
	if err := j.CreateStreamIfNotExist(streamConfig); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create stream: %w", err)
	}

	if _, err := j.CreateConsumer(); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create consumer: %w", err)
	}

	return j.CreateSubscription()
}

// CreateStreamIfNotExist creates a JetStream stream if it doesn't already exist.
//
// Parameters:
//   - streamConfig: Configuration for creating the JetStream stream.
//
// Returns:
//   - error: An error if there is any issue creating the stream or if the stream already exists.
func (j *JetStream) CreateStreamIfNotExist(streamConfig nats.StreamConfig) error {
	if j.Conn.JSContext == nil {
		err := errors.New("cannot create stream due to nil connection")
		return fmt.Errorf("nats: jetstream CreateStreamIfNotExist: %w", err)
	}

	var err error

	streamInfo, err := j.Conn.JSContext.StreamInfo(j.StreamName)
	if streamInfo != nil && err == nil {
		j.logger.Info("stream already exists, skipping creation", zap.String("stream", j.StreamName))
		return nil
	}

	if err != nil && err != nats.ErrStreamNotFound {
		j.logger.Warn("error calling JetStream StreamInfo", zap.String("stream", j.StreamName), zap.Error(err))
	}

	j.logger.Info("jetstream: creating stream", zap.String("stream", j.StreamName), zap.Any("config", streamConfig))

	_, err = j.Conn.JSContext.AddStream(&streamConfig)
	if err != nil {
		return fmt.Errorf("jetstream CreateStreamIfNotExist: error while creating stream: %s. %w", j.StreamName, err)
	}

	j.logger.Info("jetstream: stream created", zap.String("stream", j.StreamName))

	return nil
}

// CreateConsumer creates a JetStream consumer for the stream.
//
// Returns:
//   - *nats.ConsumerInfo: Information about the created JetStream consumer.
//   - error: An error if there is any issue creating the consumer.
func (j *JetStream) CreateConsumer() (*nats.ConsumerInfo, error) {
	return j.Conn.JSContext.AddConsumer(j.StreamName, &nats.ConsumerConfig{
		Durable:        j.StreamName + "-TODO",
		DeliverSubject: j.StreamName + "-DeliverSubject",
		DeliverGroup:   j.StreamName + "-TODO",
		AckPolicy:      nats.AckExplicitPolicy,
	})
}

// CreateSubscription creates a NATS subscription for consuming messages from JetStream.
//
// Returns:
//   - error: An error if there is any issue creating the subscription.
func (j *JetStream) CreateSubscription() error {
	subscription, err := j.Conn.NATSConn.QueueSubscribeSync(j.StreamName+"-DeliverSubject", j.StreamName+"-TODO")
	if err != nil {
		return fmt.Errorf("nats: jetstream CreateSubscription: failed to create subscription: %w", err)
	}
	j.Subscription = subscription
	return nil
}

// Publish publishes a protobuf message to the JetStream stream.
//
// Returns:
//   - error: An error if there is any issue publishing the message.
func (j *JetStream) Publish(data []byte) error {
	err := j.publishWithRetry(j.StreamName, data)
	if err != nil {
		return fmt.Errorf("jetstream: failed to publish: %w", err)
	}

	return nil
}

func (j *JetStream) publishWithRetry(subject string, data []byte) error {
	maxRetries := 5
	RetryInterval := 5 * time.Second
	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = j.Conn.JSContext.Publish(subject, data)
		if err == nil {
			return nil
		}

		j.logger.Warn("jetstream: publish attempt failed", zap.Int("attempt", i+1), zap.Error(err))

		time.Sleep(RetryInterval)
	}

	return err
}

// NextMessage retrieves the next message from the JetStream queue and unmarshals it into the provided protobuf message.
func (j *JetStream) NextMessage() ([]byte, error) {
	msg, err := j.Subscription.NextMsg(1 * time.Hour)
	if errors.Is(err, nats.ErrTimeout) {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("jetstream: error while getting the next message: %w", err)
	}

	err = msg.Ack()
	if err != nil {
		return nil, fmt.Errorf("jetstream: failed to ack message: %w", err)
	}

	return msg.Data, nil
}
