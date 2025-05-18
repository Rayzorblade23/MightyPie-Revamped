package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// Struct to match the JSON structure
type ButtonFunctionMetadata struct {
	FunctionName string `json:"function_name"`
	IconPath     string `json:"icon_path"`
	Description  string `json:"description"`
}

// Holds all loaded function metadata
var buttonFunctionMetadataMap map[string]ButtonFunctionMetadata

// Function name variables
var (
	maximizeFunctionName string
	minimizeFunctionName string
	closeFunctionName    string
)

func loadButtonFunctionMetadata() error {
	staticDir, err := core.GetStaticDir()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}
	relPath := env.Get("PUBLIC_DIR_BUTTONFUNCTIONS")
	if relPath == "" {
		return fmt.Errorf("PUBLIC_DIR_BUTTONFUNCTIONS not set in environment")
	}
	jsonPath := filepath.Join(staticDir, relPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read buttonFunctions.json: %w", err)
	}
	var raw map[string]ButtonFunctionMetadata
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse buttonFunctions.json: %w", err)
	}
	buttonFunctionMetadataMap = raw

	// Set function name variables for use elsewhere
	if meta, ok := buttonFunctionMetadataMap["Maximize"]; ok {
		maximizeFunctionName = meta.FunctionName
	}
	if meta, ok := buttonFunctionMetadataMap["Minimize"]; ok {
		minimizeFunctionName = meta.FunctionName
	}
	if meta, ok := buttonFunctionMetadataMap["Close"]; ok {
		closeFunctionName = meta.FunctionName
	}
	return nil
}

// --- Types ---

// Defines the signature for basic functions without coordinates.
type ButtonFunctionNoArgHandler func() error

// Defines the signature for functions needing coordinates.
type ButtonFunctionWithCoordinatesHandler func(x, y int) error

// Provides a common interface for executing different function types.
type ButtonFunctionExecutor interface {
	Execute(x, y int) error
}

// Wraps a FunctionHandler to satisfy the HandlerWrapper interface.
type NoArgButtonFunctionExecutor struct {
	fn ButtonFunctionNoArgHandler
}

// Wraps a FunctionHandlerWithCoords to satisfy the HandlerWrapper interface.
type CoordinatesButtonFunctionExecutor struct {
	fn ButtonFunctionWithCoordinatesHandler
}

// --- Function Handlers & Wrappers ---

// Execute calls the wrapped basic function, ignoring coordinates.
func (h NoArgButtonFunctionExecutor) Execute(x, y int) error {
	return h.fn()
}

// Execute calls the wrapped function, passing coordinates.
func (h CoordinatesButtonFunctionExecutor) Execute(x, y int) error {
	return h.fn(x, y)
}

// Registers known functions that can be called.
func (a *PieButtonExecutionAdapter) registerBuiltInButtonFunctionExecutors() map[string]ButtonFunctionExecutor {
	handlers := make(map[string]ButtonFunctionExecutor)

	// Register handlers that need coordinates
	handlers[maximizeFunctionName] = CoordinatesButtonFunctionExecutor{fn: a.MaximizeWindow} // Use method value
	handlers[minimizeFunctionName] = CoordinatesButtonFunctionExecutor{fn: a.MinimizeWindow} // Use method value

	// Register basic handlers
	handlers[closeFunctionName] = NoArgButtonFunctionExecutor{fn: CloseWindow} // Use method value (or static func if no 'a' needed)

	log.Printf("Initialized %d function handlers", len(handlers))
	return handlers
}
