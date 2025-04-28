package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/pieButtonExecutionAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New()
	if err != nil {
		panic(err)
	}

	println("PieButtonExecutionAdapter: NATS connection established")

	pieButtonExecutionAdapter := pieButtonExecutionAdapter.New(natsAdapter)

	pieButtonExecutionAdapter.Run()
}