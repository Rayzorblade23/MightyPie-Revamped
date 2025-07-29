package settingsManagerAdapter

import (
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

	a.natsAdapter.PublishMessage(subject, "SettingsManager", settings)
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
	configDir := filepath.Dir(settingsPath)

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

	var settings map[string]SettingsEntry
	if err := jsonUtils.ReadFromFile(settingsPath, &settings); err != nil {
		return nil, err
	}

	// Ensure "configPath" entry exists in the returned map
	configPathAdded := false
	if _, ok := settings["configPath"]; !ok {
		settings["configPath"] = SettingsEntry{
			Label:        "Config Path",
			IsExposed:    false,
			Type:         "string",
			Value:        configDir,
			DefaultValue: configDir,
		}
		configPathAdded = true
	}

	if configPathAdded {
		if err := WriteSettings(settings); err != nil {
			log.Error("Failed to persist configPath to settings.json: %v", err)
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
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		avBytes, _ := json.Marshal(v)
		bvBytes, _ := json.Marshal(bv)
		if string(avBytes) != string(bvBytes) {
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
