package windowManagementAdapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
)

const (
	jsonExtension         = ".json"
	exclusionListFileName = "windowExclusionList"
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

func getExclusionConfigPath() (string, error) {
	// Define user and default config paths
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	userConfigDir := filepath.Join(localAppData, os.Getenv("PUBLIC_APPNAME"))
	userConfigPath := filepath.Join(userConfigDir, exclusionListFileName+jsonExtension)

	return userConfigPath, nil
}

func loadExclusionConfig() (*ExclusionConfig, error) {
	configPath, err := getExclusionConfigPath()
	if err != nil {
		return nil, err
	}
	var config ExclusionConfig

	// Ensure the exclusion config file exists by copying the default if needed.
	staticDir, err := core.GetStaticDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get static dir for default exclusion list: %w", err)
	}
	defaultConfigPath := filepath.Join(staticDir, os.Getenv("PUBLIC_DIR_DEFAULTEXCLUSIONLIST"))

	if err := jsonUtils.CreateFileFromDefaultIfNotExist(defaultConfigPath, configPath); err != nil {
		return nil, fmt.Errorf("failed to copy default exclusion config if needed: %w", err)
	}

	// Now, read the file (either the original or the newly created one).
	if err := jsonUtils.ReadFromFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to read exclusion config: %w", err)
	}

	return &config, nil
}
