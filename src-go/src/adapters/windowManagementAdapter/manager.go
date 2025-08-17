package windowManagementAdapter

import (
	"runtime"
	"unsafe"

	"maps"
	"slices"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// enumContext holds per-enumeration state passed to the EnumWindows callback via lparam
type enumContext struct {
	tempWindowMapping WindowMapping
	thisWindow        win.HWND
	exclusionConfig   *ExclusionConfig
}

// Package-level EnumWindows callback to avoid allocating callbacks repeatedly
var enumWindowsProc = windows.NewCallback(func(hwnd win.HWND, lparam uintptr) uintptr {
	ctx := (*enumContext)(unsafe.Pointer(lparam))

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

	// Call passesInitialFilter
	if passesInitialFilter(hwnd, tempTitle, tempClassName, int(tempIsCloaked), ctx.thisWindow, ctx.exclusionConfig) {
		infoMap, appName := getWindowInfo(hwnd)

		// Create a temporary map to hold the cleaned window info
		cleanedInfoMap := make(WindowMapping)
		cleanWindowTitles(cleanedInfoMap, infoMap, appName)
		cleanedInfo := cleanedInfoMap[hwnd]

		// Perform exclusion check on the cleaned title
		isExcluded := false
		if slices.Contains(ctx.exclusionConfig.ExcludedTitles, cleanedInfo.Title) {
			isExcluded = true
		} else if slices.Contains(ctx.exclusionConfig.ExcludedApps, cleanedInfo.AppName) {
			isExcluded = true
		} else {
			for _, specific := range ctx.exclusionConfig.SpecificExclusions {
				if cleanedInfo.AppName == specific.App && cleanedInfo.Title == specific.Title {
					isExcluded = true
					break
				}
			}
		}

		if !isExcluded {
			ctx.tempWindowMapping[hwnd] = cleanedInfo
		}
	}
	return 1 // TRUE
})

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
func GetFilteredListOfWindows(winManager *WindowManager, thisWindow win.HWND, exclusionConfig *ExclusionConfig) WindowMapping {
	tempWindowMapping := make(WindowMapping) // Target map

	// Prepare context to pass via lparam
	ctx := enumContext{
		tempWindowMapping: tempWindowMapping,
		thisWindow:        thisWindow,
		exclusionConfig:   exclusionConfig,
	}

	procEnumWindows.Call(enumWindowsProc, uintptr(unsafe.Pointer(&ctx)))
	runtime.KeepAlive(&ctx)

	tempWindowMapping = assignInstanceNumbers(tempWindowMapping, winManager.GetOpenWindowsInfo())
	return tempWindowMapping
}

// PrintWindowList prints the current window list for debugging
func PrintWindowList(mapping WindowMapping) {
	log.Info("------------------ Current Window List ------------------")
	for hwnd, info := range mapping {
		log.Info("Window Handle: %v", hwnd)
		log.Info("  Title: %s", info.Title)
		log.Info("  ExeName: %s", info.ExeName)
		log.Info("  AppName: %s", info.AppName)
		log.Info("  Instance: %d", info.Instance)
		log.Info("  IconPath: %s", info.IconPath)
		log.Info("")
	}
	log.Info("---------------------------------------------------------")
}
