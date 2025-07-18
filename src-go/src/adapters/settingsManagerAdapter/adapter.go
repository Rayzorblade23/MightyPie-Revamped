package settingsManagerAdapter

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
	core "github.com/Rayzorblade23/MightyPie-Revamped/src/core"
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
		// Reject empty settings updates
		if len(newSettings) == 0 {
			log.Printf("ERROR: Rejected incoming settings update: settings map is empty!")
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
const (
	settingsDirPermission  = 0o755
	settingsFilePermission = 0o644
)

// copyDefaultSettingsIfNeeded copies the default settings.json if it does not exist.
func copyDefaultSettingsIfNeeded(settingsPath, configDir string) error {
	staticDir, err := core.GetStaticDir()
	if err != nil {
		log.Printf("ERROR: Failed to get static dir: %v", err)
		return err
	}
	defaultSettingsRel := env.Get("PUBLIC_DIR_DEFAULTSETTINGS")
	if defaultSettingsRel == "" {
		log.Printf("ERROR: PUBLIC_DIR_DEFAULTSETTINGS not set!")
		return err // No default to copy from
	}
	defaultSettingsPath := filepath.Join(staticDir, defaultSettingsRel)
	if err := os.MkdirAll(configDir, settingsDirPermission); err != nil {
		log.Printf("ERROR: Failed to create config dir %q: %v", configDir, err)
		return err
	}
	data, err := os.ReadFile(defaultSettingsPath)
	if err != nil {
		log.Printf("ERROR: Failed to read default settings from %q: %v", defaultSettingsPath, err)
		return err
	}
	if err := os.WriteFile(settingsPath, data, settingsFilePermission); err != nil {
		log.Printf("ERROR: Failed to write settings.json to %q: %v", settingsPath, err)
		return err
	}
	return nil
}

func ReadSettings() (map[string]SettingsEntry, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, os.ErrNotExist
	}
	settingsPath := filepath.Join(localAppData, "MightyPieRevamped", "settings.json")
	configDir := filepath.Dir(settingsPath)

	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		if err := copyDefaultSettingsIfNeeded(settingsPath, configDir); err != nil {
			return nil, err
		}
	}

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
