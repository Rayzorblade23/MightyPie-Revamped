package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/windowManagementAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
)

func main() {
	// Initialize structured logger
	log := logger.New("WindowManagement")
	logger.ReplaceStdLog("WindowManagement")
	
	natsAdapter, err := natsAdapter.New("WindowManagement")
	if err != nil {
		log.Fatal("Failed to initialize NATS adapter: %v", err)
	}

	log.Info("NATS connection established")

    // Create and start the keyboard hook
	windowManagementAdapter, err := windowManagementAdapter.New(natsAdapter)
	if err != nil {
		log.Fatal("Failed to create WindowManagementAdapter: %v", err)
	}

	windowManagementAdapter.Run()
}