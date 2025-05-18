package core

type AppInfo struct {
	ExePath          string `json:"exePath"`                    // The resolved executable path
	WorkingDirectory string `json:"workingDirectory,omitempty"` // Working directory from LNK
	Args             string `json:"args,omitempty"`             // Command line args from LNK
	URI              string `json:"uri,omitempty"`              // Add this field for store apps
	IconPath         string `json:"iconPath,omitempty"`         // Path to the icon file
}

// WindowsUpdate represents the structure received via NATS containing the current window list,
// mapping window handle (int) to core.WindowInfo.
type WindowsUpdate map[int]WindowInfo

// core.WindowInfo represents information about a single window (used in NATS messages etc.)
type WindowInfo struct {
	Title    string `json:"Title"`
	ExeName  string `json:"ExeName"`
	AppName  string `json:"AppName"`
	Instance int    `json:"Instance"`
	IconPath string `json:"IconPath"`
}


// --------------------------------------------
// --------- Button Type properties -----------
// --------------------------------------------


type ShowAnyWindowProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Window Title
	ButtonTextLower string `json:"button_text_lower"` // AppName
	IconPath        string `json:"icon_path"`
	WindowHandle    int    `json:"window_handle"`
}

type ShowProgramWindowProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // Window Title
	ButtonTextLower string `json:"button_text_lower"` // AppName
	IconPath        string `json:"icon_path"`
	WindowHandle    int    `json:"window_handle"`
}

type LaunchProgramProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // AppName
	ButtonTextLower string `json:"button_text_lower"` // " - Launch - "
	IconPath        string `json:"icon_path"`
}

type CallFunctionProperties struct {
	ButtonTextUpper string `json:"button_text_upper"` // function name
	ButtonTextLower string `json:"button_text_lower"` // empty string
	IconPath        string `json:"icon_path"`
}
