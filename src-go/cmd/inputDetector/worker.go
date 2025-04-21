package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/inputDetectionAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
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

	// Initialize the shortcut using the values from the slice
	shortcut := inputDetectionAdapter.Shortcut{
		inputDetectionAdapter.KeyMap[keys[0]],
		inputDetectionAdapter.KeyMap[keys[1]],
		inputDetectionAdapter.KeyMap[keys[2]],
	}

    // Create and start the keyboard hook
	inputDetectionAdapter.MyInputDetector(shortcut)
}