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
	ExePath  string `json:"ExePath"`
	AppName  string `json:"AppName"`
	Instance int    `json:"Instance"`
	IconPath string `json:"IconPath"`
}