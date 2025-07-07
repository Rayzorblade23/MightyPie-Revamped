package settingsManagerAdapter

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

type SettingsManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

var currentSettings map[string]SettingsEntry

func New(natsAdapter *natsAdapter.NatsAdapter) *SettingsManagerAdapter {
	a := &SettingsManagerAdapter{
		natsAdapter: natsAdapter,
	}

	subject := env.Get("PUBLIC_NATSSUBJECT_SETTINGS_UPDATE")

	settings, err := ReadSettings()
	if err != nil {
		log.Fatalf("FATAL: Failed to read settings.json: %v", err)
	}
	currentSettings = settings

	a.natsAdapter.PublishMessage(subject, settings)
	log.Println("INFO: Initial settings published.")

	natsAdapter.SubscribeJetStreamPull(subject, "", func(msg *nats.Msg) {
		var newSettings map[string]SettingsEntry
		if err := json.Unmarshal(msg.Data, &newSettings); err != nil {
			log.Printf("ERROR: Failed to unmarshal settings update: %v", err)
			return
		}
		// Only write if settings have changed
		if !settingsEqual(currentSettings, newSettings) {
			if err := WriteSettings(newSettings); err != nil {
				log.Printf("ERROR: Failed to write settings.json: %v", err)
				return
			}
			currentSettings = newSettings
			log.Println("INFO: settings.json updated from NATS message.")
		} else {
			log.Println("INFO: Received settings update, but no changes detected.")
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

// ReadSettings loads the settings.json as a map of string to SettingsEntry.
func ReadSettings() (map[string]SettingsEntry, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, os.ErrNotExist
	}
	settingsPath := filepath.Join(localAppData, "MightyPieRevamped", "settings.json")
	configDir := filepath.Dir(settingsPath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}
	var settings map[string]SettingsEntry
	if err := json.Unmarshal(data, &settings); err != nil {
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
			log.Printf("ERROR: Failed to persist configPath to settings.json: %v", err)
		}
	}

	return settings, nil
}

// WriteSettings saves the settings map to settings.json.
func WriteSettings(settings map[string]SettingsEntry) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return os.ErrNotExist
	}
	settingsPath := filepath.Join(localAppData, "MightyPieRevamped", "settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
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
	log.Println("INFO: SettingsManagerAdapter running.")
	select {} // Block indefinitely
}
