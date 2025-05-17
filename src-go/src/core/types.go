package core

type AppLaunchInfo struct {
	ExePath          string `json:"exePath"`                    // The resolved executable path
	WorkingDirectory string `json:"workingDirectory,omitempty"` // Working directory from LNK
	Args             string `json:"args,omitempty"`             // Command line args from LNK
	URI              string `json:"uri,omitempty"`              // Add this field for store apps
	IconPath         string `json:"iconPath,omitempty"`         // Path to the icon file
}