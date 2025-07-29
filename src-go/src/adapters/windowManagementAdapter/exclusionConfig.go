package windowManagementAdapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
)



// ExclusionConfig defines the rules for excluding windows from being assigned to buttons.
type ExclusionConfig struct {
	ExcludedTitles     []string            `json:"excluded_titles"`
	ExcludedApps       []string            `json:"excluded_apps"`
	ExcludedClassNames []string            `json:"excluded_class_names"`
	SpecificExclusions []SpecificExclusion `json:"specific_exclusions"`
}

// SpecificExclusion defines a granular rule for excluding a window based on its app and title.
type SpecificExclusion struct {
	App   string `json:"app"`
	Title string `json:"title"`
}

func getExclusionConfigPath() (string, error) {
	// Define user and default config paths
	userConfigDir, err := core.GetAppDataDir()
	if err != nil {
		return "", err
	}
	userConfigPath := filepath.Join(userConfigDir, os.Getenv("PUBLIC_DIR_EXCLUSIONLIST"))

	return userConfigPath, nil
}

func loadExclusionConfig() (*ExclusionConfig, error) {
	configPath, err := getExclusionConfigPath()
	if err != nil {
		return nil, err
	}
	var config ExclusionConfig

	// Ensure the exclusion config file exists by copying the default if needed.
	assetDir, err := core.GetAssetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset dir for default exclusion list: %w", err)
	}
	defaultConfigPath := filepath.Join(assetDir, os.Getenv("PUBLIC_DIR_DEFAULTEXCLUSIONLIST"))

	if err := jsonUtils.CreateFileFromDefaultIfNotExist(defaultConfigPath, configPath); err != nil {
		return nil, fmt.Errorf("failed to copy default exclusion config if needed: %w", err)
	}

	// Now, read the file (either the original or the newly created one).
	if err := jsonUtils.ReadFromFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to read exclusion config: %w", err)
	}

	return &config, nil
}
