package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/shortcutDetectionAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New()
	if err != nil {
		panic(err)
	}

	println("ShortcutDetectionAdapter: NATS connection established")

	shortcutDetectionAdapter.New(natsAdapter)

	select {}
	// Block forever so the process doesn't exit
}