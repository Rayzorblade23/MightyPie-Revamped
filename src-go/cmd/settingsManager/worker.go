package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/settingsManagerAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New("SettingsManager")
	if err != nil {
		panic(err)
	}

    // Create and start the keyboard hook
	settingsManagerAdapter := settingsManagerAdapter.New(natsAdapter)

	settingsManagerAdapter.Run()
}