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

	buttonUpdateSubject := os.Getenv("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE")
	windowUpdateSubject := os.Getenv("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	baseConfigSubject := os.Getenv("PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG")
	saveConfigBackupSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP")
	loadConfigBackupSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP")
	receiveNewBaseConfigSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_UPDATE")
	fillGapsSubject := os.Getenv("PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS")

	config, err := ReadButtonConfig()
	if err != nil {
		log.Fatal("FATAL: Failed to read initial button configuration: %v", err)
	}

	updateButtonConfig(config)
	log.Info("INFO: Initial button configuration loaded.")
	// PrintConfig(config, true)

	a.natsAdapter.PublishMessage(baseConfigSubject, config)

	a.natsAdapter.SubscribeToSubject(receiveNewBaseConfigSubject, func(msg *nats.Msg) {
		log.Info("Raw config coming in on '%s'.", msg.Subject)

		var newConfig ConfigData
		if err := json.Unmarshal(msg.Data, &newConfig); err != nil {
			log.Error("Failed to unmarshal config: %v", err)
			return
		}

		// Reject empty config updates
		if len(newConfig) == 0 {
			log.Error("Rejected incoming config update: config is empty!")
			return
		}

		if err := WriteButtonConfig(newConfig); err != nil {
			log.Error("Failed to write config: %v", err)
			return
		}

		// Read it back in to update in-memory state
		loadedConfig, err := ReadButtonConfig()
		if err != nil {
			log.Error("Failed to reload config after write: %v", err)
			return
		}

		updateButtonConfig(loadedConfig)

		log.Info("INFO: Config written and reloaded from disk.")
		// PrintConfig(buttonConfig, false)

		a.natsAdapter.PublishMessage(windowUpdateSubject, windowsList)
	})

	a.natsAdapter.SubscribeToSubject(saveConfigBackupSubject, func(msg *nats.Msg) {
		// Assume BackupConfigToFile exists and takes the config as argument
		var configToBackup ConfigData
		if err := json.Unmarshal(msg.Data, &configToBackup); err != nil {
			log.Error("Failed to unmarshal backup config: %v", err)
			return
		}
		if err := BackupConfigToFile(configToBackup); err != nil {
			log.Error("Failed to backup config to file: %v", err)
			return
		}
		log.Info("Config backup successful.")
	})

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
			// Publish the updated configuration object
			a.natsAdapter.PublishMessage(buttonUpdateSubject, processedConfig)
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
			a.natsAdapter.PublishMessage(os.Getenv("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE"), gapFilledConfig)
			log.Info("Gap-filling/compaction performed and update published (no processWindowUpdate).")
		} else {
			log.Info("Gap-filling triggered but no gaps were found.")
		}
	})

	a.natsAdapter.SubscribeToSubject(loadConfigBackupSubject, func(msg *nats.Msg) {

		// msg.Data contains the path to the backup file as a string (may include quotes)
		backupPath := string(msg.Data)
		// Remove any leading/trailing quotes (single or double)
		if len(backupPath) > 0 && (backupPath[0] == '"' || backupPath[0] == '\'') {
			backupPath = backupPath[1:]
		}
		if len(backupPath) > 0 && (backupPath[len(backupPath)-1] == '"' || backupPath[len(backupPath)-1] == '\'') {
			backupPath = backupPath[:len(backupPath)-1]
		}
		log.Info("Loading config backup from")
		log.Info("↳ '%s'", backupPath)

		// Load config from the backup file
		backupConfig, err := LoadConfigFromFile(backupPath)
		if err != nil {
			log.Error("Failed to load config from backup: %v", err)
			return
		}

		// Overwrite the current config file
		if err := WriteButtonConfig(backupConfig); err != nil {
			log.Error("Failed to overwrite buttonConfig.json: %v", err)
			return
		}

		// Update in-memory config
		updateButtonConfig(backupConfig)
		log.Info("↳ Config loaded from backup and set as current.")

		// Publish the updated config
		a.natsAdapter.PublishMessage(baseConfigSubject, backupConfig)
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
