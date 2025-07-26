package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
)

const (
	jsonExtension    = ".json"
	configFileName   = "buttonConfig"
	backupFilePrefix = "buttonConfig_BACKUP"
)

// GetButtonConfig returns a deep copy of the current button configuration.
func GetButtonConfig() ConfigData {
	mu.RLock()
	configToCopy := buttonConfig
	sourceLen := len(configToCopy)
	mu.RUnlock()

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
		return make(ConfigData)
	}

	return copiedConfig
}

// WriteButtonConfig saves the given configuration to the default config file path.
func WriteButtonConfig(config ConfigData) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, os.Getenv("PUBLIC_APPNAME"), configFileName+jsonExtension)

	return jsonUtils.WriteToFile(configPath, config)
}

// NewDefaultConfig creates a new default button configuration.
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
					ButtonType: string(core.ButtonTypeShowAnyWindow),
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

// mustMarshalProperties marshals properties or panics.
func mustMarshalProperties(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default button properties: %v", err))
	}
	return data
}

// deepCopyConfig performs a deep copy of the configuration.
func deepCopyConfig(src ConfigData) (ConfigData, error) {
	if src == nil {
		return make(ConfigData), nil
	}

	dst := make(ConfigData)
	if err := jsonUtils.Copy(src, &dst); err != nil {
		log.Printf("ERROR: deepCopyConfig - failed to copy config: %v", err)
		return nil, err
	}
	return dst, nil
}

// BackupConfigToFile writes the given config to a backup file.
func BackupConfigToFile(config ConfigData) error {
	return BackupConfigToFileWithBaseDir(config, filepath.Join(os.Getenv("LOCALAPPDATA"), os.Getenv("PUBLIC_APPNAME")))
}

// BackupConfigToFileWithBaseDir writes the config to a backup file in a specific directory.
func BackupConfigToFileWithBaseDir(config ConfigData, baseDir string) error {
	baseName := backupFilePrefix + jsonExtension
	backupPath := filepath.Join(baseDir, baseName)
	idx := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		idx++
		backupPath = filepath.Join(baseDir, fmt.Sprintf("%s_%d%s", backupFilePrefix, idx, jsonExtension))
	}

	return jsonUtils.WriteToFile(backupPath, config)
}

// ReadButtonConfig loads the button configuration from the default path.
func ReadButtonConfig() (ConfigData, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, os.Getenv("PUBLIC_APPNAME"), configFileName+jsonExtension)

	var config ConfigData
	if err := jsonUtils.ReadFromFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	if config == nil {
		log.Printf("WARN: Config file not found or is empty, creating default config at '%s'", configPath)
		defaultConfig := NewDefaultConfig()
		if err := WriteButtonConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		return defaultConfig, nil
	}

	return config, nil
}

// LoadConfigFromFile loads a ConfigData from a specific file path.
func LoadConfigFromFile(path string) (ConfigData, error) {
	var config ConfigData
	if err := jsonUtils.ReadFromFile(path, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from '%s': %w", path, err)
	}
	return config, nil
}
