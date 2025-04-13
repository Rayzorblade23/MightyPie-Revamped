package main

import (
    "log"
	"encoding/json"
    "github.com/nats-io/nats.go"
)

// Define the struct
type EventMessage struct {
    Name      string  `json:"name"`
    Handle    string  `json:"handle"`
    Something float64 `json:"something"`
}

func handleMessage(msg *nats.Msg) {
	var event EventMessage
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling message data: %v", err)
		return
	}
	log.Printf("Received message on subject %s: %+v", msg.Subject, event)
}	

func main() {
    // Connect to NATS server with token authentication
    token := "5LQ5V4KWPKGRC2LJ8JQGS"
    nc, err := nats.Connect(nats.DefaultURL, nats.Token(token))
    if err != nil {
        log.Fatalf("Error connecting to NATS: %v", err)
    }
    defer nc.Close()

    // Subscribe to a subject
    subject := "mightyPie.events.window.open"
    _, err = nc.Subscribe(subject, handleMessage)
    if err != nil {
        log.Fatalf("Error subscribing to subject: %v", err)
    }

    // Keep the connection alive
    select {}
}