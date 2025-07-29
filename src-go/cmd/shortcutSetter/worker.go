package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/shortcutSetterAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New("ShortcutSetter")
	if err != nil {
		panic(err)
	}

	shortcutSetterAdapter.New(natsAdapter)

	// Block forever so the process doesn't exit
	select {}
}
