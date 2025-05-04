package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

// PrintConfig displays the current button configuration in a readable format.
// This function NOW includes cases for all relevant task types.
func PrintConfig(config ConfigData) {
	fmt.Println("\n================== Configuration ==================")

	// Sort profile IDs if necessary (maps don't guarantee order)
	profileIDs := make([]string, 0, len(config))
	for id := range config {
		profileIDs = append(profileIDs, id)
	}
	sort.Strings(profileIDs)

	for _, profileID := range profileIDs {
		buttonMap := config[profileID]
		fmt.Printf("\nMenu %s:\n", profileID)

		// Sort button IDs for consistent printing order
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

			fmt.Printf("\n  Button %d:\n", buttonID)
			fmt.Println("  -------------------")
			fmt.Printf("Task Type: %s\n", task.TaskType)

			// --- SWITCH TO HANDLE PRINTING FOR DIFFERENT TYPES ---
			switch TaskType(task.TaskType) {
			case TaskTypeShowAnyWindow:
				props, err := GetTaskProperties[ShowAnyWindowProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get/print ShowAnyWindowProperties for %s:%s - %v", profileID, buttonIDStr, err)
					fmt.Println("Properties: <Error reading>")
					continue
				}
				fmt.Println("Properties:")
				fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
				fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)
				fmt.Printf("  Icon Path: %s\n", props.IconPath)
				fmt.Printf("  Window Handle: %d\n", props.WindowHandle)
				fmt.Printf("  Exe Path: %s\n", props.ExePath)

			case TaskTypeShowProgramWindow:
				props, err := GetTaskProperties[ShowProgramWindowProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get/print ShowProgramWindowProperties for %s:%s - %v", profileID, buttonIDStr, err)
					fmt.Println("Properties: <Error reading>")
					continue
				}
				fmt.Println("Properties:")
				fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
				fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)
				fmt.Printf("  Icon Path: %s\n", props.IconPath)
				fmt.Printf("  Window Handle: %d\n", props.WindowHandle)
				fmt.Printf("  Exe Path: %s\n", props.ExePath)

			// --- ADDED CASE for CallFunction ---
			case TaskTypeCallFunction:
				props, err := GetTaskProperties[CallFunctionProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get/print CallFunctionProperties for %s:%s - %v", profileID, buttonIDStr, err)
					fmt.Println("Properties: <Error reading>")
					continue
				}
				fmt.Println("Properties:")
				fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
				fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)

			// --- ADDED CASE for LaunchProgram ---
			case TaskTypeLaunchProgram:
				props, err := GetTaskProperties[LaunchProgramProperties](task)
				if err != nil {
					log.Printf("ERROR: Failed to get/print LaunchProgramProperties for %s:%s - %v", profileID, buttonIDStr, err)
					fmt.Println("Properties: <Error reading>")
					continue
				}
				fmt.Println("Properties:")
				fmt.Printf("  Button Text Upper: %s\n", props.ButtonTextUpper)
				fmt.Printf("  Button Text Lower: %s\n", props.ButtonTextLower)
				fmt.Printf("  Icon Path: %s\n", props.IconPath)
				fmt.Printf("  Exe Path: %s\n", props.ExePath)

			case TaskTypeDisabled:
				fmt.Println("Properties: (Disabled)")

			default:
				fmt.Printf("Properties: (Unknown Task Type: %s)\n", task.TaskType)
			}
		}
	}
	fmt.Println("================================================")
}
