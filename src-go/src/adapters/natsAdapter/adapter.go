package natsAdapter

import (
	"encoding/json"
	"log"
	"time"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/nats-io/nats.go"
)

type NatsAdapter struct {
    Connection *nats.Conn
}


func New() (*NatsAdapter, error) {
    // Connect to NATS server with token authentication
    token := env.Get("NATS_AUTH_TOKEN")
    urlStr := env.Get("NATS_SERVER_URL")

    connection, err := nats.Connect(urlStr, nats.Token(token))
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
