package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"log"
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
	staticDir, err := core.GetStaticDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine static dir: %w", err)
	}
	jsonPath := filepath.Join(staticDir, "data", "buttonFunctions.json")
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
		log.Fatalf("Could not load button function metadata: %v", err)
	}
	for key := range handlers {
		if _, ok := metadataMap[key]; !ok {
			log.Fatalf("functionHandlers key '%s' is not present in buttonFunctions.json", key)
		}
	}
}

