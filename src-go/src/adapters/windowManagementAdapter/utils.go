package windowManagementAdapter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

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

        mapping[hwnd] = WindowInfo{
            Title:    cleanTitle,
            ExeName:  info.ExeName,
            ExePath:  info.ExePath,
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

// getWindowInfo gets information about a window
func getWindowInfo(hwnd win.HWND) (WindowMapping, string) {
	result := make(WindowMapping)
	windowTitle := GetWindowText(hwnd)

	if hwnd != 0 {
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		if pid != 0 {
			// exePathFromProcess will be the TARGET_PATH for Sourcetree
			exePathFromProcess, err := getProcessExePath(pid)
			if err != nil {
				// Handle error getting process path - return unknown app info
				result[hwnd] = WindowInfo{ Title: windowTitle, AppName: "ErrorApp", ExeName: "Error", IconPath: "" }
				return result, "ErrorApp"
			}

			if fileExists(exePathFromProcess) {
				exeNameFromProcess := strings.ToLower(filepath.Base(exePathFromProcess))

				// --- AppName and Key Lookup ---
				appName := "Unknown App"
				// keyUsedForIcon is the path string used as the key in discoveredApps
				// and therefore used for icon creation/lookup. Initialize to empty.
				keyUsedForIcon := ""

				// Try exact match first using the process path (might work for non-launcher apps)
				if app, exists := discoveredApps[exePathFromProcess]; exists {
					appName = app.Name
					keyUsedForIcon = exePathFromProcess // Found by exact match
				} else {
					// Fallback: Try matching basename against keys in discoveredApps
					for mapKey, app := range discoveredApps { // mapKey is the LAUNCHER_PATH for Sourcetree
						if strings.EqualFold(filepath.Base(mapKey), exeNameFromProcess) {
							appName = app.Name
							keyUsedForIcon = mapKey // CAPTURE THE MAP KEY!
							break
						}
					}
				}
				// --- End AppName and Key Lookup ---

				// --- Get icon path using the CORRECT key ---
				iconPath := ""
				if keyUsedForIcon != "" { // If we found a matching key in discoveredApps
					// Use the captured key (LAUNCHER_PATH for Sourcetree) to get the icon path
					if path, err := GetIconPathForExe(keyUsedForIcon); err == nil {
						iconPath = path
					} else {
						log.Printf("Warning: GetIconPathForExe failed for known key '%s': %v", keyUsedForIcon, err)
					}
				} else {
					// We couldn't associate this running process with any known app in discoveredApps
					// based on exact path or basename. Icon path remains empty.
					log.Printf("Warning: Could not find key in discoveredApps matching process '%s' (basename '%s') to determine icon path.",
						exePathFromProcess, exeNameFromProcess)
				}
				// --- End Get icon path ---

				result[hwnd] = WindowInfo{
					Title:    windowTitle,
					ExeName:  exeNameFromProcess,         // Basename from process
					ExePath:  exePathFromProcess,         // Full path from process (TARGET_PATH)
					AppName:  appName,                    // Name found from discoveredApps
					Instance: 0,                          // Instance logic not shown/needed here
					IconPath: iconPath,                   // Path derived using keyUsedForIcon
				}
				return result, appName // Return appName found
			}
			// else: exePathFromProcess doesn't exist (unlikely if process running)
		}
		// Fallback if pid is 0 or getProcessExePath fails or file doesn't exist
		result[hwnd] = WindowInfo{ Title: windowTitle, AppName: "Unknown App", ExeName: "Unknown", IconPath: "" }
	} // else: hwnd is 0
	return result, "Unknown App"
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