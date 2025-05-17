package buttonManagerAdapter

import (
	"encoding/json"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

type TaskType string

const (
	TaskTypeShowProgramWindow TaskType = "show_program_window"
	TaskTypeShowAnyWindow     TaskType = "show_any_window"
	TaskTypeCallFunction      TaskType = "call_function"
	TaskTypeLaunchProgram     TaskType = "launch_program"
	TaskTypeDisabled          TaskType = "disabled"
)

// Use -1 consistently for null/invalid handle
const InvalidHandle = -1

type ShowAnyWindowProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Mapped to Window Title
	ButtonTextLower string `json:"button_text_lower"` // Mapped to AppName
	IconPath        string `json:"icon_path"`         // Mapped to Window IconPath
	WindowHandle    int    `json:"window_handle"`     // Mapped to Window Handle
	ExePath         string `json:"exe_path"`          // Mapped to Window ExePath
}

type ShowProgramWindowProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Mapped to Window Title
	ButtonTextLower string `json:"button_text_lower"` // Mapped to AppName
	IconPath        string `json:"icon_path"`         // Mapped to Window IconPath
	WindowHandle    int    `json:"window_handle"`     // Mapped to Window Handle
	ExePath         string `json:"exe_path"`          // PRE-CONFIGURED - Used for matching, DO NOT OVERWRITE from core.WindowInfo
}

type LaunchProgramProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Should be AppName (ideally from cache)
	ButtonTextLower string `json:"button_text_lower"` // Should be " - Launch - " (or similar static text)
	IconPath        string `json:"icon_path"`         // Should be App Icon (ideally from cache)
	ExePath         string `json:"exe_path"`          // PRE-CONFIGURED
}

type CallFunctionProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Static function name
	ButtonTextLower string `json:"button_text_lower"` // Empty string
	IconPath        string `json:"icon_path"`         // Path to function icon
}

// Task represents a single task configuration
type Task struct {
	TaskType   string          `json:"task_type"`
	Properties json.RawMessage `json:"properties"`
}

// PageID (string, e.g., "0", "1") -> Tasks
type PageConfig map[string]Task

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
