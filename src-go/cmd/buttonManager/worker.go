package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/buttonManagerAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New("ButtonManager")
	if err != nil {
		panic(err)
	}

    // Create and start the keyboard hook
	buttonManagerAdapter := buttonManagerAdapter.New(natsAdapter)

	buttonManagerAdapter.Run()
}