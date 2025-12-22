package settingsManagerAdapter

import (
	"slices"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	core "github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("SettingsManager")



type SettingsManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

var currentSettings map[string]SettingsEntry

func New(natsAdapter *natsAdapter.NatsAdapter) *SettingsManagerAdapter {
	a := &SettingsManagerAdapter{
		natsAdapter: natsAdapter,
	}

	subject := os.Getenv("PUBLIC_NATSSUBJECT_SETTINGS_UPDATE")

	settings, err := ReadSettings()
	if err != nil {
		log.Fatal("Failed to read settings.json: %v", err)
	}
	currentSettings = settings

	a.natsAdapter.PublishMessage(subject, settings)
	log.Info("Initial settings published.")

	natsAdapter.SubscribeJetStreamPull(subject, "", func(msg *nats.Msg) {
		var newSettings map[string]SettingsEntry
		if err := json.Unmarshal(msg.Data, &newSettings); err != nil {
			log.Error("Failed to unmarshal settings update: %v", err)
			return
		}
		// Reject empty settings updates
		if len(newSettings) == 0 {
			log.Error("Rejected incoming settings update: settings map is empty!")
			return
		}
		// Only write if settings have changed
		if !settingsEqual(currentSettings, newSettings) {
			if err := WriteSettings(newSettings); err != nil {
				log.Error("Failed to write settings.json: %v", err)
				return
			}
			currentSettings = newSettings
			log.Info("settings.json updated from NATS message.")
		} else {
			log.Info("Received settings update, but no changes detected.")
		}
	})

	return a
}

// SettingsEntry represents a single settings entry with type info, value, and metadata.
type SettingsEntry struct {
	Index        int      `json:"index"`
	Category     string   `json:"category,omitempty"` // For grouping settings in UI
	Label        string   `json:"label"`
	IsExposed    bool     `json:"isExposed"`
	Type         string   `json:"type"` // "int", "float", "string", "bool", "enum", "color"
	Value        any      `json:"value"`
	DefaultValue any      `json:"defaultValue"`
	Options      []string `json:"options,omitempty"` // Only for enum type
}

func ReadSettings() (map[string]SettingsEntry, error) {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return nil, err
	}
	settingsPath := filepath.Join(appDataDir, os.Getenv("PUBLIC_DIR_SETTINGS"))

	// Ensure the settings file exists by copying the default if needed.
	assetDir, err := core.GetAssetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset dir for default settings: %w", err)
	}
	defaultSettingsRel := os.Getenv("PUBLIC_DIR_DEFAULTSETTINGS")
	if defaultSettingsRel == "" {
		return nil, fmt.Errorf("environment variable PUBLIC_DIR_DEFAULTSETTINGS is not set")
	}
	defaultSettingsPath := filepath.Join(assetDir, defaultSettingsRel)

	if err := jsonUtils.CreateFileFromDefaultIfNotExist(defaultSettingsPath, settingsPath); err != nil {
		return nil, fmt.Errorf("failed to copy default settings if needed: %w", err)
	}

	// Load default settings for validation
	var defaultSettings map[string]SettingsEntry
	if err := jsonUtils.ReadFromFile(defaultSettingsPath, &defaultSettings); err != nil {
		return nil, fmt.Errorf("failed to read default settings for validation: %w", err)
	}

	// Load user settings
	var settings map[string]SettingsEntry
	if err := jsonUtils.ReadFromFile(settingsPath, &settings); err != nil {
		return nil, err
	}

	// Initialize settings map if it's nil
	if settings == nil {
		settings = make(map[string]SettingsEntry)
	}

	// Validate that all default settings exist in the user settings
	settingsChanged := false
	for key, defaultEntry := range defaultSettings {
		// Check if the key exists in user settings
		userEntry, exists := settings[key]
		if !exists {
			// Key doesn't exist, add it with default values
			log.Warn("Missing setting '%s' in settings.json, adding with default value", key)
			settings[key] = defaultEntry
			settingsChanged = true
			continue
		}

		// Migrate category field if missing
		if userEntry.Category == "" && defaultEntry.Category != "" {
			log.Info("Migrating setting '%s' to add category '%s'", key, defaultEntry.Category)
			userEntry.Category = defaultEntry.Category
			settings[key] = userEntry
			settingsChanged = true
		}

		// Validate the entry has the correct structure
		if userEntry.Type != defaultEntry.Type {
			log.Warn("Setting '%s' has incorrect type '%s', expected '%s', resetting to default", 
				key, userEntry.Type, defaultEntry.Type)
			settings[key] = defaultEntry
			settingsChanged = true
			continue
		}

		// For enum types, validate that the value is one of the options
		if userEntry.Type == "enum" && len(defaultEntry.Options) > 0 {
			valueStr, ok := userEntry.Value.(string)
			if !ok {
				log.Warn("Setting '%s' has non-string value for enum type, resetting to default", key)
				settings[key] = defaultEntry
				settingsChanged = true
				continue
			}

			validOption := slices.Contains(defaultEntry.Options, valueStr)

			if !validOption {
				log.Warn("Setting '%s' has invalid enum value '%s', resetting to default", 
					key, valueStr)
				settings[key] = defaultEntry
				settingsChanged = true
			}
		}
	}

	// If settings were changed during validation, write them back to the file
	if settingsChanged {
		log.Info("Settings were updated during validation, writing changes to file")
		if err := WriteSettings(settings); err != nil {
			log.Error("Failed to write validated settings: %v", err)
			// Continue with the validated settings in memory even if write fails
		}
	}

	return settings, nil
}

// WriteSettings saves the settings map to settings.json.
func WriteSettings(settings map[string]SettingsEntry) error {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(appDataDir, os.Getenv("PUBLIC_DIR_SETTINGS"))
	return jsonUtils.WriteToFile(settingsPath, settings)
}

// settingsEqual compares two settings maps for equality.
func settingsEqual(a, b map[string]SettingsEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for key, aEntry := range a {
		bEntry, ok := b[key]
		if !ok {
			return false
		}
		aBytes, _ := json.Marshal(aEntry)
		bBytes, _ := json.Marshal(bEntry)
		if string(aBytes) != string(bBytes) {
			return false
		}
	}
	return true
}

// Run keeps the adapter alive (if needed, e.g., for non-NATS goroutines)
func (a *SettingsManagerAdapter) Run() error {
	log.Info("SettingsManagerAdapter running.")
	select {} // Block indefinitely
}
