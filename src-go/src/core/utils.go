package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// GetTypeName returns the name of the type of the given value.
func GetTypeName(i any) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Returns the project root directory.
func GetRootDir() (string, error) {
	projectRoot := os.Getenv("MIGHTYPIE_ROOT_DIR")
	if projectRoot == "" {
		return "", fmt.Errorf("MIGHTYPIE_ROOT_DIR environment variable not set")
	}
	return projectRoot, nil
}

// GetAssetDir returns the absolute path to the 'assets' directory, typically for build assets.
func GetAssetDir() (string, error) {
	projectRoot, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project root to determine asset dir: %w", err)
	}

	assetDirRel := os.Getenv("PUBLIC_DIR_ASSETS")
	if assetDirRel == "" {
		return "", fmt.Errorf("PUBLIC_DIR_ASSETS environment variable is not set")
	}
	assetDirPath := filepath.Join(projectRoot, assetDirRel)

	// Verify that the asset directory actually exists
	stat, err := os.Stat(assetDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("asset directory not found at expected path '%s'", assetDirPath)
		}
		return "", fmt.Errorf("error accessing asset directory at '%s': %w", assetDirPath, err)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("expected '%s' to be a directory, but it's not", assetDirPath)
	}

	return assetDirPath, nil
}

// getAppDataDir returns the appropriate AppData directory for MightyPie based on OS
func GetAppDataDir() (string, error) {
	var appDataDir string

	// Get the app name from environment variable, or use default if not set
	appName := os.Getenv("PUBLIC_APPNAME")
	if appName == "" {
		appName = "MightyPieRevamped" // Default fallback
	}

	// On Windows, use %LOCALAPPDATA%\AppName
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	appDataDir = filepath.Join(appData, appName)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("could not create AppData directory: %w", err)
	}

	return appDataDir, nil
}