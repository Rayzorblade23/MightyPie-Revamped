package windowManagementAdapter

import (
	"sync"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// WindowManagementAdapter is the main adapter struct
type WindowManagementAdapter struct {
	natsAdapter   *natsAdapter.NatsAdapter
	winManager    *WindowManager
	stopChan      chan struct{} // Adapter's overall stop
	windowWatcher *WindowWatcher
}

// shortcutPressed_Message represents a shortcut key press event
type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}

// WindowInfo stores information about a window
type WindowInfo struct {
	Title    string
	ExeName  string
	ExePath  string
	AppName  string
	Instance int
	IconPath string
}

// WindowMapping maps window handles to window information
type WindowMapping map[win.HWND]WindowInfo

// WindowManager keeps track of open windows
type WindowManager struct {
	openWindowsInfo WindowMapping
	mutex           sync.RWMutex
}

// WindowEvents represents window change events for publishing
type WindowEvents struct {
	WindowsChanged bool              `json:"windowsChanged"`
	Windows        map[string]string `json:"windows"`
}

// HWINEVENTHOOK is a Windows event hook handle
type HWINEVENTHOOK windows.Handle

// WindowWatcher watches for window events
type WindowWatcher struct {
	mutex          sync.RWMutex
	eventHook      HWINEVENTHOOK
	changeDetected chan struct{}
	stopChan       chan struct{} // Watcher's specific stop
	lastEventTime  time.Time
	isRunning      bool // Track if the hook loop goroutine is active
}

// MSG represents a Windows message
type MSG struct {
	HWnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      Point
}

// Point represents a 2D point
type Point struct {
	X, Y int32
}

type AppEntry struct {
	Name string
	Path string // Resolved executable path
	URI  string // Optional URI for store apps
}

type PackageInfo struct {
	PackageFamilyName string `json:"PackageFamilyName"`
	InstallLocation   string `json:"InstallLocation"`
}
