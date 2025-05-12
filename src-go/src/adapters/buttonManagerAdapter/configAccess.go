package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// ReadButtonConfig (No DEBUG logs originally)
func ReadButtonConfig() (ConfigData, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	var config ConfigData
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", configPath, err)
	}
	return config, nil
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
		log.Printf("ERROR: deepCopyConfig - FAILED TO DECODE JSON into dst: %v", err) // Keep ERROR
		log.Println("WARN: deepCopyConfig - Returning NEW EMPTY MAP due to decode failure.") // Keep WARN
		return make(ConfigData), nil // Return EMPTY MAP on decode error
	}

	// log.Printf("DEBUG: deepCopyConfig successful. Decoded map length: %d", len(dst)) // Removed DEBUG
	return dst, nil
}