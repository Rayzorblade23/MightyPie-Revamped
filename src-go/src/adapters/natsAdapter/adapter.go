package natsAdapter

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger
var log = logger.New("NATS")
var streamName = os.Getenv("PUBLIC_NATS_STREAM")

type NatsAdapter struct {
	Connection *nats.Conn
}

func New(adapterLabel string) (*NatsAdapter, error) {
	token := os.Getenv("NATS_AUTH_TOKEN")
	urlStr := os.Getenv("NATS_SERVER_URL")

	var connection *nats.Conn
	var err error

	// Retry connecting to NATS with a backoff strategy
	for {
		connection, err = nats.Connect(urlStr, nats.Token(token))
		if err == nil {
			log.Info("[%s] Successfully connected to NATS server.", adapterLabel)
			break // Connection successful
		}

		log.Warn("[%s] Failed to connect to NATS: %v. Retrying in 5 seconds...", adapterLabel, err)
		time.Sleep(5 * time.Second)
	}

	adapter := &NatsAdapter{
		Connection: connection,
	}

	// Only create the JetStream stream in the main coordinator process
	// Workers should just use the existing stream
	if os.Getenv("MIGHTYPIE_WORKER_TYPE") != "worker" {
		// Ensure the JetStream stream is created, with retries
		for {
			if err := adapter.CreateEventsStream(); err == nil {
				log.Debug("[%s] Successfully created or verified JetStream stream.", adapterLabel)
				break // Stream creation successful
			}
			log.Warn("[%s] Failed to create JetStream stream: %v. Retrying in 5 seconds...", adapterLabel, err)
			time.Sleep(5 * time.Second)
		}
	}

	return adapter, nil
}

// Print incoming messages
func PrintMessage(msg *nats.Msg) {
	var decodedMessage map[string]any
	if err := json.Unmarshal(msg.Data, &decodedMessage); err != nil {
		log.Error("Error unmarshaling message data: %v", err)
		return
	}
	log.Info("Received message on subject %s: %+v", msg.Subject, decodedMessage)
}

func (a *NatsAdapter) PublishMessage(subject string, publisherName string, message any) {
	if a.Connection == nil {
		log.Warn("[%s] NATS connection is not established", publisherName)
		return
	}

	msgData, err := json.Marshal(message)
	if err != nil {
		log.Error("[%s] Error marshaling message: %v", publisherName, err)
		return
	}

	err = a.Connection.Publish(subject, msgData)
	if err != nil {
		log.Error("[%s] Error publishing message: %v", publisherName, err)
	} else {
		log.Debug("[%s] Message successfully published to subject: %s", publisherName, subject)
	}
}

func (a *NatsAdapter) SubscribeToSubject(subject string, subscriberName string, handleMessage func(*nats.Msg)) {
	if a.Connection == nil || a.Connection.IsClosed() {
		log.Warn("[%s] Cannot subscribe: Not connected to NATS. Retrying in 1s...", subscriberName)
		time.Sleep(1 * time.Second)
		a.SubscribeToSubject(subject, subscriberName, handleMessage)
		return
	}

	sub, err := a.Connection.Subscribe(subject, func(msg *nats.Msg) {
		log.Debug("[%s] Received message on '%s'", subscriberName, msg.Subject)
		handleMessage(msg)
	})

	if err != nil {
		log.Error("[%s] Failed to subscribe to topic '%s': %v", subscriberName, subject, err)
		return
	}

	log.Info("[%s] Subscribed to topic: %s", subscriberName, subject)
	_ = sub
}

// CreateEventsStream sets up a JetStream stream covering all mightyPie events.
func (a *NatsAdapter) CreateEventsStream() error {
	if a.Connection == nil {
		log.Warn("NATS connection is not established")
		return nats.ErrConnectionClosed
	}

	js, err := a.Connection.JetStream()
	if err != nil {
		log.Error("Error getting JetStream context: %v", err)
		return err
	}

	// Get the stream subject from the environment and add wildcard
	baseSubject := os.Getenv("PUBLIC_NATSSUBJECT_STREAM")
	if baseSubject == "" {
		log.Error("Stream subject not set in environment variable PUBLIC_NATSSUBJECT_STREAM")
		return nats.ErrBadSubject
	}
	streamSubject := baseSubject + ".>"

	streamCfg := &nats.StreamConfig{
		Name:              streamName,
		Subjects:          []string{streamSubject},
		Storage:           nats.FileStorage,
		MaxMsgs:           50,
		MaxMsgsPerSubject: 1,
	}
	_, err = js.AddStream(streamCfg)
	if err == nats.ErrStreamNameAlreadyInUse {
		_, err = js.UpdateStream(streamCfg)
	}
	if err != nil {
		log.Error("Error creating/updating stream: %v", err)
		return err
	}

	log.Debug("Stream '%s' created or already exists with subject: %s", streamName, streamSubject)
	return nil
}

// StreamOverview prints a summary of the MIGHTYPIE_EVENTS stream.
func (a *NatsAdapter) StreamOverview() error {
	if a.Connection == nil {
		log.Warn("NATS connection is not established")
		return nats.ErrConnectionClosed
	}

	js, err := a.Connection.JetStream()

	if err != nil {
		log.Error("Error getting JetStream context: %v", err)
		return err
	}

	info, err := js.StreamInfo(streamName)
	if err != nil {
		log.Error("Error fetching stream info: %v", err)
		return err
	}

	log.Info("Stream config: %+v", info.Config)

	log.Info("Stream: %s", info.Config.Name)
	log.Info("Subjects: %v", info.Config.Subjects)
	log.Info("Messages: %d", info.State.Msgs)
	log.Info("First Sequence: %d", info.State.FirstSeq)
	log.Info("Last Sequence: %d", info.State.LastSeq)
	log.Info("Bytes: %d", info.State.Bytes)
	return nil
}

// SubscribeJetStreamPull sets up a JetStream pull consumer for the given subject.
// If durableName is empty, an ephemeral consumer is created.
func (a *NatsAdapter) SubscribeJetStreamPull(subject, durableName string, handler func(*nats.Msg)) error {
	if a.Connection == nil {
		return nats.ErrConnectionClosed
	}
	js, err := a.Connection.JetStream()
	if err != nil {
		return err
	}

	var sub *nats.Subscription
	if durableName == "" {
		// Ephemeral consumer: do not specify durable name or BindStream
		sub, err = js.PullSubscribe(subject, "", nats.BindStream(streamName))
	} else {
		// Durable consumer
		sub, err = js.PullSubscribe(subject, durableName, nats.BindStream(streamName))
	}
	if err != nil {
		return err
	}

	go func() {
		for {
			msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
			if err != nil && err != nats.ErrTimeout {
				log.Error("Error fetching JetStream messages: %v", err)
				time.Sleep(time.Second)
				continue
			}
			for _, msg := range msgs {
				handler(msg)
				msg.Ack()
			}
		}
	}()
	return nil
}

func (a *NatsAdapter) PurgeEventsStream() error {
	if a.Connection == nil {
		log.Warn("NATS connection is not established")
		return nats.ErrConnectionClosed
	}
	js, err := a.Connection.JetStream()
	if err != nil {
		log.Error("Error getting JetStream context: %v", err)
		return err
	}
	err = js.PurgeStream(streamName)
	if err != nil {
		log.Error("Error purging stream: %v", err)
		return err
	}
	log.Info("Stream '%s' purged successfully.", streamName)
	return nil
}
