package buttonManagerAdapter

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Constants for shortening thresholds
const maxPathDisplayLength = 40  // Max length before shortening paths
const maxTextDisplayLength = 30  // Max length before shortening other text
const ellipsis = "..."

// // PrintWindowList prints the current window list for debugging
func PrintWindowList(mapping map[int]WindowInfo) {
	fmt.Println("------------------ Current Window List ------------------")
	if len(mapping) == 0 {
		fmt.Println("(empty)")
		return
	}
	for hwnd, info := range mapping {
		fmt.Printf("Window Handle: %d\n", hwnd)
		fmt.Printf("  Title: %s\n", info.Title)
		fmt.Printf("  ExeName: %s\n", info.ExeName)
		fmt.Printf("  ExePath: %s\n", info.ExePath)
		fmt.Printf("  AppName: %s\n", info.AppName)
		fmt.Printf("  Instance: %d\n", info.Instance)
		fmt.Printf("  IconPath: %s\n", info.IconPath)
		fmt.Println()
	}
	fmt.Println("---------------------------------------------------------")
}

func PrintTask(task Task) {
	fmt.Printf("Task Type: %s\n", task.TaskType)

	switch task.TaskType {
	case string(TaskTypeShowProgramWindow):
		props, err := GetTaskProperties[ShowProgramWindowProperties](task)
		if err != nil {
			fmt.Printf("Error parsing properties: %v\n", err)
			return
		}
		fmt.Printf("Properties:\n")
		fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
		fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)
		fmt.Printf("  Icon Path: %s\n", props.IconPath)
		fmt.Printf("  Window Handle: %d\n", props.WindowHandle)
		fmt.Printf("  Exe Path: %s\n", props.ExePath)

	case string(TaskTypeShowAnyWindow):
		props, err := GetTaskProperties[ShowAnyWindowProperties](task)
		if err != nil {
			fmt.Printf("Error parsing properties: %v\n", err)
			return
		}
		fmt.Printf("Properties:\n")
		fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
		fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)
		fmt.Printf("  Icon Path: %s\n", props.IconPath)
		fmt.Printf("  Window Handle: %d\n", props.WindowHandle)
		fmt.Printf("  Exe Path: %s\n", props.ExePath)

		// ... add other cases as needed
	}
}

// formatProperties concatenates non-empty strings with ", ".
func formatProperties(parts ...string) string {
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, ", ")
}

// PrintConfig displays the button configuration, conditionally shortening long strings.
func PrintConfig(config ConfigData, shorten bool) { // Added 'shorten' parameter
	var sb strings.Builder

	sb.WriteString("\n======================= Mighty Pie Configuration =======================\n")

	if len(config) == 0 {
		sb.WriteString("  (No profiles configured or configuration is nil)\n")
		sb.WriteString("======================================================================\n")
		fmt.Print(sb.String())
		return
	}

	// --- Iterate Profiles ---
	profileIDs := make([]string, 0, len(config))
	for id := range config { profileIDs = append(profileIDs, id) }
	sort.Strings(profileIDs)

	for i, profileID := range profileIDs {
		if i > 0 { sb.WriteString("----------------------------------------------------------------------\n") }
		menuConfig := config[profileID]
		fmt.Fprintf(&sb, "Profile: %s\n", profileID)

		if len(menuConfig) == 0 {
			sb.WriteString("  (No menus configured for this profile)\n")
			continue
		}

		// --- Iterate Menus ---
		menuIDs := make([]string, 0, len(menuConfig))
		for id := range menuConfig { menuIDs = append(menuIDs, id) }
		sort.Strings(menuIDs)

		for j, menuID := range menuIDs {
			if j > 0 { sb.WriteString("  ---\n") }
			buttonMap := menuConfig[menuID]
			fmt.Fprintf(&sb, "  Menu: %s\n", menuID)

			if len(buttonMap) == 0 {
				sb.WriteString("    (No buttons configured for this menu)\n")
				continue
			}

			// --- Iterate Buttons ---
			buttonIDs := make([]int, 0, len(buttonMap))
			for idStr := range buttonMap {
				id, err := strconv.Atoi(idStr)
				if err != nil { log.Printf("WARN: Invalid button ID format '%s' in P:%s M:%s", idStr, profileID, menuID); continue }
				buttonIDs = append(buttonIDs, id)
			}
			sort.Ints(buttonIDs)

			for _, buttonID := range buttonIDs {
				buttonIDStr := strconv.Itoa(buttonID)
				task := buttonMap[buttonIDStr]
				fmt.Fprintf(&sb, "    Btn %2d: [%-20s] ", buttonID, task.TaskType)

				taskSpecificDetails := ""
				switch TaskType(task.TaskType) {
				case TaskTypeShowAnyWindow:
					props, err := GetTaskProperties[ShowAnyWindowProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for ShowAnyWindow (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
						taskSpecificDetails = "<Error reading props>"
					} else {
						taskSpecificDetails = formatProperties(
							// Use shortenString for text fields
							fmt.Sprintf("Upper: '%s'", shortenString(props.ButtonTextUpper, maxTextDisplayLength, shorten)),
							fmt.Sprintf("Lower: '%s'", shortenString(props.ButtonTextLower, maxTextDisplayLength, shorten)),
							// Use shortenPath for path fields
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", shortenPath(props.IconPath, maxPathDisplayLength, shorten))),
							condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", shortenPath(props.ExePath, maxPathDisplayLength, shorten))),
							condStr(props.WindowHandle != InvalidHandle, fmt.Sprintf("HWND: %d", props.WindowHandle)),
						)
					}
				case TaskTypeShowProgramWindow:
					props, err := GetTaskProperties[ShowProgramWindowProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for ShowProgramWindow (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
						taskSpecificDetails = "<Error reading props>"
					} else {
						taskSpecificDetails = formatProperties(
							// Use shortenString for text fields
							fmt.Sprintf("Upper: '%s'", shortenString(props.ButtonTextUpper, maxTextDisplayLength, shorten)),
							fmt.Sprintf("Lower: '%s'", shortenString(props.ButtonTextLower, maxTextDisplayLength, shorten)),
							// Use shortenPath for path fields
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", shortenPath(props.IconPath, maxPathDisplayLength, shorten))),
							condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", shortenPath(props.ExePath, maxPathDisplayLength, shorten))),
							condStr(props.WindowHandle != InvalidHandle, fmt.Sprintf("HWND: %d", props.WindowHandle)),
						)
					}
				case TaskTypeCallFunction:
					props, err := GetTaskProperties[CallFunctionProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for CallFunction (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
						taskSpecificDetails = "<Error reading props>"
					} else {
						taskSpecificDetails = formatProperties(
							// Use shortenString for text fields
							fmt.Sprintf("Upper: '%s'", shortenString(props.ButtonTextUpper, maxTextDisplayLength, shorten)),
							fmt.Sprintf("Lower: '%s'", shortenString(props.ButtonTextLower, maxTextDisplayLength, shorten)),
							// Use shortenPath for icon path
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", shortenPath(props.IconPath, maxPathDisplayLength, shorten))),
						)
					}
				case TaskTypeLaunchProgram:
					props, err := GetTaskProperties[LaunchProgramProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for LaunchProgram (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
						taskSpecificDetails = "<Error reading props>"
					} else {
						taskSpecificDetails = formatProperties(
							// Use shortenString for text fields
							fmt.Sprintf("Upper: '%s'", shortenString(props.ButtonTextUpper, maxTextDisplayLength, shorten)),
							fmt.Sprintf("Lower: '%s'", shortenString(props.ButtonTextLower, maxTextDisplayLength, shorten)),
							// Use shortenPath for path fields
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", shortenPath(props.IconPath, maxPathDisplayLength, shorten))),
							condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", shortenPath(props.ExePath, maxPathDisplayLength, shorten))),
						)
					}
				case TaskTypeDisabled:
					taskSpecificDetails = "(Disabled)"
				default:
					taskSpecificDetails = fmt.Sprintf("(Unknown Task Type: %s)", task.TaskType)
				}

				if taskSpecificDetails != "" { sb.WriteString(taskSpecificDetails) }
				sb.WriteString("\n")
			} // End Button Loop
		} // End Menu Loop
	} // End Profile Loop

	sb.WriteString("======================================================================\n")
	fmt.Print(sb.String())
}

// condStr returns str if condition is true, otherwise an empty string.
func condStr(condition bool, str string) string {
	if condition {
		return str
	}
	return ""
}

// shortenString conditionally shortens a general string with an ellipsis.
// If shouldShorten is false, or if the string is within maxLen, it returns the original.
func shortenString(s string, maxLen int, shouldShorten bool) string {
	if !shouldShorten || s == "" || len(s) <= maxLen {
		return s
	}
	// Ensure we have enough space for the ellipsis itself
	if maxLen <= len(ellipsis) {
		return s[:maxLen] // Just truncate if not enough space for ellipsis
	}
	return s[:maxLen-len(ellipsis)] + ellipsis
}

// shortenPath conditionally returns "...<basename>" if the path length > maxLen.
// If shouldShorten is false, or if the path is within maxLen, it returns the original.
func shortenPath(path string, maxLen int, shouldShorten bool) string {
	if !shouldShorten || path == "" || len(path) <= maxLen {
		return path
	}
	// Use filepath.Base for cross-platform compatibility
	base := filepath.Base(path)
	// Add ellipsis, ensure filename itself isn't overly long (optional refinement)
	// simple version:
	return ellipsis + base
}