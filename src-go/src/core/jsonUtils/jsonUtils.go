package jsonUtils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// defaultDirPermissions are the default file permissions for directories (rwxr-xr-x).
	defaultDirPermissions = 0755
	// defaultFilePermissions are the default file permissions for files (rw-r--r--).
	defaultFilePermissions = 0644
	// jsonIndent is the string used for indenting nested JSON objects.
	jsonIndent = "  "
	// jsonPrefix is the prefix for each new line of indented JSON.
	jsonPrefix = ""
)

// ReadFromFile reads a JSON file into a given interface.
// It returns nil if the file does not exist or is empty, allowing the caller to handle initialization.
func ReadFromFile(filePath string, v any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, which is a valid case for creating a new one.
		}
		return err // Other read error.
	}

	if len(data) == 0 {
		return nil // File is empty, treat as uninitialized.
	}

	return json.Unmarshal(data, v)
}

// WriteToFile marshals an interface to an indented JSON string and writes it to a file.
// It automatically creates the destination directory if it does not exist.
func WriteToFile(filePath string, v any) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, defaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(v, jsonPrefix, jsonIndent)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, defaultFilePermissions)
}

// CopyFile reads a JSON file from srcPath and writes its contents to dstPath.
// This ensures the copied file is valid JSON.
func CopyFile(srcPath, dstPath string) error {
	var data any
	if err := ReadFromFile(srcPath, &data); err != nil {
		return err
	}
	return WriteToFile(dstPath, data)
}

// CreateFileFromDefaultIfNotExist copies a file from srcPath to dstPath, but only if dstPath does not already exist.
func CreateFileFromDefaultIfNotExist(srcPath, dstPath string) error {
	if _, err := os.Stat(dstPath); err == nil {
		// Destination file already exists, do nothing.
		return nil
	} else if !os.IsNotExist(err) {
		// Another error occurred with stat, return it.
		return fmt.Errorf("failed to check destination file %s: %w", dstPath, err)
	}

	// Destination file does not exist, so proceed with the copy.
	return CopyFile(srcPath, dstPath)
}

// Copy performs a deep copy from src to dst using JSON marshaling and unmarshaling.
func Copy(src, dst any) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}
