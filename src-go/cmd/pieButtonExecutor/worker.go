package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/pieButtonExecutionAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
)

func main() {
	natsAdapter, err := natsAdapter.New("PieButtonExecutor")
	if err != nil {
		panic(err)
	}

	pieButtonExecutionAdapter := pieButtonExecutionAdapter.New(natsAdapter)

	pieButtonExecutionAdapter.Run()
}