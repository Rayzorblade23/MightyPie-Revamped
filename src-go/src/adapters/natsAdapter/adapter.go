package natsAdapter

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsAdapter struct {
    Connection *nats.Conn
}


func New() (*NatsAdapter, error) {
    // Connect to NATS server with token authentication
    token := "5LQ5V4KWPKGRC2LJ8JQGS"

    connection, err := nats.Connect(nats.DefaultURL, nats.Token(token))
    if err != nil {
        log.Fatalf("Error connecting to NATS: %v", err)
        return nil, err
    }

    return &NatsAdapter{
        Connection: connection,
    }, nil
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

func (a *NatsAdapter) SubscribeToSubject(subject string, handleMessage func(*nats.Msg)) {
    if a.Connection == nil || a.Connection.IsClosed() {
        log.Printf("Cannot subscribe: Not connected to NATS. Retrying in 1s...")
        time.Sleep(1 * time.Second)
        a.SubscribeToSubject(subject, handleMessage)
        return
    }

    sub, err := a.Connection.Subscribe(subject, func(msg *nats.Msg) {
        log.Printf("Received message on '%s': %s", msg.Subject, string(msg.Data))
        handleMessage(msg)
    })

    if err != nil {
        log.Printf("Failed to subscribe to topic '%s': %v", subject, err)
        return
    }

    log.Printf("Subscribed to topic: %s", subject)

    // Optional: keep sub alive or handle lifecycle explicitly
    // Add cleanup/Unsubscribe logic as needed
    _ = sub
}
