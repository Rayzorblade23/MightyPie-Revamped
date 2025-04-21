package main

import (
	"encoding/json"
	"fmt"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/mouseInputAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

func main() {
	natsConnection := natsAdapter.StartConnection()
	defer natsConnection.Close()

	mouseInputAdapter.Run()

	type EventMessage struct {
		ShortcutDetected int `json:"shortcutDetected"`
	}

	natsAdapter.SubscribeToTopic("mightyPie.events.shortcut.detected", func(msg *nats.Msg) {
    
		var event EventMessage
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			println("Failed to decode message: %v", err)
			return
		}

		fmt.Printf("Shortcut detected: %+v", event)
	})


	println("Mouse input handler started")
}