package pieButtonExecutionAdapter

// ButtonType represents the available button types
type ButtonType string

const (
	ButtonTypeShowProgramWindow ButtonType = "show_program_window"
	ButtonTypeShowAnyWindow     ButtonType = "show_any_window"
	ButtonTypeCallFunction      ButtonType = "call_function"
	ButtonTypeLaunchProgram     ButtonType = "launch_program"
	ButtonTypeDisabled          ButtonType = "disabled"
)

const (
	ClickTypeLeftUp   = "left_up"
	ClickTypeRightUp  = "right_up"
	ClickTypeMiddleUp = "middle_up"
)

// Message type for pie button execution
type pieButtonExecute_Message struct {
	PageIndex   int        `json:"page_index"`
	ButtonIndex int        `json:"button_index"`
	ButtonType  ButtonType `json:"button_type"`
	Properties  any        `json:"properties"`
	ClickType   string     `json:"click_type"`
}

type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}
