package buttonManagerAdapter

import (
	"encoding/json"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// Use -1 consistently for null/invalid handle
const InvalidHandle = -1

// Button represents a single button configuration
type Button struct {
	ButtonType string          `json:"button_type"`
	Properties json.RawMessage `json:"properties"`
}

// PageID (string, e.g., "0", "1") -> Buttons
type PageConfig map[string]Button

// MenuID (string, e.g., "0", "1") -> PageConfigs
// This represents the configuration for all Pages within a single Menu.
type MenuConfig map[string]PageConfig

// ProfileID (string, e.g., "0", "1") -> MenuConfigs
// This is the new top-level type for the entire application's button configuration.
type ConfigData map[string]MenuConfig

// Helper types for ShowAnyWindow assignment
type availableSlotInfo struct {
	MenuID   string
	PageID   string
	ButtonID string
	// Store numeric IDs for easy sorting
	MenuIdx   int
	PageIdx   int
	ButtonIdx int
}

// Helper types for ShowAnyWindow assignment
type availableWindowInfo struct {
	Handle int
	Info   core.WindowInfo
}

// SeparatedButtons holds buttons separated by type for a single page
type SeparatedButtons struct {
	ShowProgram      map[string]*Button
	ShowAny          map[string]*Button
	LaunchProgram    map[string]*Button
	FunctionCall     map[string]*Button
	OpenPageInMenu   map[string]*Button
	OpenResource     map[string]*Button
	KeyboardShortcut map[string]*Button
}

// SeparatedButtonsCache holds separated buttons for all menus and pages
// Structure: MenuID -> PageID -> SeparatedButtons
type SeparatedButtonsCache map[string]map[string]*SeparatedButtons
