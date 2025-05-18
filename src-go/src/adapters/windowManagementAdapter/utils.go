package windowManagementAdapter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// GetWindowText gets the title of a window
func GetWindowText(hwnd win.HWND) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if textLen == 0 {
		return ""
	}
	buf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(textLen+1))
	return windows.UTF16ToString(buf)
}

// GetClassName gets the class name of a window
func GetClassName(hwnd win.HWND) string {
	var buf [256]uint16
	len, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if len == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:len])
}

// IsWindowVisible checks if a window is visible
func IsWindowVisible(hwnd win.HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

// AddHwndToExclude adds a window handle to the exclusion list
func AddHwndToExclude(hwnd win.HWND) {
	hwndToExclude = append(hwndToExclude, hwnd)
}

// shouldIncludeWindow determines if a window should be included in the window list
func shouldIncludeWindow(hwnd win.HWND, windowTitle, className string, isCloaked int, thisWindow win.HWND) bool {

	// Check specific HWND exclusion list
	for _, excluded := range hwndToExclude {
		if hwnd == excluded {
			return false
		}
	}

	// Check main properties
	return isCloaked == 0 &&
		strings.TrimSpace(windowTitle) != "" &&
		!excludedClassNames[className] &&
		hwnd != thisWindow
}

// cleanWindowTitles updates window titles in the mapping
func cleanWindowTitles(mapping WindowMapping, entry WindowMapping, appName string) {
	for hwnd, info := range entry {
		cleanTitle := info.Title

		if info.ExeName == "explorer.exe" && strings.Contains(info.Title, " - File Explorer") {
			cleanTitle = strings.Replace(info.Title, " - File Explorer", "", -1)
		} else if strings.Contains(info.Title, " - "+appName) {
			cleanTitle = strings.Replace(info.Title, " - "+appName, "", -1)
		}

		mapping[hwnd] = core.WindowInfo{
			Title:    cleanTitle,
			ExeName:  info.ExeName,
			AppName:  info.AppName,
			IconPath: info.IconPath,
			Instance: 0,
		}
	}
}

// assignInstanceNumbers assigns instance numbers to windows with the same title and exe
func assignInstanceNumbers(tempMapping WindowMapping, existingMapping WindowMapping) WindowMapping {
	resultMapping := make(WindowMapping)

	// Track used instance numbers for each title/exe pair
	titleExeMapping := make(map[string]map[int]bool)

	// First register all instances from existing mapping
	for _, info := range existingMapping {
		key := fmt.Sprintf("%s|%s", info.Title, info.ExeName)
		if _, exists := titleExeMapping[key]; !exists {
			titleExeMapping[key] = make(map[int]bool)
		}
		titleExeMapping[key][info.Instance] = true
	}

	// Process each window
	for hwnd, info := range tempMapping {
		// If window exists in existing mapping, update title and exe but keep instance number
		if existingInfo, exists := existingMapping[hwnd]; exists {
			newInfo := tempMapping[hwnd]
			if newInfo.Title != existingInfo.Title {
				newInfo.Instance = 0
			} else {
				newInfo.Instance = existingInfo.Instance
				resultMapping[hwnd] = newInfo
				continue
			}
		}

		key := fmt.Sprintf("%s|%s", info.Title, info.ExeName)
		if _, exists := titleExeMapping[key]; !exists {
			titleExeMapping[key] = make(map[int]bool)
		}

		// Find next available instance number
		newInstance := 0
		for titleExeMapping[key][newInstance] {
			newInstance++
		}

		// Add new instance to tracking
		titleExeMapping[key][newInstance] = true
		info.Instance = newInstance
		resultMapping[hwnd] = info
	}

	return resultMapping
}

// getWindowInfo gets information about a window by its handle (HWND).
// It attempts to identify the application using the installedAppsInfo map and returns
// a WindowMapping containing details and the identified application name.
func getWindowInfo(hwnd win.HWND) (WindowMapping, string) {
	result := make(WindowMapping)
	windowTitle := GetWindowText(hwnd) // Assume GetWindowText is defined

	// Default values if app cannot be fully identified
	defaultAppName := "Unknown App"
	defaultExeName := "Unknown"

	if hwnd == 0 {
		result[hwnd] = core.WindowInfo{Title: windowTitle, AppName: defaultAppName, ExeName: defaultExeName, IconPath: ""}
		return result, defaultAppName
	}

	var pid uint32
	// Assume procGetWindowThreadProcessId and other win.* types/funcs are available
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

	if pid == 0 {
		result[hwnd] = core.WindowInfo{Title: windowTitle, AppName: defaultAppName, ExeName: defaultExeName, IconPath: ""}
		return result, defaultAppName
	}

	// exePathFromProcess is the full path to the actual running executable.
	exePathFromProcess, err := getProcessExePath(pid) // Assume getProcessExePath is defined
	if err != nil {
		log.Printf("Error getting process exe path for PID %d: %v", pid, err)
		// Use a distinct AppName to indicate this specific error state
		errorAppName := "ErrorApp"
		result[hwnd] = core.WindowInfo{Title: windowTitle, AppName: errorAppName, ExeName: "Error", IconPath: ""}
		return result, errorAppName
	}

	// Ensure the path obtained is not empty and theoretically exists.
	// fileExists might be redundant if getProcessExePath already ensures validity,
	// but it's a safe check.
	if exePathFromProcess == "" || !fileExists(exePathFromProcess) { // Assume fileExists is defined
		log.Printf("Warning: Process exe path '%s' for PID %d is invalid or file does not exist.", exePathFromProcess, pid)
		result[hwnd] = core.WindowInfo{Title: windowTitle, AppName: defaultAppName, ExeName: defaultExeName, IconPath: ""}
		return result, defaultAppName
	}

	exeNameFromProcess := strings.ToLower(filepath.Base(exePathFromProcess))

	// --- Identify AppName and IconPath from installedAppsInfo ---
	identifiedAppName := defaultAppName // This will be the key from the installedAppsInfo map (e.g., "My App")
	appIconPath := ""                   // This will be appLaunchInfo.IconPath from installedAppsInfo

	var bestMatchInfo *core.AppInfo // Using pointer to distinguish from zero-value struct
	var bestMatchKey string         // The AppName key from installedAppsInfo

	// Priority 1: Exact match of the running process's ExePath against AppInfo.ExePath
	for appKey, appInfoEntry := range installedAppsInfo {
		// appKey is the unique application name (e.g., "Firefox", "Firefox (1)")
		// appInfoEntry.ExePath is the launcher/configured path for this discovered application
		if appInfoEntry.ExePath != "" && strings.EqualFold(appInfoEntry.ExePath, exePathFromProcess) {
			// Must copy appInfoEntry if we were to take its address and it's a loop variable used later by pointer.
			// Here, we only need its fields, so direct use or copy is fine.
			// Since we just need its fields for now and not the pointer to the loop variable, this is safe.
			tempAppInfo := appInfoEntry // Create a local copy to safely point to if needed
			bestMatchInfo = &tempAppInfo
			bestMatchKey = appKey
			break // Found the best possible match
		}
	}

	// Priority 2: Basename match if no exact ExePath match was found
	if bestMatchInfo == nil {
		for appKey, appInfoEntry := range installedAppsInfo {
			if appInfoEntry.ExePath != "" && strings.EqualFold(filepath.Base(appInfoEntry.ExePath), exeNameFromProcess) {
				tempAppInfo := appInfoEntry
				bestMatchInfo = &tempAppInfo
				bestMatchKey = appKey
				break // Take the first basename match encountered
			}
		}
	}

	if bestMatchInfo != nil {
		identifiedAppName = bestMatchKey     // Use the map key (e.g., "My App") as the AppName
		appIconPath = bestMatchInfo.IconPath // Directly use the pre-resolved icon path
	} else {
		// Process not found in installedAppsInfo by its ExePath or basename.
		// It might be an app not in our list, or a transient system process.
		// Icon path remains empty if not associated with an entry in installedAppsInfo.
		// TODO: Maybe save this not-found app to not process it again every time
		// log.Printf("Info: Running process '%s' (basename: '%s') not found in installedAppsInfo. AppName set to '%s'.",
		// 	exePathFromProcess, exeNameFromProcess, identifiedAppName)
		// Optional: if you still want an icon for a completely unknown app, you could call GetIconPathForExe here:
		// genericIconPath, errIcon := GetIconPathForExe(exePathFromProcess)
		// if errIcon == nil { appIconPath = genericIconPath }
	}
	// --- End AppName and IconPath Lookup ---

	result[hwnd] = core.WindowInfo{
		Title:    windowTitle,
		ExeName:  exeNameFromProcess, // Basename from the actual running process
		AppName:  identifiedAppName,  // Name identified from installedAppsInfo (map key)
		Instance: 0,                  // Instance logic is not part of this snippet
		IconPath: appIconPath,        // IconPath from installedAppsInfo.AppInfo
	}
	return result, identifiedAppName
}

// getProcessExePath gets the executable path for a process
func getProcessExePath(pid uint32) (string, error) {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("failed to open process: %w", err)
	}
	defer syscall.CloseHandle(handle)

	var buf [syscall.MAX_PATH]uint16
	size := uint32(syscall.MAX_PATH)

	queryFullProcessImageName := syscall.NewLazyDLL("kernel32.dll").NewProc("QueryFullProcessImageNameW")
	ret, _, err := queryFullProcessImageName.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to query process image name: %w", err)
	}

	return syscall.UTF16ToString(buf[:size]), nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
