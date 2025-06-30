package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// GetButtonConfig (Cleaned)
func GetButtonConfig() ConfigData {
	mu.RLock()
	configToCopy := buttonConfig
	sourceLen := len(configToCopy)
	mu.RUnlock()

	// log.Printf("DEBUG: GetButtonConfig - Source length before copy: %d", sourceLen) // Removed DEBUG
	// log.Println("DEBUG: GetButtonConfig - Entering deepCopyConfig...") // Removed DEBUG

	copiedConfig, err := deepCopyConfig(configToCopy)
	if err != nil {
		log.Printf("ERROR: GetButtonConfig - deepCopyConfig returned an error: %v. Returning empty config.", err)
		return make(ConfigData)
	}
	if copiedConfig == nil { // Should not happen with current deepCopyConfig logic
		log.Printf("ERROR: GetButtonConfig - deepCopyConfig returned nil unexpectedly. Returning empty config.")
		return make(ConfigData)
	}
	if len(copiedConfig) == 0 && sourceLen > 0 {
		log.Printf("WARN: GetButtonConfig - deepCopyConfig resulted in an EMPTY map, but source was NOT empty (len %d)! Decode likely failed inside deepCopyConfig.", sourceLen)
		// Return the potentially problematic empty map as per deepCopyConfig's logic
		return make(ConfigData)
	}

	// log.Printf("DEBUG: GetButtonConfig - Deep copy finished. Copied config length: %d", len(copiedConfig)) // Removed DEBUG
	return copiedConfig
}

func ReadButtonConfig() (ConfigData, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("WARN: Config file not found, creating default config at '%s'", configPath)
			defaultConfig := NewDefaultConfig()
			if err := WriteButtonConfig(defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	var config ConfigData
	if len(data) == 0 || json.Unmarshal(data, &config) != nil || len(config) == 0 {
		log.Printf("WARN: Config file is empty, invalid, or contains an empty config. Creating default config at '%s'", configPath)
		defaultConfig := NewDefaultConfig()
		if err := WriteButtonConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		return defaultConfig, nil
	}

	return config, nil
}

func WriteButtonConfig(config ConfigData) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file '%s': %w", configPath, err)
	}
	return nil
}

func NewDefaultConfig() ConfigData {
	const (
		numMenus   = 2
		numPages   = 3
		numButtons = 8
	)
	config := make(ConfigData)
	for menuIdx := range numMenus {
		menuID := fmt.Sprintf("%d", menuIdx)
		menuConfig := make(MenuConfig)
		for pageIdx := range numPages {
			pageID := fmt.Sprintf("%d", pageIdx)
			pageConfig := make(PageConfig)
			for btnIdx := range numButtons {
				btnID := fmt.Sprintf("%d", btnIdx)
				button := Button{
					ButtonType: string(ButtonTypeShowAnyWindow),
					Properties: mustMarshalProperties(core.ShowAnyWindowProperties{
						ButtonTextUpper: "",
						ButtonTextLower: "",
						IconPath:        "",
						WindowHandle:    InvalidHandle,
					}),
				}
				pageConfig[btnID] = button
			}
			menuConfig[pageID] = pageConfig
		}
		config[menuID] = menuConfig
	}
	return config
}

// mustMarshalProperties marshals properties or panics (for use in default config creation).
func mustMarshalProperties(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default button properties: %v", err))
	}
	return data
}

// deepCopyConfig (Cleaned)
func deepCopyConfig(src ConfigData) (ConfigData, error) {
	// log.Println("DEBUG: Entering deepCopyConfig...") // Removed DEBUG
	if src == nil {
		// log.Println("DEBUG: deepCopyConfig source is nil, returning new empty map.") // Removed DEBUG
		return make(ConfigData), nil
	}
	// log.Printf("DEBUG: deepCopyConfig source map length: %d", len(src)) // Removed DEBUG

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// enc.SetIndent("", "  ") // Indent not needed for production copy

	if err := enc.Encode(src); err != nil {
		log.Printf("ERROR: deepCopyConfig - FAILED TO ENCODE source: %v", err) // Keep ERROR
		return nil, fmt.Errorf("failed to encode config for deep copy: %w", err)
	}

	// encodedJSON := buf.String() // No need to log encoded JSON in prod
	// log.Printf("DEBUG: deepCopyConfig - Encoded JSON (first 300 bytes):\n---\n%s\n---", limitString(encodedJSON, 300)) // Removed DEBUG

	dec := json.NewDecoder(&buf)
	var dst ConfigData
	if err := dec.Decode(&dst); err != nil {
		log.Printf("ERROR: deepCopyConfig - FAILED TO DECODE JSON into dst: %v", err)        // Keep ERROR
		log.Println("WARN: deepCopyConfig - Returning NEW EMPTY MAP due to decode failure.") // Keep WARN
		return make(ConfigData), nil                                                         // Return EMPTY MAP on decode error
	}

	// log.Printf("DEBUG: deepCopyConfig successful. Decoded map length: %d", len(dst)) // Removed DEBUG
	return dst, nil
}

// BackupConfigToFile writes the given config to a backup file with an incrementing suffix if needed.
func BackupConfigToFile(config ConfigData) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	baseDir := filepath.Join(localAppData, "MightyPieRevamped")
	baseName := "buttonConfig_BACKUP.json"
	backupPath := filepath.Join(baseDir, baseName)

	// Find a non-existing backup filename
	idx := 0
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		idx++
		backupPath = filepath.Join(baseDir, fmt.Sprintf("buttonConfig_BACKUP_%d.json", idx))
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup config file '%s': %w", backupPath, err)
	}
	return nil
}

// LoadConfigFromFile loads a ConfigData from the specified file path.
func LoadConfigFromFile(path string) (ConfigData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}
	var config ConfigData
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from '%s': %w", path, err)
	}
	return config, nil
}
