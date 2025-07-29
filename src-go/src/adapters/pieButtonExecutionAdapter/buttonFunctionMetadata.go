package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

type ButtonFunctionMetadata struct {
	IconPath    string `json:"icon_path"`
	Description string `json:"description"`
}

// Loads buttonFunctions.json and returns a map of displayName to metadata.
func loadButtonFunctionMetadata() (map[string]ButtonFunctionMetadata, error) {
	assetDir, err := core.GetAssetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine asset dir: %w", err)
	}
	buttonFunctionsPath := os.Getenv("PUBLIC_DIR_BUTTONFUNCTIONS")
	if buttonFunctionsPath == "" {
		return nil, fmt.Errorf("PUBLIC_DIR_BUTTONFUNCTIONS environment variable not set")
	}
	jsonPath := filepath.Join(assetDir, buttonFunctionsPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read buttonFunctions.json: %w", err)
	}
	var raw map[string]ButtonFunctionMetadata
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse buttonFunctions.json: %w", err)
	}
	return raw, nil
}

// Validates that all handler keys are present in buttonFunctions.json.
func ValidateFunctionHandlers(handlers map[string]ButtonFunctionExecutor) {
	metadataMap, err := loadButtonFunctionMetadata()
	if err != nil {
		log.Fatal("Could not load button function metadata: %v", err)
	}
	for key := range handlers {
		if _, ok := metadataMap[key]; !ok {
			log.Warn("Warning: Handler key '%s' from code is not present in the button functions metadata file. Please add it.", key)
		}
	}
}
