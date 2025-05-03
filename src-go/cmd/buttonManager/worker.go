package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/buttonManagerAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New()
	if err != nil {
		panic(err)
	}

	println("ButtonManagerAdapter: NATS connection established")

    // Create and start the keyboard hook
	buttonManagerAdapter := buttonManagerAdapter.New(natsAdapter)

	buttonManagerAdapter.Run()
}