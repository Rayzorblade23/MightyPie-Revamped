package buttonManagerAdapter

import (
	"encoding/json"
	"maps"
	"os"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("ButtonManager")

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
		log.Fatal("FATAL: NATS Adapter dependency cannot be nil") // Fail fast if dependency missing
	}
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	windowUpdateSubject := os.Getenv("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	// New unified full-config flow subjects
	backendFullConfigSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE")
	liveButtonsSubject := os.Getenv("PUBLIC_NATSSUBJECT_LIVEBUTTONCONFIG")
	fillGapsSubject := os.Getenv("PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS")

	// Do NOT read/write the unified on-disk config here. The unified file is owned by PieMenuConfigManager.
	// Initialize with an empty in-memory config and wait for full-config updates from PieMenuConfigManager.
	updateButtonConfig(make(ConfigData))
	log.Info("Waiting for full config from PieMenuConfigManager...")

	// Removed legacy base config update subscription; full-config flow is the source of truth

	// Subscribe to full-config backend updates and extract buttons for this adapter
	if backendFullConfigSubject != "" {
		a.natsAdapter.SubscribeToSubject(backendFullConfigSubject, func(msg *nats.Msg) {
			// Only care about the buttons field from the unified config
			var payload struct {
				Buttons ConfigData `json:"buttons"`
			}
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				log.Error("Failed to unmarshal full config (buttons): %v", err)
				return
			}
			if len(payload.Buttons) == 0 {
				log.Warn("Full config update contained empty buttons; ignoring")
				return
			}

			// Update in-memory state and publish buttons to live subject
			updateButtonConfig(payload.Buttons)
			a.natsAdapter.PublishMessage(liveButtonsSubject, payload.Buttons)
			a.natsAdapter.PublishMessage(windowUpdateSubject, windowsList)
			log.Info("Processed full backend update and republished buttons to '%s'", liveButtonsSubject)
		})
	}

	// Subscribe to window updates for subsequent changes
	a.natsAdapter.SubscribeToSubject(windowUpdateSubject, func(msg *nats.Msg) {
		var currentWindows core.WindowsUpdate
		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Error("Failed to decode window update message: %v", err)
			return
		}

		// PrintWindowList(currentWindows)
		updateWindowsList(currentWindows)

		currentConfigSnapshot := GetButtonConfig() // Get clean snapshot

		// Process updates
		processedConfig, err := a.processWindowUpdate(currentConfigSnapshot, currentWindows)
		if err != nil {
			log.Error("Failed to process window update for button config: %v", err)
			return
		}

		// Publish ONLY if changes were detected
		if processedConfig != nil {
			// Update global state first
			updateButtonConfig(processedConfig)
			log.Info("Button configuration updated (due to window event) and will be published.")
			// Publish the updated configuration
			a.natsAdapter.PublishMessage(liveButtonsSubject, processedConfig)
			// PrintConfig(processedConfig, true)
		}
	})

	// Gap-filling/compaction subscription
	a.natsAdapter.SubscribeToSubject(fillGapsSubject, func(msg *nats.Msg) {
		mu.Lock()
		currentConfig := buttonConfig
		mu.Unlock()
		gapFilledConfig, cleared := FillWindowAssignmentGaps(currentConfig)
		if cleared > 0 {
			updateButtonConfig(gapFilledConfig)
			a.natsAdapter.PublishMessage(liveButtonsSubject, gapFilledConfig)
			log.Info("Gap-filling/compaction performed and update published (no processWindowUpdate).")
		} else {
			log.Info("Gap-filling triggered but no gaps were found.")
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
	log.Info("ButtonManagerAdapter running.")
	select {} // Block indefinitely
}
