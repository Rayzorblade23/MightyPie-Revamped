package inputDetectionAdapter

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type EventMessage struct {
    ShortcutDetected int           `json:"shortcutDetected"`
}

// Global NATS connection
var natsConnection *nats.Conn

// Handle incoming messages
func HandleMessage(msg *nats.Msg) {
    var event EventMessage
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        log.Printf("Error unmarshaling message data: %v", err)
        return
    }
    log.Printf("Received message on subject %s: %+v", msg.Subject, event)
}

// Function to publish a message
func PublishMessage(subject string, message EventMessage) {
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


func StartNATS_Connection () {
	go start_Connection()
}


func start_Connection () {
    // Connect to NATS server with token authentication
    token := "5LQ5V4KWPKGRC2LJ8JQGS"
	var err error
    natsConnection, err = nats.Connect(nats.DefaultURL, nats.Token(token))
    if err != nil {
        log.Fatalf("Error connecting to NATS: %v", err)
    }
    defer natsConnection.Close()

    // // Subscribe to a subject
    // subject := "mightyPie.events.window.open"
    // _, err = natsConnection.Subscribe(subject, HandleMessage)
    // if err != nil {
    //     log.Fatalf("Error subscribing to subject: %v", err)
    // }

    // Keep the connection alive
    select {}
}
