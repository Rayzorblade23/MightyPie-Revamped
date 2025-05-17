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