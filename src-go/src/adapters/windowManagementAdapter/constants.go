package windowManagementAdapter

import (
	"log"
	"os"
	"sync"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	EVENT_OBJECT_SHOW                 = 0x8002
	EVENT_OBJECT_HIDE                 = 0x8003
	WINEVENT_OUTOFCONTEXT             = 0x0000
	WINEVENT_SKIPOWNPROCESS           = 0x0002
	OBJID_WINDOW                      = 0
	CHILDID_SELF                      = 0
	GA_ROOTOWNER                      = 3
	WM_QUIT                           = 0x0012
	DWMWA_CLOAKED                     = 14
	MAX_PATH                          = 260
)

// Global variables
var (
	// Windows DLLs
	user32   = windows.NewLazySystemDLL("user32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	// DLL procs
	procSetWinEventHook            = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent             = user32.NewProc("UnhookWinEvent")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW         = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procGetClassNameW              = user32.NewProc("GetClassNameW")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procDwmGetWindowAttribute      = dwmapi.NewProc("DwmGetWindowAttribute")

	// Global variables
	hwndToExclude      []win.HWND
	excludedClassNames = map[string]bool{"Progman": true, "AutoHotkeyGUI": true, "RainmeterMeterWindow": true}
	logger             = log.New(os.Stdout, "[WindowManager] ", log.LstdFlags)

	// Active window watcher for callback access
	activeWindowWatcher *WindowWatcher

	// Mutex for thread safety when accessing activeWindowWatcher
	activeWatcherMutex sync.RWMutex
)