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
