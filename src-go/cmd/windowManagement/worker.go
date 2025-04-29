package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/windowManagementAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New()
	if err != nil {
		panic(err)
	}

	println("windowManagementAdapter: NATS connection established")

    // Create and start the keyboard hook
	windowManagementAdapter := windowManagementAdapter.New(natsAdapter)

	windowManagementAdapter.Run()
}