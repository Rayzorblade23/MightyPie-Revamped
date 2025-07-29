package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/mouseInputAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New("MouseInputHandler")
	if err != nil {
		panic(err)
	}
	
	mouseInputAdapter := mouseInputAdapter.New(natsAdapter)
	
	mouseInputAdapter.Run()
}