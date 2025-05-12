package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// GetButtonConfig returns a DEEP COPY of the current button configuration for safe reading.
// This prevents external callers from accidentally modifying the shared state.
func GetButtonConfig() ConfigData {
	mu.RLock() // Lock for reading global state
	configToCopy := buttonConfig
	mu.RUnlock()

	// Perform deep copy
	copiedConfig, err := deepCopyConfig(configToCopy)
	if err != nil {
		// Log the error and return an empty config or handle as appropriate
		log.Printf("ERROR: Failed to deep copy button configuration: %v. Returning empty config.", err)
		return make(ConfigData) // Return empty map instead of nil or original
	}
	return copiedConfig
}

// ReadButtonConfig reads the configuration file and unmarshals it.
func ReadButtonConfig() (ConfigData, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Provide more context in error message
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	var config ConfigData // Uses NEW ConfigData definition from types.go
	if err := json.Unmarshal(data, &config); err != nil {
		// Provide more context in error message
		return nil, fmt.Errorf("failed to parse config file '%s': %w", configPath, err)
	}

	// Optional: Perform validation after unmarshaling if needed

	return config, nil
}

// deepCopyConfig creates a deep copy of ConfigData using JSON marshal/unmarshal.
func deepCopyConfig(src ConfigData) (ConfigData, error) {
	if src == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)

	if err := enc.Encode(src); err != nil {
		return nil, fmt.Errorf("failed to encode for deep copy: %w", err)
	}

	var dst ConfigData
	if err := dec.Decode(&dst); err != nil {
		return nil, fmt.Errorf("failed to decode for deep copy: %w", err)
	}
	return dst, nil
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

// PrintConfig displays the button configuration with Profile -> Menu -> Button hierarchy.
func PrintConfig(config ConfigData) { // Expects NEW ConfigData definition
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
	for id := range config {
		profileIDs = append(profileIDs, id)
	}
	sort.Strings(profileIDs) // Sort Profile IDs

	for i, profileID := range profileIDs {
		if i > 0 {
			sb.WriteString("----------------------------------------------------------------------\n") // Separator between profiles
		}
		menuConfig := config[profileID] // menuConfig is of type MenuConfig (map[string]ButtonMap)
		fmt.Fprintf(&sb, "Profile: %s\n", profileID)

		if len(menuConfig) == 0 {
			sb.WriteString("  (No menus configured for this profile)\n")
			continue
		}

		// --- Iterate Menus within Profile ---
		menuIDs := make([]string, 0, len(menuConfig))
		for id := range menuConfig {
			menuIDs = append(menuIDs, id)
		}
		sort.Strings(menuIDs) // Sort Menu IDs

		for j, menuID := range menuIDs {
			if j > 0 {
				sb.WriteString("  ---\n") // Separator between menus
			}
			buttonMap := menuConfig[menuID]          // buttonMap is of type ButtonMap (map[string]Task)
			fmt.Fprintf(&sb, "  Menu: %s\n", menuID) // Indent Menu ID

			if len(buttonMap) == 0 {
				sb.WriteString("    (No buttons configured for this menu)\n")
				continue
			}

			// --- Iterate Buttons within Menu ---
			buttonIDs := make([]int, 0, len(buttonMap))
			for idStr := range buttonMap {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					// Log with Profile and Menu context
					log.Printf("WARN: Invalid button ID format '%s' in Profile '%s', Menu '%s'", idStr, profileID, menuID)
					continue
				}
				buttonIDs = append(buttonIDs, id)
			}
			sort.Ints(buttonIDs) // Sort Button IDs

			for _, buttonID := range buttonIDs {
				buttonIDStr := strconv.Itoa(buttonID)
				task := buttonMap[buttonIDStr] // Get the specific task

				// Indent Button info further
				fmt.Fprintf(&sb, "    Btn %2d: [%-20s] ", buttonID, task.TaskType)

				taskSpecificDetails := ""
				// Switch logic remains the same, but logging includes profile/menu IDs
				switch TaskType(task.TaskType) {
				case TaskTypeShowAnyWindow:
					props, err := GetTaskProperties[ShowAnyWindowProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for ShowAnyWindow (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
						taskSpecificDetails = "<Error reading props>"
					} else {
						taskSpecificDetails = formatProperties(
							fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
							fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
							condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", props.ExePath)),
							// Use constant for clarity, check against default/invalid value
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
							fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
							fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
							condStr(props.ExePath != "", fmt.Sprintf("Exe: '%s'", props.ExePath)),
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
							fmt.Sprintf("Upper: '%s'", props.ButtonTextUpper),
							fmt.Sprintf("Lower: '%s'", props.ButtonTextLower),
							condStr(props.IconPath != "", fmt.Sprintf("Icon: '%s'", props.IconPath)),
						)
					}
				case TaskTypeLaunchProgram:
					props, err := GetTaskProperties[LaunchProgramProperties](task)
					if err != nil {
						log.Printf("ERROR: Failed to get props for LaunchProgram (P:%s M:%s B:%s) - %v", profileID, menuID, buttonIDStr, err)
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
			} // End Button Loop
		} // End Menu Loop
	} // End Profile Loop

	sb.WriteString("======================================================================\n")
	fmt.Print(sb.String()) // Print the complete string
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
