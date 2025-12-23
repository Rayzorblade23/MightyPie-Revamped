package windowManagementAdapter

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

type focusedApp_Message struct {
	AppName string `json:"appName"`
}

// startFocusMonitoring sets up an event-based monitor for window focus changes
func (a *WindowManagementAdapter) startFocusMonitoring() {
	// Create callback for focus change events
	var focusEventCallback = windows.NewCallback(func(
		hWinEventHook windows.Handle,
		event uint32,
		hwnd windows.HWND,
		idObject int32,
		idChild int32,
		idEventThread uint32,
		dwmsEventTime uint32) uintptr {

		// Only process foreground window changes
		if event == EVENT_SYSTEM_FOREGROUND && hwnd != 0 {
			a.detectFocusedApp(hwnd)
		}
		return 0
	})

	// Set up the event hook for foreground window changes
	hook, _, _ := procSetWinEventHook.Call(
		uintptr(EVENT_SYSTEM_FOREGROUND), // eventMin
		uintptr(EVENT_SYSTEM_FOREGROUND), // eventMax
		0,                                // hmodWinEventProc
		focusEventCallback,               // callback
		0,                                // idProcess (0 = all processes)
		0,                                // idThread (0 = all threads)
		uintptr(WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS),
	)

	if hook == 0 {
		log.Error("Failed to set up focus change event hook")
		return
	}

	defer procUnhookWinEvent.Call(hook)

	log.Info("Focus change monitoring started (event-based)")

	// Message loop required for WINEVENT_OUTOFCONTEXT hooks
	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
	}
}

// detectFocusedApp detects the focused app for a specific window handle and publishes via NATS
func (a *WindowManagementAdapter) detectFocusedApp(hwnd windows.HWND) {
	// Convert windows.HWND to win.HWND for helper functions
	winHwnd := win.HWND(hwnd)

	// Check if this is an excluded window by class name
	className := GetClassName(winHwnd)
	if slices.Contains(a.exclusionConfig.ExcludedClassNames, className) {
		log.Debug("Excluded window class focused: %s, sending default", className)
		a.publishFocusedApp("default")
		return
	}

	// Get process ID from window handle
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		a.publishFocusedApp("default")
		return
	}

	// Get process executable path
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		a.publishFocusedApp("default")
		return
	}
	defer syscall.CloseHandle(handle)

	var buf [syscall.MAX_PATH]uint16
	size := uint32(syscall.MAX_PATH)

	ret, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		a.publishFocusedApp("default")
		return
	}

	exePath := syscall.UTF16ToString(buf[:size])
	exeName := strings.ToLower(filepath.Base(exePath))

	// Get window title for exclusion checks
	windowTitle := GetWindowText(winHwnd)

	// Check if this program is in the discovered apps list
	installedAppsInfoMutex.RLock()
	defer installedAppsInfoMutex.RUnlock()

	if len(installedAppsInfo) == 0 {
		a.publishFocusedApp("default")
		return
	}

	// Try to match by exact path first
	for appName, appInfo := range installedAppsInfo {
		if appInfo.ExePath != "" && strings.EqualFold(appInfo.ExePath, exePath) {
			// Check exclusions before publishing
			if a.isAppExcluded(appName, windowTitle) {
				log.Debug("Excluded app focused: %s, sending default", appName)
				a.publishFocusedApp("default")
				return
			}
			log.Debug("Focused window: %s (matched by path)", appName)
			a.publishFocusedApp(appName)
			return
		}
	}

	// Try to match by executable name
	for appName, appInfo := range installedAppsInfo {
		if appInfo.ExePath != "" && strings.EqualFold(filepath.Base(appInfo.ExePath), exeName) {
			// Check exclusions before publishing
			if a.isAppExcluded(appName, windowTitle) {
				log.Debug("Excluded app focused: %s, sending default", appName)
				a.publishFocusedApp("default")
				return
			}
			log.Debug("Focused window: %s (matched by exe name)", appName)
			a.publishFocusedApp(appName)
			return
		}
	}

	// Program not in discovered apps - send default
	log.Debug("Focused window not in discovered apps: %s", exeName)
	a.publishFocusedApp("default")
}

// isAppExcluded checks if an app/title combination should be excluded
func (a *WindowManagementAdapter) isAppExcluded(appName, title string) bool {
	// Check excluded apps
	if slices.Contains(a.exclusionConfig.ExcludedApps, appName) {
		return true
	}

	// Check excluded titles
	if slices.Contains(a.exclusionConfig.ExcludedTitles, title) {
		return true
	}

	// Check specific exclusions
	for _, specific := range a.exclusionConfig.SpecificExclusions {
		if appName == specific.App && title == specific.Title {
			return true
		}
	}

	return false
}

// publishFocusedApp publishes the focused app name via NATS
func (a *WindowManagementAdapter) publishFocusedApp(appName string) {
	msg := focusedApp_Message{
		AppName: appName,
	}
	a.natsAdapter.PublishMessage(os.Getenv("PUBLIC_NATSSUBJECT_FOCUSEDAPP_UPDATE"), msg)
}
