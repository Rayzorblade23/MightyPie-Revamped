package buttonManagerAdapter

import "encoding/json"


type WindowInfo_Message struct {
    Title    string `json:"Title"`
    ExeName  string `json:"ExeName"`
    ExePath  string `json:"ExePath"`
    AppName  string `json:"AppName"`
    Instance int    `json:"Instance"`
    IconPath string `json:"IconPath"`
}

type WindowInfo struct {
    Title    string
    ExeName  string
    ExePath  string
    AppName  string
    Instance int
    IconPath string
}

type WindowsUpdate_Message map[int]WindowInfo_Message

type WindowMapping map[int]WindowInfo


type TaskType string

const (
    TaskTypeShowProgramWindow TaskType = "show_program_window"
    TaskTypeShowAnyWindow    TaskType = "show_any_window"
    TaskTypeCallFunction     TaskType = "call_function"
    TaskTypeLaunchProgram   TaskType = "launch_program"
    TaskTypeDisabled        TaskType = "disabled"
)

type ShowAnyWindowProperties struct {
    ButtonTextUpper string `json:"button_text_upper"`
    ButtonTextLower string `json:"button_text_lower"`
    IconPath        string `json:"icon_path"`
    WindowHandle    int    `json:"window_handle"`
    ExePath         string `json:"exe_path"`
}

type ShowProgramWindowProperties struct {
    ButtonTextUpper string `json:"button_text_upper"`
    ButtonTextLower string `json:"button_text_lower"`
    IconPath        string `json:"icon_path"`
    WindowHandle    int    `json:"window_handle"`
    ExePath         string `json:"exe_path"`
}

type LaunchProgramProperties struct {
    ButtonTextUpper string `json:"button_text_upper"`
    ButtonTextLower string `json:"button_text_lower"`
    IconPath        string `json:"icon_path"`
    ExePath         string `json:"exe_path"`
}

type CallFunctionProperties struct {
    ButtonTextUpper string `json:"button_text_upper"`
    ButtonTextLower string `json:"button_text_lower"`
}

// Task represents a single task configuration
type Task struct {
    TaskType   string          `json:"task_type"`
    Properties json.RawMessage `json:"properties"`
}

// ButtonMap represents a mapping of button indices to tasks
type ButtonMap map[string]Task

// ConfigData represents the entire configuration structure
type ConfigData map[string]ButtonMap