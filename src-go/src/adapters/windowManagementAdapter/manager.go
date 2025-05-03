package windowManagementAdapter

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"maps"
)

// NewWindowManager creates a new window manager
func NewWindowManager() *WindowManager {
	return &WindowManager{
		openWindowsInfo: make(WindowMapping),
	}
}

// UpdateOpenWindowsInfo updates the manager's window information
func (wm *WindowManager) UpdateOpenWindowsInfo(mapping WindowMapping) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	wm.openWindowsInfo = mapping
}

// GetOpenWindowsInfo returns a copy of the open windows info
func (wm *WindowManager) GetOpenWindowsInfo() WindowMapping {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	// Create a copy of the map to avoid concurrent modification issues
	result := make(WindowMapping)
	maps.Copy(result, wm.openWindowsInfo)
	return result
}

// GetFilteredListOfWindows returns a list of filtered windows
func GetFilteredListOfWindows(winManager *WindowManager, thisWindow win.HWND) WindowMapping {
	tempWindowMapping := make(WindowMapping) // Target map

	enumFunc := windows.NewCallback(func(hwnd win.HWND, lparam uintptr) uintptr {

		// Get window properties
		tempIsVisible := IsWindowVisible(hwnd)
		if !tempIsVisible {
			return 1
		} // Early exit if not visible

		tempTitle := GetWindowText(hwnd)
		tempClassName := GetClassName(hwnd)

		var tempIsCloaked int32

		// Call DwmGetWindowAttribute to check if the window is cloaked (e.g., hidden or in a virtual desktop)
		// Result indicates success (S_OK) or failure (HRESULT error code)
		result, _, _ := procDwmGetWindowAttribute.Call(
			uintptr(hwnd),
			uintptr(DWMWA_CLOAKED),
			uintptr(unsafe.Pointer(&tempIsCloaked)),
			unsafe.Sizeof(tempIsCloaked),
		)

		if result != 0 {
			tempIsCloaked = 0
		} // Default to not cloaked on failure

		// Call shouldIncludeWindow
		if shouldIncludeWindow(hwnd, tempTitle, tempClassName, int(tempIsCloaked), thisWindow) {
			infoMap, appName := getWindowInfo(hwnd)
			cleanWindowTitles(tempWindowMapping, infoMap, appName)
		}
		return 1 // TRUE
	})

	procEnumWindows.Call(enumFunc, 0)
	runtime.KeepAlive(enumFunc)

	tempWindowMapping = assignInstanceNumbers(tempWindowMapping, winManager.GetOpenWindowsInfo())
	return tempWindowMapping
}

// PrintWindowList prints the current window list for debugging
func PrintWindowList(mapping WindowMapping) {
	fmt.Println("------------------ Current Window List ------------------")
	for hwnd, info := range mapping {
		fmt.Printf("Window Handle: %v\n", hwnd)
		fmt.Printf("  Title: %s\n", info.Title)
		fmt.Printf("  ExeName: %s\n", info.ExeName)
		fmt.Printf("  ExePath: %s\n", info.ExePath)
		fmt.Printf("  AppName: %s\n", info.AppName)
		fmt.Printf("  Instance: %d\n", info.Instance)
		fmt.Println()
	}
}
