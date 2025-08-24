package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
)

const (
	backupFilePrefix = "piemenuConfig_BACKUP"
)

// GetButtonConfig returns a deep copy of the current button configuration.
func GetButtonConfig() ConfigData {
	mu.RLock()
	configToCopy := buttonConfig
	sourceLen := len(configToCopy)
	mu.RUnlock()

	copiedConfig, err := deepCopyConfig(configToCopy)
	if err != nil {
		log.Error("GetButtonConfig - deepCopyConfig returned an error: %v. Returning empty config.", err)
		return make(ConfigData)
	}
	if copiedConfig == nil { // Should not happen with current deepCopyConfig logic
		log.Error("GetButtonConfig - deepCopyConfig returned nil unexpectedly. Returning empty config.")
		return make(ConfigData)
	}
	if len(copiedConfig) == 0 && sourceLen > 0 {
		log.Warn("GetButtonConfig - deepCopyConfig resulted in an EMPTY map, but source was NOT empty (len %d)! Decode likely failed inside deepCopyConfig.", sourceLen)
		return make(ConfigData)
	}

	return copiedConfig
}

// WriteButtonConfig saves the given configuration to the default config file path.
func WriteButtonConfig(config ConfigData) error {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(appDataDir, os.Getenv("PUBLIC_DIR_PIEMENUCONFIG"))

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
		log.Error("deepCopyConfig - failed to copy config: %v", err)
		return nil, err
	}
	return dst, nil
}

// BackupConfigToFile writes the given config to a backup file.
func BackupConfigToFile(config ConfigData) error {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return err
	}
	// Determine backups directory from environment (no hardcoded fallback)
	backupsRel := os.Getenv("PUBLIC_DIR_CONFIGBACKUPS")
	backupDir := filepath.Join(appDataDir, backupsRel)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory '%s': %w", backupDir, err)
	}
	return BackupConfigToFileWithBaseDir(config, backupDir)
}

// BackupConfigToFileWithBaseDir writes the config to a backup file in a specific directory.
func BackupConfigToFileWithBaseDir(config ConfigData, baseDir string) error {
	// Ensure the base directory exists (robust if a custom path is provided)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory '%s': %w", baseDir, err)
	}

	baseName := backupFilePrefix + ".json"
	backupPath := filepath.Join(baseDir, baseName)
	idx := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		idx++
		backupPath = filepath.Join(baseDir, fmt.Sprintf("%s_%d%s", backupFilePrefix, idx, ".json"))
	}

	return jsonUtils.WriteToFile(backupPath, config)
}

// ReadButtonConfig loads the button configuration from the default path.
func ReadButtonConfig() (ConfigData, error) {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(appDataDir, os.Getenv("PUBLIC_DIR_PIEMENUCONFIG"))

	var config ConfigData
	if err := jsonUtils.ReadFromFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	if config == nil {
		log.Warn("Config file not found or is empty, creating default config at '%s'", configPath)
		defaultConfig := NewDefaultConfig()
		if err := WriteButtonConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		return defaultConfig, nil
	}

	// Validate and repair the configuration structure
	configChanged := validateAndRepairConfig(config)
	
	// If the config was changed during validation, write it back to disk
	if configChanged {
		log.Info("Button configuration was updated during validation, writing changes to file")
		if err := WriteButtonConfig(config); err != nil {
			log.Error("Failed to write validated button config: %v", err)
			// Continue with the validated config in memory even if write fails
		}
	}

	return config, nil
}

// LoadConfigFromFile loads a ConfigData from a specific file path.
func LoadConfigFromFile(path string) (ConfigData, error) {
    // Read raw bytes to probe structure without cross-adapter imports
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file '%s': %w", path, err)
    }

    // Probe for unified full-config by checking presence of top-level fields
    var probe struct {
        Buttons   json.RawMessage `json:"buttons"`
        Shortcuts any             `json:"shortcuts"`
        Starred   any             `json:"starred"`
    }
    if err := json.Unmarshal(raw, &probe); err == nil && (probe.Buttons != nil || probe.Shortcuts != nil || probe.Starred != nil) {
        // Extract only buttons into our local ConfigData type
        var buttons ConfigData
        if probe.Buttons == nil {
            return make(ConfigData), nil
        }
        if err := json.Unmarshal(probe.Buttons, &buttons); err == nil {
            return buttons, nil
        }
        // If buttons failed to unmarshal, fall through to legacy attempt below
    }

    // Fallback: legacy buttons-only ConfigData
    var config ConfigData
    if err := json.Unmarshal(raw, &config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config from '%s': %w", path, err)
    }
    return config, nil
}

// validateAndRepairConfig checks that each page has exactly 8 buttons (indexes 0-7)
// and validates button types and properties. It returns true if any changes were made.
func validateAndRepairConfig(config ConfigData) bool {
	if config == nil {
		return false
	}

	configChanged := false
	
	// Iterate through all menus in the config
	for menuID, menuConfig := range config {
		// Iterate through all pages in this menu
		for pageID, pageConfig := range menuConfig {
			// Ensure each page has buttons 0-7
			for buttonIdx := range 8 {
				buttonID := fmt.Sprintf("%d", buttonIdx)

				// Check if this button exists in the page
				button, exists := pageConfig[buttonID]
				if !exists {
					// Button doesn't exist, create a default ShowAnyWindow button
					log.Warn("Missing button '%s' in page '%s' of menu '%s', adding default ShowAnyWindow button", 
						buttonID, pageID, menuID)
					
					// Create a default ShowAnyWindow button
					pageConfig[buttonID] = Button{
						ButtonType: "show_any_window",
						Properties: mustMarshalProperties(core.ShowAnyWindowProperties{
							ButtonTextUpper: "",
							ButtonTextLower: "",
							IconPath:        "",
							WindowHandle:    InvalidHandle,
						}),
					}
					configChanged = true
					continue
				}

				// Validate button type
				validButtonType := validateButtonType(button.ButtonType)
				if !validButtonType {
					log.Warn("Invalid button type '%s' for button '%s' in page '%s' of menu '%s', resetting to default ShowAnyWindow button", 
						button.ButtonType, buttonID, pageID, menuID)
					
					// Reset to a default ShowAnyWindow button
					pageConfig[buttonID] = Button{
						ButtonType: "show_any_window",
						Properties: mustMarshalProperties(core.ShowAnyWindowProperties{
							ButtonTextUpper: "",
							ButtonTextLower: "",
							IconPath:        "",
							WindowHandle:    InvalidHandle,
						}),
					}
					configChanged = true
					continue
				}
				
				// Validate button properties based on its type
				if !validateButtonProperties(button) {
					log.Warn("Invalid properties for button '%s' in page '%s' of menu '%s' with type '%s', resetting to default ShowAnyWindow button", 
						buttonID, pageID, menuID, button.ButtonType)
					
					// Reset to a default ShowAnyWindow button
					pageConfig[buttonID] = Button{
						ButtonType: "show_any_window",
						Properties: mustMarshalProperties(core.ShowAnyWindowProperties{
							ButtonTextUpper: "",
							ButtonTextLower: "",
							IconPath:        "",
							WindowHandle:    InvalidHandle,
						}),
					}
					configChanged = true
				}
			}
		}
	}

	return configChanged
}

// validateButtonType checks if the button type is one of the valid types
func validateButtonType(buttonType string) bool {
	validTypes := []string{
		"show_program_window",
		"show_any_window",
		"call_function",
		"launch_program",
		"open_page_in_menu",
		"open_resource",
		"disabled",
	}

	return slices.Contains(validTypes, buttonType)
}

// validateButtonProperties checks if the button properties match the expected structure for its type
func validateButtonProperties(button Button) bool {
	// For each button type, try to unmarshal the properties into the expected struct
	// If it fails, the properties don't match the expected structure
	switch button.ButtonType {
	case "show_any_window":
		var props core.ShowAnyWindowProperties
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal ShowAnyWindowProperties: %v", err)
			return false
		}
		return true

	case "show_program_window":
		var props core.ShowProgramWindowProperties
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal ShowProgramWindowProperties: %v", err)
			return false
		}
		return true

	case "call_function":
		var props core.CallFunctionProperties
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal CallFunctionProperties: %v", err)
			return false
		}
		return true

	case "launch_program":
		var props core.LaunchProgramProperties
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal LaunchProgramProperties: %v", err)
			return false
		}
		return true

	case "open_page_in_menu":
		var props core.OpenSpecificPieMenuPage
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal OpenSpecificPieMenuPage: %v", err)
			return false
		}
		return true

	case "open_resource":
		var props core.OpenResourceProperties
		if err := json.Unmarshal(button.Properties, &props); err != nil {
			log.Warn("Failed to unmarshal OpenResourceProperties: %v", err)
			return false
		}
		return true

	case "disabled":
		// Disabled buttons don't need specific properties
		return true

	default:
		// Unknown button type
		return false
	}
}
