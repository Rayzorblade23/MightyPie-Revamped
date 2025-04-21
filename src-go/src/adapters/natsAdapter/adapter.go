package natsAdapter

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Global NATS connection
var natsConnection *nats.Conn

// Print incoming messages
func PrintMessage(msg *nats.Msg) {
    var event map[string]interface{}
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        log.Printf("Error unmarshaling message data: %v", err)
        return
    }
    log.Printf("Received message on subject %s: %+v", msg.Subject, event)
}


func PublishMessage(subject string, message interface{}) {
    if natsConnection == nil {
        log.Println("NATS connection is not established")
        return
    }

    msgData, err := json.Marshal(message)
    if err != nil {
        log.Printf("Error marshaling message: %v", err)
        return
    }

    err = natsConnection.Publish(subject, msgData)
    if err != nil {
        log.Printf("Error publishing message: %v", err)
    } else {
        log.Printf("Message successfully published to subject: %s", subject)
    }
}

func SubscribeToTopic(subject string, handleMessage func(*nats.Msg)) {
    if natsConnection == nil || natsConnection.IsClosed() {
        log.Printf("Cannot subscribe: Not connected to NATS. Retrying in 1s...")
        time.Sleep(1 * time.Second)
        SubscribeToTopic(subject, handleMessage)
        return
    }

    sub, err := natsConnection.Subscribe(subject, func(msg *nats.Msg) {
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




func StartConnection () (*nats.Conn) {
    // Connect to NATS server with token authentication
    token := "5LQ5V4KWPKGRC2LJ8JQGS"
	var err error
    natsConnection, err = nats.Connect(nats.DefaultURL, nats.Token(token))
    if err != nil {
        log.Fatalf("Error connecting to NATS: %v", err)
    }

    // // Subscribe to a subject
    // subject := "mightyPie.events.window.open"
    // _, err = natsConnection.Subscribe(subject, PrintMessage)
    // if err != nil {
    //     log.Fatalf("Error subscribing to subject: %v", err)
    // }

    return natsConnection
}
