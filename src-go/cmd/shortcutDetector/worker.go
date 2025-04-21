package main

import (
	"encoding/json"
	"fmt"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/shortcutDetectionAdapter"
	"github.com/nats-io/nats.go"
)

func main() {
	natsConnection := natsAdapter.StartConnection()
	defer natsConnection.Close()

	println("NATS connection established")

	keys := []string{
		"Shift",
		"Ctrl",
		"D",
	}


	type EventMessage struct {
		ShortcutDetected int `json:"shortcutDetected"`
	}

	natsAdapter.SubscribeToTopic("mightyPie.events.window.open", func(msg *nats.Msg) {
    
		var event EventMessage
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			println("Failed to decode message: %v", err)
			return
		}

		fmt.Printf("Shortcut detected: %+v", event)
	})

	// Initialize the shortcut using the values from the slice
	shortcut := shortcutDetectionAdapter.Shortcut{
		shortcutDetectionAdapter.KeyMap[keys[0]],
		shortcutDetectionAdapter.KeyMap[keys[1]],
		shortcutDetectionAdapter.KeyMap[keys[2]],
	}

    // Create and start the keyboard hook
	shortcutDetectionAdapter.ShortcutDetector(shortcut)
}