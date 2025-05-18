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
	buttonConfig     ConfigData
	baseButtonConfig ConfigData
	windowsList      core.WindowsUpdate // May not be strictly needed globally if only used in NATS callback scope
	mu               sync.RWMutex
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
	baseButtonConfig = config
	mu.Unlock()
	log.Println("INFO: Initial button configuration loaded.")
	// Removed: PrintConfig(config) // Removed debug print on startup

	buttonUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE")
	windowUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	requestUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_UPDATE")
	requestBaseConfigSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_BASECONFIG")
	baseConfigSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG")
	receiveNewBaseConfigSubject := env.Get("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_UPDATE")

	a.natsAdapter.SubscribeToSubject(requestBaseConfigSubject, core.GetTypeName(a), func(msg *nats.Msg) {
		log.Printf("INFO: Raw config request on '%s'.", msg.Subject)
		mu.RLock()
		rawCopy := baseButtonConfig
		mu.RUnlock()
		a.natsAdapter.PublishMessage(baseConfigSubject, rawCopy)
	})

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
	
		mu.Lock()
		buttonConfig = loadedConfig
		baseButtonConfig = loadedConfig
		mu.Unlock()
	
		log.Println("INFO: Config written and reloaded from disk.")
		// PrintConfig(buttonConfig, false)
		
		a.natsAdapter.PublishMessage(buttonUpdateSubject, loadedConfig)
	})

	a.natsAdapter.SubscribeToSubject(requestUpdateSubject, core.GetTypeName(a), func(msg *nats.Msg) {
		log.Printf("INFO: Config request on '%s'.", msg.Subject)

		currentConfigSnapshot := GetButtonConfig()

		if currentConfigSnapshot == nil {
			log.Printf("WARN: Button config not available for request on '%s'. No config published.", msg.Subject)
			return
		}

		a.natsAdapter.PublishMessage(buttonUpdateSubject, currentConfigSnapshot)
	})

	// Subscribe to window updates for subsequent changes
	a.natsAdapter.SubscribeToSubject(windowUpdateSubject, core.GetTypeName(a), func(msg *nats.Msg) {
		var currentWindows core.WindowsUpdate
		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Printf("ERROR: Failed to decode window update message: %v", err)
			return
		}

		// Update global windows list (if needed elsewhere, otherwise could be local)
		mu.Lock()
		windowsList = make(core.WindowsUpdate, len(currentWindows))
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
			// PrintConfig(updatedConfig, true)
		}
		// No log needed if no changes occurred (avoids log spam)
	})
	return a
}

// Run keeps the adapter alive (if needed, e.g., for non-NATS goroutines)
func (a *ButtonManagerAdapter) Run() error {
	log.Println("INFO: ButtonManagerAdapter running.")
	select {} // Block indefinitely
}
