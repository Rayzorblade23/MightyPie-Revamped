package buttonManagerAdapter

import "encoding/json"

// WindowInfo represents information about a single window (used in NATS messages and internally)
type WindowInfo struct {
	Title    string `json:"Title"`
	ExeName  string `json:"ExeName"`
	ExePath  string `json:"ExePath"`
	AppName  string `json:"AppName"`
	Instance int    `json:"Instance"`
	IconPath string `json:"IconPath"`
}

// WindowsUpdate represents the structure received via NATS containing the current window list,
// mapping window handle (int) to WindowInfo.
type WindowsUpdate map[int]WindowInfo

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
	ExePath         string `json:"exe_path"`          // PRE-CONFIGURED - Used for matching, DO NOT OVERWRITE from WindowInfo
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
}

// Task represents a single task configuration
type Task struct {
	TaskType   string          `json:"task_type"`
	Properties json.RawMessage `json:"properties"`
}

type ButtonMap map[string]Task       // ButtonID (string) -> Task
type ConfigData map[string]ButtonMap // MenuID (string) -> ButtonMap
