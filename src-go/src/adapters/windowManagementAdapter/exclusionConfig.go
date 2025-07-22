package windowManagementAdapter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// ExclusionConfig defines the rules for excluding windows from being assigned to buttons.
type ExclusionConfig struct {
	ExcludedTitles     []string            `json:"excluded_titles"`
	ExcludedApps       []string            `json:"excluded_apps"`
	SpecificExclusions []SpecificExclusion `json:"specific_exclusions"`
}

// SpecificExclusion defines a granular rule for excluding a window based on its app and title.
type SpecificExclusion struct {
	App   string `json:"app"`
	Title string `json:"title"`
}

func loadExclusionConfig() (*ExclusionConfig, error) {
	// Define user and default config paths
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	userConfigDir := filepath.Join(localAppData, "MightyPieRevamped")
	userConfigPath := filepath.Join(userConfigDir, "window_exclusion_list.json")

	// If user config doesn't exist, create it from the default
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		log.Printf("DEBUG: User config not found. Attempting to create from default.")
		staticDir, err := core.GetStaticDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get static dir for default exclusion list: %w", err)
		}
		defaultConfigPath := filepath.Join(staticDir, env.Get("PUBLIC_DIR_DEFAULTEXCLUSIONLIST"))

		if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("default exclusion config not found at %s", defaultConfigPath)
		}

		// Create user config directory if it doesn't exist
		if err := os.MkdirAll(userConfigDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create user config directory: %w", err)
		}

		// Copy default config to user directory
		sourceFile, err := os.Open(defaultConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open default config for copying: %w", err)
		}
		defer sourceFile.Close()

		destFile, err := os.Create(userConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create user config file: %w", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, sourceFile); err != nil {
			log.Printf("ERROR: Failed to copy default config: %v", err)
			return nil, fmt.Errorf("failed to copy default config to user directory: %w", err)
		}
	}

	// Load the config from the user path
	file, err := os.Open(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open exclusion config file at %s: %w", userConfigPath, err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read exclusion config file: %w", err)
	}

	var config ExclusionConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse exclusion config JSON: %w", err)
	}

	return &config, nil
}
