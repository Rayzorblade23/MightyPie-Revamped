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
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Dir(dir), nil
		}
		if filepath.Base(dir) == "src-go" {
			return filepath.Dir(filepath.Dir(dir)), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root directory from %s", wd)
		}
		dir = parent
	}
}

// Returns the absolute path to the 'static' directory within the project.
func GetStaticDir() (string, error) {
	projectRoot, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project root to determine static dir: %w", err)
	}
	staticDirPath := filepath.Join(projectRoot, "static")

	// It's a good idea to verify that the static directory actually exists
	// This helps catch configuration errors early.
	stat, err := os.Stat(staticDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("static directory not found at expected path '%s' (derived from project root '%s')", staticDirPath, projectRoot)
		}
		return "", fmt.Errorf("error accessing static directory at '%s': %w", staticDirPath, err)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("expected '%s' to be a directory, but it's not", staticDirPath)
	}

	return staticDirPath, nil
}