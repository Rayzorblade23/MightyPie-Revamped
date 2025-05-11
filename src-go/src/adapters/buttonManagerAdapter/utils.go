package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// GetButtonConfig returns the current button configuration
func GetButtonConfig() ConfigData {
	return buttonConfig
}

func ReadButtonConfig() (ConfigData, error) {
	// Get user's AppData Local path
	localAppData := os.Getenv("LOCALAPPDATA")
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON
	var config ConfigData
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// Helper function to get typed properties from a task
func GetTaskProperties[T any](task Task) (T, error) {
	var props T
	if err := json.Unmarshal(task.Properties, &props); err != nil {
		return props, err
	}
	return props, nil
}

// SetTaskProperties updates the properties of a task with new values
func SetTaskProperties[T any](task *Task, props T) error {
	// Marshal the properties to JSON
	jsonData, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %v", err)
	}

	// Set the raw message
	task.Properties = json.RawMessage(jsonData)
	return nil
}


// ------- Printing Functions -------

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

// Helper to format properties concisely
func formatProperties(parts ...string) string {
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	if len(nonEmptyParts) == 0 {
		return ""
	}
	return strings.Join(nonEmptyParts, ", ")
}

// PrintConfig displays the current button configuration in a readable format.
func PrintConfig(config ConfigData) {
	var sb strings.Builder

	sb.WriteString("\n======================= Configuration =======================\n")

	profileIDs := make([]string, 0, len(config))
	for id := range config {
		profileIDs = append(profileIDs, id)
	}
	sort.Strings(profileIDs)

	for i, profileID := range profileIDs {
		if i > 0 {
			sb.WriteString("-----------------------------------------------------------\n") // Separator between profiles
		}
		buttonMap := config[profileID]
		fmt.Fprintf(&sb, "Profile: %s\n", profileID)

		buttonIDs := make([]int, 0, len(buttonMap))
		for idStr := range buttonMap {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Printf("WARN: Invalid button ID format '%s' in profile '%s'", idStr, profileID)
				continue
			}
			buttonIDs = append(buttonIDs, id)
		}
		sort.Ints(buttonIDs)

		for _, buttonID := range buttonIDs {
			buttonIDStr := strconv.Itoa(buttonID)
			task := buttonMap[buttonIDStr]
			
			fmt.Fprintf(&sb, "  Btn %2d: [%-20s] ", buttonID, task.TaskType) // Left-align TaskType

			taskSpecificDetails := ""

			switch TaskType(task.TaskType) {
			case TaskTypeShowAnyWindow:
				props, err := GetTaskProperties[ShowAnyWindowProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get props for ShowAnyWindow %s:%s - %v", profileID, buttonIDStr, err)
					taskSpecificDetails = "<Error reading props>"
				} else {
					taskSpecificDetails = formatProperties(
						fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
						fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
						condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
						condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", props.ExePath)),
						condStr(props.WindowHandle != 0, fmt.Sprintf("HWND: %d", props.WindowHandle)),
					)
				}
			case TaskTypeShowProgramWindow:
				props, err := GetTaskProperties[ShowProgramWindowProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get props for ShowProgramWindow %s:%s - %v", profileID, buttonIDStr, err)
					taskSpecificDetails = "<Error reading props>"
				} else {
					taskSpecificDetails = formatProperties(
						fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
						fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
						condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
						condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", props.ExePath)),
						condStr(props.WindowHandle != 0, fmt.Sprintf("HWND: %d", props.WindowHandle)),
					)
				}
			case TaskTypeCallFunction:
				props, err := GetTaskProperties[CallFunctionProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get props for CallFunction %s:%s - %v", profileID, buttonIDStr, err)
					taskSpecificDetails = "<Error reading props>"
				} else {
					taskSpecificDetails = formatProperties(
						fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
						fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
						condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
					)
				}
			case TaskTypeLaunchProgram:
				props, err := GetTaskProperties[LaunchProgramProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get props for LaunchProgram %s:%s - %v", profileID, buttonIDStr, err)
					taskSpecificDetails = "<Error reading props>"
				} else {
					taskSpecificDetails = formatProperties(
						fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
						fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
						condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
						condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", props.ExePath)),
					)
				}
			case TaskTypeDisabled:
				taskSpecificDetails = "(Disabled)"
			default:
				taskSpecificDetails = fmt.Sprintf("(Unknown Task Type: %s)", task.TaskType)
			}
			
			if taskSpecificDetails != "" {
				sb.WriteString(taskSpecificDetails)
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("===========================================================\n")
	fmt.Print(sb.String()) // Use fmt.Print as sb already contains newlines appropriately
}

// condStr is a helper to conditionally return a string.
// If condition is true, returns str; otherwise, returns an empty string.
// Useful for omitting empty property values in the formatted string.
func condStr(condition bool, str string) string {
	if condition {
		return str
	}
	return ""
}