package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/mouseInputAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New()
	if err != nil {
		panic(err)
	}
	
	mouseInputAdapter := mouseInputAdapter.New(natsAdapter)

	println("Mouse input handler started")
	
	mouseInputAdapter.Run()
}