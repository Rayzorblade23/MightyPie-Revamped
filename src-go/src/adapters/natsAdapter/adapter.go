package natsAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/nats-io/nats.go"
)

type NatsAdapter struct {
	Connection *nats.Conn
}

func New() (*NatsAdapter, error) {
	token := env.Get("NATS_AUTH_TOKEN")
	urlStr := env.Get("NATS_SERVER_URL")

	var connection *nats.Conn
	var err error

	// Retry connecting to NATS with a backoff strategy
	for {
		connection, err = nats.Connect(urlStr, nats.Token(token))
		if err == nil {
			log.Println("Successfully connected to NATS server.")
			break // Connection successful
		}

		log.Printf("Failed to connect to NATS: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}

	adapter := &NatsAdapter{
		Connection: connection,
	}

	// Ensure the JetStream stream is created, with retries
	for {
		if err := adapter.CreateEventsStream(); err == nil {
			log.Println("Successfully created or verified JetStream stream.")
			break // Stream creation successful
		}
		log.Printf("Failed to create JetStream stream: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}

	return adapter, nil
}

// Print incoming messages
func PrintMessage(msg *nats.Msg) {
	var decodedMessage map[string]interface{}
	if err := json.Unmarshal(msg.Data, &decodedMessage); err != nil {
		log.Printf("Error unmarshaling message data: %v", err)
		return
	}
	log.Printf("Received message on subject %s: %+v", msg.Subject, decodedMessage)
}

func (a *NatsAdapter) PublishMessage(subject string, message interface{}) {
	if a.Connection == nil {
		log.Println("NATS connection is not established")
		return
	}

	msgData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	err = a.Connection.Publish(subject, msgData)
	if err != nil {
		log.Printf("Error publishing message: %v", err)
	} else {
		log.Printf("Message successfully published to subject: %s", subject)
	}
}

func (a *NatsAdapter) SubscribeToSubject(subject string, subscriberName string, handleMessage func(*nats.Msg)) {
	if a.Connection == nil || a.Connection.IsClosed() {
		log.Printf("[%s] Cannot subscribe: Not connected to NATS. Retrying in 1s...", subscriberName)
		time.Sleep(1 * time.Second)
		a.SubscribeToSubject(subject, subscriberName, handleMessage)
		return
	}

	sub, err := a.Connection.Subscribe(subject, func(msg *nats.Msg) {
		log.Printf("[%s] Received message on '%s'", subscriberName, msg.Subject)
		handleMessage(msg)
	})

	if err != nil {
		log.Printf("[%s] Failed to subscribe to topic '%s': %v", subscriberName, subject, err)
		return
	}

	log.Printf("[%s] Subscribed to topic: %s", subscriberName, subject)
	_ = sub
}

// CreateEventsStream sets up a JetStream stream covering all mightyPie events.
func (a *NatsAdapter) CreateEventsStream() error {
	if a.Connection == nil {
		log.Println("NATS connection is not established")
		return nats.ErrConnectionClosed
	}

	js, err := a.Connection.JetStream()
	if err != nil {
		log.Printf("Error getting JetStream context: %v", err)
		return err
	}

	// Get the stream subject from the environment and add wildcard
	baseSubject := env.Get("PUBLIC_NATSSUBJECT_STREAM")
	if baseSubject == "" {
		log.Println("Stream subject not set in environment variable PUBLIC_NATSSUBJECT_STREAM")
		return nats.ErrBadSubject
	}
	streamSubject := baseSubject + ".>"

	streamCfg := &nats.StreamConfig{
		Name:              "MIGHTYPIE_EVENTS",
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
		log.Printf("Error creating/updating stream: %v", err)
		return err
	}

	log.Printf("Stream 'MIGHTYPIE_EVENTS' created or already exists with subject: %s", streamSubject)
	return nil
}

// StreamOverview prints a summary of the MIGHTYPIE_EVENTS stream.
func (a *NatsAdapter) StreamOverview() error {
	if a.Connection == nil {
		log.Println("NATS connection is not established")
		return nats.ErrConnectionClosed
	}

	js, err := a.Connection.JetStream()

	if err != nil {
		log.Printf("Error getting JetStream context: %v", err)
		return err
	}

	info, err := js.StreamInfo("MIGHTYPIE_EVENTS")
	if err != nil {
		log.Printf("Error fetching stream info: %v", err)
		return err
	}

	log.Printf("Stream config: %+v", info.Config)

	log.Printf("Stream: %s", info.Config.Name)
	log.Printf("Subjects: %v", info.Config.Subjects)
	log.Printf("Messages: %d", info.State.Msgs)
	log.Printf("First Sequence: %d", info.State.FirstSeq)
	log.Printf("Last Sequence: %d", info.State.LastSeq)
	log.Printf("Bytes: %d", info.State.Bytes)
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
		sub, err = js.PullSubscribe(subject, "", nats.BindStream("MIGHTYPIE_EVENTS"))
	} else {
		// Durable consumer
		sub, err = js.PullSubscribe(subject, durableName, nats.BindStream("MIGHTYPIE_EVENTS"))
	}
	if err != nil {
		return err
	}

	go func() {
		for {
			msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
			if err != nil && err != nats.ErrTimeout {
				fmt.Printf("Error fetching JetStream messages: %v\n", err)
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
		log.Println("NATS connection is not established")
		return nats.ErrConnectionClosed
	}
	js, err := a.Connection.JetStream()
	if err != nil {
		log.Printf("Error getting JetStream context: %v", err)
		return err
	}
	err = js.PurgeStream("MIGHTYPIE_EVENTS")
	if err != nil {
		log.Printf("Error purging stream: %v", err)
		return err
	}
	log.Println("MIGHTYPIE_EVENTS stream purged successfully.")
	return nil
}
