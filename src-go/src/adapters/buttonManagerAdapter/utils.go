package buttonManagerAdapter

import (
	"fmt"
	"sort"
)

// ------- Printing Functions -------

// // PrintWindowList prints the current window list for debugging
func PrintWindowList(mapping map[int]WindowInfo_Message) {
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

func PrintConfig(config ConfigData) {
	fmt.Println("================== Configuration ==================")
	if len(config) == 0 {
		fmt.Println("(empty configuration)")
		return
	}

	// Sort menus for consistent output
	menuIndices := make([]string, 0, len(config))
	for menuIndex := range config {
		menuIndices = append(menuIndices, menuIndex)
	}
	sort.Strings(menuIndices)

	for _, menuIndex := range menuIndices {
		fmt.Printf("\nMenu %s:\n", menuIndex)
		menu := config[menuIndex]

		// Sort buttons for consistent output
		buttonIndices := make([]string, 0, len(menu))
		for buttonIndex := range menu {
			buttonIndices = append(buttonIndices, buttonIndex)
		}
		sort.Strings(buttonIndices)

		for _, buttonIndex := range buttonIndices {
			fmt.Printf("\n  Button %s:\n", buttonIndex)
			fmt.Printf("  -------------------\n")
			PrintTask(menu[buttonIndex])
		}
	}
	fmt.Println("================================================")
}
