package buttonManagerAdapter

import (
	"encoding/json"
	"log"
	"maps"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"                  // Verify path
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter" // Verify path
	"github.com/nats-io/nats.go"
)

var (
	// Assumes ConfigData is map[string]MenuConfig (ProfileID -> MenuID -> ButtonMap)
	buttonConfig ConfigData
	windowsList  WindowsUpdate // May not be strictly needed globally if only used in NATS callback scope
	mu           sync.RWMutex
)

type ButtonManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

// New creates and initializes the ButtonManagerAdapter
func New(natsAdapter *natsAdapter.NatsAdapter) *ButtonManagerAdapter {
	if natsAdapter == nil {
		log.Fatalf("FATAL: NATS Adapter dependency cannot be nil") // Fail fast if dependency missing
	}
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	config, err := ReadButtonConfig() // Assumed from config_access.go
	if err != nil {
		// Consider retrying or default config instead of fatal
		log.Fatalf("FATAL: Failed to read initial button configuration: %v", err)
	}

	mu.Lock()
	buttonConfig = config
	mu.Unlock()
	log.Println("INFO: Initial button configuration loaded.")
	// Removed: PrintConfig(config) // Removed debug print on startup

	buttonUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE")
	windowUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")

	// Subscribe to window updates for subsequent changes
	a.natsAdapter.SubscribeToSubject(windowUpdateSubject, func(msg *nats.Msg) {
		var currentWindows WindowsUpdate
		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Printf("ERROR: Failed to decode window update message: %v", err)
			return
		}

		// Update global windows list (if needed elsewhere, otherwise could be local)
		mu.Lock()
		windowsList = make(WindowsUpdate, len(currentWindows))
		maps.Copy(windowsList, currentWindows)
		mu.Unlock()

		currentConfigSnapshot := GetButtonConfig() // Get clean snapshot

		// Process updates
		updatedConfig, err := a.processWindowUpdate(currentConfigSnapshot, currentWindows)
		if err != nil {
			log.Printf("ERROR: Failed to process window update for button config: %v", err)
			// Depending on error type, maybe attempt recovery or just skip update?
			return
		}

		// Publish ONLY if changes were detected
		if updatedConfig != nil {
			// Update global state first
			mu.Lock()
			buttonConfig = updatedConfig
			mu.Unlock()
			log.Println("INFO: Button configuration updated (due to window event) and will be published.")
			// Publish the updated configuration object
			a.natsAdapter.PublishMessage(buttonUpdateSubject, updatedConfig)
			PrintConfig(updatedConfig, true)
		}
		// No log needed if no changes occurred (avoids log spam)
	})

	log.Printf("INFO: ButtonManagerAdapter subscribed to window updates on '%s'", windowUpdateSubject)
	return a
}

// Run keeps the adapter alive (if needed, e.g., for non-NATS goroutines)
func (a *ButtonManagerAdapter) Run() error {
	log.Println("INFO: ButtonManagerAdapter running.")
	select {} // Block indefinitely
}
