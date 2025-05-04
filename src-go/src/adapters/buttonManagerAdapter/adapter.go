package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"

	"maps"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
)

var (
	buttonConfig ConfigData
	windowsList  WindowsUpdate
	mu           sync.RWMutex
)

type ButtonManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ButtonManagerAdapter {
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	config, err := ReadButtonConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Store config at package level
	buttonConfig = config
	PrintConfig(config)

	buttonUpdateSubject := env.Get("NATSSUBJECT_BUTTONMANAGER_UPDATE") // Assuming this env var exists

	a.natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_WINDOWMANAGER_UPDATE"), func(msg *nats.Msg) {
		var currentWindows WindowsUpdate // Use new type name

		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Printf("ERROR: Failed to decode window update message: %v", err)
			return
		}

		// --- Update internal state ---
		mu.Lock()
		// Make a copy to avoid race conditions if used elsewhere immediately
		// Although current GetCurrentWindowsList returns a copy, it's safer here too.
		windowsList = make(WindowsUpdate, len(currentWindows))
		maps.Copy(windowsList, currentWindows)
		mu.Unlock()

		// --- Process the update and generate new button config ---
		currentConfig := GetButtonConfig() // Read the immutable config
		updatedConfig, err := a.processWindowUpdate(currentConfig, currentWindows)
		if err != nil {
			log.Printf("ERROR: Failed to process window update for button config: %v", err)
			return
		}

		PrintConfig(updatedConfig) // Print the updated config for debugging

		// --- Publish the updated configuration ---
		if updatedConfig != nil { // Only publish if changes were made (processWindowUpdate can return nil)
			updatedConfigJSON, err := json.Marshal(updatedConfig)
			if err != nil {
				log.Printf("ERROR: Failed to marshal updated button config: %v", err)
				return
			}

			a.natsAdapter.PublishMessage(buttonUpdateSubject, updatedConfigJSON)

		}
	})

	return a
}


// processWindowUpdate takes the current config and window list, returning an updated config.
// Returns nil, nil if no effective changes were made to the configuration.
func (a *ButtonManagerAdapter) processWindowUpdate(currentConfig ConfigData, windows WindowsUpdate) (ConfigData, error) {
	if len(currentConfig) == 0 {
		log.Println("DEBUG: Skipping button processing - no config loaded.")
		return nil, nil // No config, no changes possible
	}

	// 1. Deep Copy Config
	updatedConfig, err := deepCopyConfig(currentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy config: %w", err)
	}

	// 2. Handle Empty Window List Case
	if len(windows) == 0 {
		return a.handleEmptyWindowList(updatedConfig)
	}

	// 3. Setup for Processing
	availableWindows := make(WindowsUpdate, len(windows))
	maps.Copy(availableWindows, windows)
	processedButtons := make(map[string]bool) // Key: "menuID:buttonID"

	// 4. Process Each Profile
	for menuID, buttonMap := range updatedConfig {
		// Separate tasks by type for focused processing
		showProgramButtons, showAnyButtons, launchProgramButtons := a.separateTasksByType(buttonMap)

		// Process each type, modifying tasks and availableWindows/processedButtons directly
		a.processLaunchProgramTasks(menuID, launchProgramButtons, buttonMap)
		a.processShowProgramTasks(menuID, showProgramButtons, availableWindows, processedButtons, buttonMap)
		a.processShowAnyTasks(menuID, showAnyButtons, availableWindows, processedButtons, buttonMap)
	}

	// 5. Final Comparison
	if reflect.DeepEqual(currentConfig, updatedConfig) {
		log.Println("DEBUG: Button configuration unchanged after processing window update.")
		return nil, nil // No effective change, return nil
	}

	log.Println("DEBUG: Button configuration updated based on window changes.")
	return updatedConfig, nil
}

// handleEmptyWindowList clears window-related properties from tasks when the window list is empty.
// It modifies the provided config directly and returns it along with nil error if changes were made.
// Returns nil, nil if no changes were needed.
func (a *ButtonManagerAdapter) handleEmptyWindowList(config ConfigData) (ConfigData, error) {
	log.Println("DEBUG: Window list is empty. Clearing existing handles in button config.")
	changed := false
	for menuID, buttonMap := range config {
		for btnID, task := range buttonMap {
			// Create a copy to modify, then potentially assign back
			taskCopy := task
			err := clearButtonWindowProperties(&taskCopy) // Use existing helper
			if err != nil {
				// Log error but continue clearing others
				log.Printf("ERROR: Failed to clear properties for task (%s:%s) on empty window list: %v", menuID, btnID, err)
			} else if !reflect.DeepEqual(task, taskCopy) {
				// Only update map and flag changed if clearing actually modified the task
				buttonMap[btnID] = taskCopy
				changed = true
			}
		}
	}
	if changed {
		log.Printf("DEBUG: Cleared handles in config due to empty window list.")
		return config, nil // Return the modified config
	}

	log.Printf("DEBUG: No handles needed clearing on empty window list.")
	return nil, nil // No effective changes
}

// separateTasksByType classifies tasks in a button map by their type.
func (a *ButtonManagerAdapter) separateTasksByType(buttonMap ButtonMap) (
	showProgram map[string]*Task, showAny map[string]*Task, launchProgram map[string]*Task) {

	showProgram = make(map[string]*Task)
	showAny = make(map[string]*Task)
	launchProgram = make(map[string]*Task)

	for btnID, task := range buttonMap {
		// Create a pointer to the task *copy* for modification within processing funcs
		// The original task in buttonMap will be replaced if modified.
		taskPtr := new(Task)
		*taskPtr = task // Copy value

		switch TaskType(taskPtr.TaskType) {
		case TaskTypeShowProgramWindow:
			showProgram[btnID] = taskPtr
		case TaskTypeShowAnyWindow:
			showAny[btnID] = taskPtr
		case TaskTypeLaunchProgram:
			launchProgram[btnID] = taskPtr
			// TaskTypeCallFunction, TaskTypeDisabled: ignored for window processing
		}
	}
	return
}

// processLaunchProgramTasks handles updates for LaunchProgram tasks (e.g., fetching icons).
// Modifies tasks pointed to by launchProgramButtons and updates buttonMap accordingly.
func (a *ButtonManagerAdapter) processLaunchProgramTasks(menuID string, launchProgramButtons map[string]*Task, buttonMap ButtonMap) {
	// A. Update Launch Program Buttons (Optional: Fill missing info like IconPath)
	for btnID, taskPtr := range launchProgramButtons {
		// --- Example Placeholder: Icon Fetching Logic ---
		// props, err := GetTaskProperties[LaunchProgramProperties](*taskPtr)
		// if err != nil {
		//  log.Printf("WARN: Failed get Launch props (%s:%s): %v", menuID, btnID, err)
		//  continue
		// }
		// originalIcon := props.IconPath
		// if props.IconPath == "" {
		//    // Hypothetical function to look up icon based on exe path
		//    // cachedIcon := a.appCache.GetIcon(props.ExePath)
		//    cachedIcon := "" // Replace with actual cache lookup if implemented
		//    if cachedIcon != "" {
		//      props.IconPath = cachedIcon
		//      if err := SetTaskProperties(taskPtr, props); err != nil {
		//          log.Printf("ERROR: Failed set updated Launch props (%s:%s): %v", menuID, btnID, err)
		//          // Optional: revert icon path on error? props.IconPath = originalIcon
		//      }
		//    }
		// }
		// --- End Placeholder ---

		// Update the main map ONLY if the task was potentially modified
		// For now, we always update as the placeholder doesn't track changes.
		// If icon logic is added, only update if SetTaskProperties was called successfully.
		buttonMap[btnID] = *taskPtr
	}
}

// processShowProgramTasks handles updates for ShowProgramWindow tasks (existing handles and assignment).
// Modifies tasks, availableWindows, processedButtons, and updates buttonMap.
func (a *ButtonManagerAdapter) processShowProgramTasks(
	menuID string,
	showProgramButtons map[string]*Task,
	availableWindows WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap ButtonMap,
) {
	// B. Update Show Program Window - Step 1: Check existing handles
	for btnID, taskPtr := range showProgramButtons {
		buttonKey := menuID + ":" + btnID
		props, err := GetTaskProperties[ShowProgramWindowProperties](*taskPtr)
		if err != nil {
			log.Printf("WARN: Failed to get ShowProgram props (%s): %v", buttonKey, err)
			buttonMap[btnID] = *taskPtr // Ensure map has the (unmodified) task
			continue
		}

		taskModified := false // Track if task is changed in this iteration

		if props.WindowHandle != -1 {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				if winInfo.ExePath == props.ExePath {
					log.Printf("DEBUG: [%s] Found existing valid handle %d for %s", buttonKey, props.WindowHandle, props.ExePath)
					if err := updateButtonWithWindowInfo(taskPtr, winInfo, props.WindowHandle); err != nil {
						log.Printf("ERROR: [%s] Failed to update task with window info: %v", buttonKey, err)
					} else {
						delete(availableWindows, props.WindowHandle)
						processedButtons[buttonKey] = true
						taskModified = true
					}
				} else {
					log.Printf("DEBUG: [%s] Handle %d (%s) mismatches ExePath %s. Clearing.", buttonKey, props.WindowHandle, winInfo.ExePath, props.ExePath)
					originalTask := *taskPtr
					if err := clearButtonWindowProperties(taskPtr); err != nil {
						log.Printf("ERROR: [%s] Failed to clear properties after mismatch: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalTask, *taskPtr) {
						taskModified = true
					}
				}
			} else {
				log.Printf("DEBUG: [%s] Existing handle %d is invalid. Clearing.", buttonKey, props.WindowHandle)
				originalTask := *taskPtr
				if err := clearButtonWindowProperties(taskPtr); err != nil {
					log.Printf("ERROR: [%s] Failed to clear properties for invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, *taskPtr) {
					taskModified = true
				}
			}
		}
		// Update map only if the task was modified in this step
		if taskModified {
			buttonMap[btnID] = *taskPtr
		} else if _, ok := buttonMap[btnID]; !ok { // Ensure task exists if unmodified
            buttonMap[btnID] = *taskPtr
        }
	}

	// C. Update Show Program Window - Step 2: Assign free windows matching ExePath
	for btnID, taskPtr := range showProgramButtons {
		buttonKey := menuID + ":" + btnID
		if processedButtons[buttonKey] {
			continue // Already processed
		}

		props, err := GetTaskProperties[ShowProgramWindowProperties](*taskPtr)
		if err != nil { // Should have been logged before if failed during handle check
			continue
		}

		// Find matching window
		foundHandle, foundWinInfo := a.findMatchingWindow(availableWindows, props.ExePath)

		taskModified := false
		if foundHandle != -1 {
			log.Printf("DEBUG: [%s] Assigning free handle %d for %s", buttonKey, foundHandle, props.ExePath)
			if err := updateButtonWithWindowInfo(taskPtr, foundWinInfo, foundHandle); err != nil {
				log.Printf("ERROR: [%s] Failed to update task with assigned window info: %v", buttonKey, err)
			} else {
				delete(availableWindows, foundHandle)
				processedButtons[buttonKey] = true
				taskModified = true
			}
		} else {
			// No matching window found. Ensure properties are cleared.
			// log.Printf("DEBUG: [%s] No available window found for %s. Ensuring clear.", buttonKey, props.ExePath)
			originalTask := *taskPtr
			if err := clearButtonWindowProperties(taskPtr); err != nil {
				log.Printf("ERROR: [%s] Failed to clear properties when no window found: %v", buttonKey, err)
			} else if !reflect.DeepEqual(originalTask, *taskPtr) {
				taskModified = true
			}
		}
		// Update map only if the task was modified in this step
		if taskModified {
			buttonMap[btnID] = *taskPtr
		}
	}
}

// findMatchingWindow searches availableWindows for a window with the specified exePath.
// Returns handle and info if found, otherwise -1 and zero WindowInfo.
func (a *ButtonManagerAdapter) findMatchingWindow(availableWindows WindowsUpdate, exePath string) (int, WindowInfo) {
	for handle, winInfo := range availableWindows {
		if winInfo.ExePath == exePath {
			return handle, winInfo
		}
	}
	return -1, WindowInfo{}
}

// processShowAnyTasks handles updates for ShowAnyWindow tasks (existing handles and assignment).
// Modifies tasks, availableWindows, processedButtons, and updates buttonMap.
func (a *ButtonManagerAdapter) processShowAnyTasks(
	menuID string,
	showAnyButtons map[string]*Task,
	availableWindows WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap ButtonMap,
) {
	// D. Update Show Any Window - Step 1: Check existing handles
	for btnID, taskPtr := range showAnyButtons {
		buttonKey := menuID + ":" + btnID
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetTaskProperties[ShowAnyWindowProperties](*taskPtr)
		if err != nil {
			log.Printf("WARN: Failed to get ShowAny props (%s): %v", buttonKey, err)
			buttonMap[btnID] = *taskPtr // Ensure map has the (unmodified) task
			continue
		}

		taskModified := false
		if props.WindowHandle != -1 {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				log.Printf("DEBUG: [%s] Found existing valid handle %d (any window)", buttonKey, props.WindowHandle)
				if err := updateButtonWithWindowInfo(taskPtr, winInfo, props.WindowHandle); err != nil {
					log.Printf("ERROR: [%s] Failed to update task with window info: %v", buttonKey, err)
				} else {
					delete(availableWindows, props.WindowHandle)
					processedButtons[buttonKey] = true
					taskModified = true
				}
			} else {
				log.Printf("DEBUG: [%s] Existing handle %d is invalid. Clearing.", buttonKey, props.WindowHandle)
				originalTask := *taskPtr
				if err := clearButtonWindowProperties(taskPtr); err != nil {
					log.Printf("ERROR: [%s] Failed to clear properties for invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, *taskPtr) {
					taskModified = true
				}
			}
		}
		if taskModified {
			buttonMap[btnID] = *taskPtr
		} else if _, ok := buttonMap[btnID]; !ok { // Ensure task exists if unmodified
            buttonMap[btnID] = *taskPtr
        }
	}

	// E. Update Show Any Window - Step 2: Assign remaining free windows
	handles := make([]int, 0, len(availableWindows))
	for h := range availableWindows {
		handles = append(handles, h)
	}
	sort.Ints(handles) // Sort for deterministic assignment

	for btnID, taskPtr := range showAnyButtons {
		buttonKey := menuID + ":" + btnID
		if processedButtons[buttonKey] {
			continue // Already has a window
		}

		taskModified := false
		if len(handles) > 0 && len(availableWindows) > 0 {
			assignedHandle := handles[0]
			assignedWinInfo := availableWindows[assignedHandle]
			handles = handles[1:] // Consume handle

			log.Printf("DEBUG: [%s] Assigning free handle %d (any window - %s)", buttonKey, assignedHandle, assignedWinInfo.Title)
			if err := updateButtonWithWindowInfo(taskPtr, assignedWinInfo, assignedHandle); err != nil {
				log.Printf("ERROR: [%s] Failed to update task with assigned window info: %v", buttonKey, err)
				// Should we put handle back in 'handles'? For now, just log.
			} else {
				delete(availableWindows, assignedHandle) // Mark window as used in the map
				processedButtons[buttonKey] = true
				taskModified = true
			}
		} else {
			// No windows left to assign. Ensure button is clear.
			// log.Printf("DEBUG: [%s] No more available windows to assign.", buttonKey)
			originalTask := *taskPtr
			if err := clearButtonWindowProperties(taskPtr); err != nil {
				log.Printf("ERROR: [%s] Failed to clear properties when no window available: %v", buttonKey, err)
			} else if !reflect.DeepEqual(originalTask, *taskPtr) {
				taskModified = true
			}
		}
		if taskModified {
			buttonMap[btnID] = *taskPtr
		}
	}
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

// updateButtonWithWindowInfo updates the properties of a task based on WindowInfo.
// It takes pointers to the specific property structs to modify them directly.
func updateButtonWithWindowInfo(task *Task, winInfo WindowInfo, newHandle int) error {
	switch TaskType(task.TaskType) {
	case TaskTypeShowProgramWindow:
		props, err := GetTaskProperties[ShowProgramWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("failed to get properties for %s: %w", task.TaskType, err)
		}
		props.WindowHandle = newHandle
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		// Set properties and check error immediately
		if err = SetTaskProperties(task, props); err != nil {
			return fmt.Errorf("failed to set updated properties for %s: %w", task.TaskType, err)
		}
		return nil // Success for this case

	case TaskTypeShowAnyWindow:
		props, err := GetTaskProperties[ShowAnyWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("failed to get properties for %s: %w", task.TaskType, err)
		}
		props.WindowHandle = newHandle
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		props.ExePath = winInfo.ExePath
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		// Set properties and check error immediately
		if err = SetTaskProperties(task, props); err != nil {
			return fmt.Errorf("failed to set updated properties for %s: %w", task.TaskType, err)
		}
		return nil // Success for this case

	// Add cases for other types IF they were ever dynamically updated by window info

	default:
		// No update needed for this task type based on window info
		return nil // Not an error, just no action
	}
	// No final error check needed here anymore
}

// clearButtonWindowProperties resets window-specific fields in a task's properties
// to their default/empty state.
func clearButtonWindowProperties(task *Task) error {
	switch TaskType(task.TaskType) {
	case TaskTypeShowProgramWindow:
		props, err := GetTaskProperties[ShowProgramWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("failed to get properties for %s: %w", task.TaskType, err)
		}
		// Check if already cleared BEFORE modifying props
		if props.WindowHandle == -1 && props.ButtonTextUpper == "" && props.ButtonTextLower == "" && props.IconPath == "" {
			return nil // Already cleared, no change needed
		}
		// Modify props
		props.WindowHandle = -1
		props.ButtonTextUpper = ""
		props.ButtonTextLower = ""
		props.IconPath = ""
		// Set properties and check error immediately
		if err = SetTaskProperties(task, props); err != nil {
			return fmt.Errorf("failed to set cleared properties for %s: %w", task.TaskType, err)
		}
		return nil // Success for this case

	case TaskTypeShowAnyWindow:
		props, err := GetTaskProperties[ShowAnyWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("failed to get properties for %s: %w", task.TaskType, err)
		}
		// Check if already cleared BEFORE modifying props
		if props.WindowHandle == -1 && props.ButtonTextUpper == "" && props.ButtonTextLower == "" && props.IconPath == "" && props.ExePath == "" {
			return nil // Already cleared
		}
		// Modify props
		props.WindowHandle = -1
		props.ButtonTextUpper = ""
		props.ButtonTextLower = ""
		props.IconPath = ""
		props.ExePath = ""
		// Set properties and check error immediately
		if err = SetTaskProperties(task, props); err != nil {
			return fmt.Errorf("failed to set cleared properties for %s: %w", task.TaskType, err)
		}
		return nil // Success for this case

	// Add cases for other types IF they ever need clearing based on window info loss

	default:
		// No clearing needed for this task type
		return nil // Not an error
	}
	// No final error check needed here anymore
}

func (a *ButtonManagerAdapter) Run() error {
	fmt.Println("ButtonManagerAdapter started")
	select {}
}
