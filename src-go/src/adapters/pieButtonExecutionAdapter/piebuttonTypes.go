package pieButtonExecutionAdapter

// TaskType represents the available task types
type TaskType string

const (
	TaskTypeShowProgramWindow TaskType = "show_program_window"
	TaskTypeShowAnyWindow     TaskType = "show_any_window"
	TaskTypeCallFunction      TaskType = "call_function"
	TaskTypeLaunchProgram     TaskType = "launch_program"
	TaskTypeDisabled          TaskType = "disabled"
)

const (
	ClickTypeLeftUp   = "left_up"
	ClickTypeRightUp  = "right_up"
	ClickTypeMiddleUp = "middle_up"
)

// Message type for pie button execution
type pieButtonExecute_Message struct {
	MenuIndex   int      `json:"pageID"`
	ButtonIndex int      `json:"buttonID"`
	TaskType    TaskType `json:"task_type"`
	Properties  any      `json:"properties"`
	ClickType   string   `json:"click_type"`
}

type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}

// --------------------------------------------
// --------- Button Type properties -----------
// --------------------------------------------

// ShowWindowProperties contains common properties for window-related tasks
type ShowWindowProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // window title
	ButtonTextLower string `json:"button_text_lower"` // app name
	IconPath        string `json:"icon_path"`
	WindowHandle    int64  `json:"window_handle"`
	ExePath         string `json:"exe_path"`
}

// LaunchProgramProperties contains properties for launching programs
type LaunchProgramProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // app name
	ButtonTextLower string `json:"button_text_lower"` // " - Launch - "
	IconPath        string `json:"icon_path"`
	ExePath         string `json:"exe_path"`
}

// CallFunctionProperties contains properties for function calls
type CallFunctionProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // function name
	ButtonTextLower string `json:"button_text_lower"` // empty string
	IconPath        string `json:"icon_path"`
}
