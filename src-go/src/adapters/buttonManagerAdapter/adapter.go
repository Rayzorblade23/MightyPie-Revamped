package buttonManagerAdapter

import (
	"encoding/json"
	"log"
	"maps"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"                  // Verify path
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter" // Verify path
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

var (
	// Assumes ConfigData is map[string]MenuConfig (MenuID -> PageID -> PageConfiguration)
	buttonConfig ConfigData
	windowsList  core.WindowsUpdate
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

	buttonUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE")
	windowUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	baseConfigSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG")
	receiveNewBaseConfigSubject := env.Get("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_UPDATE")

	config, err := ReadButtonConfig()
	if err != nil {
		log.Fatalf("FATAL: Failed to read initial button configuration: %v", err)
	}

	updateButtonConfig(config)
	log.Println("INFO: Initial button configuration loaded.")
	// Removed: PrintConfig(config) // Removed debug print on startup

	a.natsAdapter.PublishMessage(baseConfigSubject, config)

	a.natsAdapter.SubscribeToSubject(receiveNewBaseConfigSubject, core.GetTypeName(a), func(msg *nats.Msg) {
		log.Printf("INFO: Raw config coming in on '%s'.", msg.Subject)

		var newConfig ConfigData
		if err := json.Unmarshal(msg.Data, &newConfig); err != nil {
			log.Printf("ERROR: Failed to unmarshal config: %v", err)
			return
		}

		if err := WriteButtonConfig(newConfig); err != nil {
			log.Printf("ERROR: Failed to write config: %v", err)
			return
		}

		// Read it back in to update in-memory state
		loadedConfig, err := ReadButtonConfig()
		if err != nil {
			log.Printf("ERROR: Failed to reload config after write: %v", err)
			return
		}

		updateButtonConfig(loadedConfig)

		log.Println("INFO: Config written and reloaded from disk.")
		// PrintConfig(buttonConfig, false)

		a.natsAdapter.PublishMessage(baseConfigSubject, loadedConfig)
	})

	// Subscribe to window updates for subsequent changes
	a.natsAdapter.SubscribeToSubject(windowUpdateSubject, core.GetTypeName(a), func(msg *nats.Msg) {
		var currentWindows core.WindowsUpdate
		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Printf("ERROR: Failed to decode window update message: %v", err)
			return
		}

		updateWindowsList(currentWindows)

		currentConfigSnapshot := GetButtonConfig() // Get clean snapshot

		// Process updates
		processedConfig, err := a.processWindowUpdate(currentConfigSnapshot, currentWindows)
		if err != nil {
			log.Printf("ERROR: Failed to process window update for button config: %v", err)
			return
		}

		// Publish ONLY if changes were detected
		if processedConfig != nil {
			// Update global state first
			updateButtonConfig(processedConfig)
			log.Println("INFO: Button configuration updated (due to window event) and will be published.")
			// Publish the updated configuration object
			a.natsAdapter.PublishMessage(buttonUpdateSubject, processedConfig)
			// PrintConfig(processedConfig, true)
		}
	})
	return a
}

// updateButtonConfig safely updates the global buttonConfig variable.
func updateButtonConfig(config ConfigData) {
	mu.Lock()
	buttonConfig = config
	mu.Unlock()
}

// updateWindowsList safely updates the global windowsList variable.
func updateWindowsList(newList core.WindowsUpdate) {
	mu.Lock()
	windowsList = make(core.WindowsUpdate, len(newList))
	maps.Copy(windowsList, newList)
	mu.Unlock()
}

// Run keeps the adapter alive (if needed, e.g., for non-NATS goroutines)
func (a *ButtonManagerAdapter) Run() error {
	log.Println("INFO: ButtonManagerAdapter running.")
	select {} // Block indefinitely
}
