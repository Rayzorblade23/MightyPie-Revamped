package buttonManagerAdapter

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"
)

// Helper type for Step E
type availableSlotInfo struct {
	ProfileID string
	MenuID    string
	ButtonID  string
	// Store numeric IDs for easy sorting
	ProfileIdx int
	MenuIdx    int
	ButtonIdx  int
}

// Helper type for Step E
type availableWindowInfo struct {
    Handle int
    Info WindowInfo
}

// processExistingShowProgramHandles: ONLY checks/updates/clears existing handles. Does NOT assign new ones.
func (a *ButtonManagerAdapter) processExistingShowProgramHandles(
	profileID, menuID string,
	showProgramButtons map[string]*Task, // Pointers to tasks in buttonMap
	availableWindows WindowsUpdate, // Shared map, gets modified
	processedButtons map[string]bool, // Shared map, gets modified
	buttonMap ButtonMap, // The map from updatedConfig to modify
) {
	log.Printf("DEBUG: processExistingShowProgramHandles - Starting for P:%s M:%s", profileID, menuID)
	for btnID, taskPtr := range showProgramButtons { // Use pointer from separated map
        taskCopy := *taskPtr // Work on a copy
		buttonKey := fmt.Sprintf("%s:%s:%s", profileID, menuID, btnID)
        if processedButtons[buttonKey] { continue } // Skip if already handled (e.g., by ShowAny if ID reused)

		props, err := GetTaskProperties[ShowProgramWindowProperties](taskCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed to get ShowProgram props: %v", buttonKey, err)
			continue
		}

		taskModified := false
		if props.WindowHandle != InvalidHandle {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				// Handle Found & Valid
				if winInfo.ExePath == props.ExePath { // Path match required
					log.Printf("DEBUG: [%s] Found valid existing handle %d for %s.", buttonKey, props.WindowHandle, props.ExePath)
					originalTask := taskCopy // Copy before update
					if err := updateButtonWithWindowInfo(&taskCopy, winInfo, props.WindowHandle); err != nil {
						log.Printf("ERROR: [%s] Failed update ShowProgram task: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalTask, taskCopy) {
						taskModified = true
					}
					delete(availableWindows, props.WindowHandle)
					processedButtons[buttonKey] = true
				} else {
					// Handle Found but ExePath mismatch - Clear
					log.Printf("DEBUG: [%s] Handle %d ExePath '%s' mismatch expected '%s'. Clearing.", buttonKey, props.WindowHandle, winInfo.ExePath, props.ExePath)
					originalTask := taskCopy // Copy before clear
					if err := clearButtonWindowProperties(&taskCopy); err != nil {
						log.Printf("ERROR: [%s] Failed clear ShowProgram on mismatch: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalTask, taskCopy) {
						taskModified = true
					}
				}
			} else {
				// Handle Not Found (Invalid/Closed) - Clear
				log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle)
				originalTask := taskCopy // Copy before clear
				if err := clearButtonWindowProperties(&taskCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowProgram on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
			}
		} else {
            // Handle was already InvalidHandle - Ensure clear just in case
            originalTask := taskCopy
            if err := clearButtonWindowProperties(&taskCopy); err == nil && !reflect.DeepEqual(originalTask, taskCopy) {
                taskModified = true
            }
        }

		// Write back modified copy if necessary
		if taskModified {
			log.Printf("DEBUG: [%s] ShowProgram Task modified, updating buttonMap.", buttonKey)
			buttonMap[btnID] = taskCopy
		}
	}
}


// processExistingShowAnyHandles: ONLY checks/updates/clears existing handles. Does NOT assign new ones.
func (a *ButtonManagerAdapter) processExistingShowAnyHandles(
	profileID, menuID string,
	showAnyButtons map[string]*Task, // Pointers to tasks in buttonMap
	availableWindows WindowsUpdate, // Shared map, gets modified
	processedButtons map[string]bool, // Shared map, gets modified
	buttonMap ButtonMap, // The map from updatedConfig to modify
) {
	log.Printf("DEBUG: processExistingShowAnyHandles - Starting Step D logic for P:%s M:%s", profileID, menuID)
	for btnID, taskPtr := range showAnyButtons { // Use pointer from separated map
        taskCopy := *taskPtr // Work on a copy
		buttonKey := fmt.Sprintf("%s:%s:%s", profileID, menuID, btnID)
		if processedButtons[buttonKey] { continue }

		props, err := GetTaskProperties[ShowAnyWindowProperties](taskCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed get ShowAny props: %v", buttonKey, err)
			continue
		}

		taskModified := false
		if props.WindowHandle != InvalidHandle {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				// Handle Found & Valid: Update Task
				log.Printf("DEBUG: [%s] Found valid existing handle %d.", buttonKey, props.WindowHandle)
				originalTask := taskCopy
				if err := updateButtonWithWindowInfo(&taskCopy, winInfo, props.WindowHandle); err != nil {
					log.Printf("ERROR: [%s] Failed update ShowAny task: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
				delete(availableWindows, props.WindowHandle)
				processedButtons[buttonKey] = true
			} else {
				// Handle Found but Invalid/Closed: Clear properties
				log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle)
				originalTask := taskCopy
				if err := clearButtonWindowProperties(&taskCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowAny on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
			}
		} else {
            // Handle was already InvalidHandle - Ensure clear
             originalTask := taskCopy
             if err := clearButtonWindowProperties(&taskCopy); err == nil && !reflect.DeepEqual(originalTask, taskCopy) {
                 taskModified = true
             }
        }

		// Write back modified copy if necessary
		if taskModified {
			log.Printf("DEBUG: [%s] ShowAny Task modified, updating buttonMap.", buttonKey)
			buttonMap[btnID] = taskCopy
		}
	}
}

// assignMatchingProgramWindows: Assigns remaining windows to ShowProgramWindow tasks if ExePath matches.
// Runs AFTER phase 1 (existing handle processing) and BEFORE phase 2 (assign remaining any).
func (a *ButtonManagerAdapter) assignMatchingProgramWindows(
    availableWindows WindowsUpdate, // Remaining windows after phase 1
    processedButtons map[string]bool, // Buttons already handled
    fullUpdatedConfig ConfigData,    // The config potentially modified by Phase 1
) {
	log.Println("DEBUG: assignMatchingProgramWindows - Starting assignment based on ExePath match.")
	if len(availableWindows) == 0 {
        log.Println("DEBUG: assignMatchingProgramWindows - No windows available to assign.")
		return // Nothing to assign
	}

    windowsConsumed := make(map[int]bool) // Track windows assigned in this function

	// Iterate through all profiles, menus, buttons looking for ShowProgramWindow tasks
	for pID, mConfig := range fullUpdatedConfig {
		if mConfig == nil { continue }
		for mID, bMap := range mConfig {
			if bMap == nil { continue }
			for bID, task := range bMap { // Iterate task copies
				buttonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)

				// Skip if not a ShowProgramWindow task or already processed
				if TaskType(task.TaskType) != TaskTypeShowProgramWindow || processedButtons[buttonKey] {
					continue
				}

				props, err := GetTaskProperties[ShowProgramWindowProperties](task)
				if err != nil {
					log.Printf("WARN: [%s] assignMatchingProgramWindows - Failed to get props: %v", buttonKey, err)
					continue
				}

                // Task needs a window (handle is invalid) and has a specific ExePath configured
				if props.WindowHandle == InvalidHandle && props.ExePath != "" {
                    // Search availableWindows for a match
                    foundHandle := -1
                    var foundWinInfo WindowInfo
                    for handle, winInfo := range availableWindows {
                        // Check if window is already consumed in this pass and if ExePath matches
                        if !windowsConsumed[handle] && winInfo.ExePath == props.ExePath {
                            foundHandle = handle
                            foundWinInfo = winInfo
                            break // Found the first match
                        }
                    }

                    // If a matching window was found
                    if foundHandle != InvalidHandle {
                        log.Printf("DEBUG: [%s] assignMatchingProgramWindows - Found matching window H:%d for ExePath '%s'. Assigning.", buttonKey, foundHandle, props.ExePath)

                        // Get the actual task struct to modify
                        targetButtonMap := fullUpdatedConfig[pID][mID]
                        taskToModify := targetButtonMap[bID] // Get copy
                        originalTask := taskToModify // Store original

                        // Attempt to update the task copy
                        err := updateButtonWithWindowInfo(&taskToModify, foundWinInfo, foundHandle)
                        if err != nil {
                             log.Printf("ERROR: [%s] assignMatchingProgramWindows - Failed update task: %v", buttonKey, err)
                             // Continue searching for other tasks, maybe another button can use this window? No, proceed.
                        } else {
                            // Write the modified task copy back to the map
                            if !reflect.DeepEqual(originalTask, taskToModify) {
                                log.Printf("DEBUG: [%s] Task updated by assignment, writing back.", buttonKey)
                                targetButtonMap[bID] = taskToModify
                            } else {
                                log.Printf("DEBUG: [%s] Task update resulted in no change.", buttonKey)
                            }

                            // Mark button as processed and window as consumed for this function pass
                            processedButtons[buttonKey] = true
                            windowsConsumed[foundHandle] = true
                        }
                    }
				} // End if needs window and has ExePath
			} // End button loop
		} // End menu loop
	} // End profile loop

    // Remove windows consumed by this function from the main availableWindows map
    if len(windowsConsumed) > 0 {
         log.Printf("DEBUG: assignMatchingProgramWindows - Consumed %d windows.", len(windowsConsumed))
         for handle := range windowsConsumed {
             delete(availableWindows, handle)
         }
    } else {
         log.Println("DEBUG: assignMatchingProgramWindows - Consumed 0 windows.")
    }
     log.Println("DEBUG: assignMatchingProgramWindows - Finished.")
} // End assignMatchingProgramWindows

// assignRemainingWindows: New function to assign leftover windows to lowest P/M/B slots.
// Runs ONCE after Phase 1 processing. Modifies updatedConfig directly.
func (a *ButtonManagerAdapter) assignRemainingWindows(
    availableWindows WindowsUpdate, // The final state after Phase 1
    processedButtons map[string]bool, // The final state after Phase 1
    fullUpdatedConfig ConfigData,    // The config potentially modified by Phase 1
) {
    log.Printf("DEBUG: assignRemainingWindows - Starting. Windows to assign: %d", len(availableWindows))
	if len(availableWindows) == 0 {
        log.Println("DEBUG: assignRemainingWindows - No windows left to assign.")
		return // Nothing to do
	}

	// 1. Gather ALL available ShowAnyWindow slots from the *entire* updatedConfig
	var availableSlots []availableSlotInfo // Use helper struct defined earlier
	for pID, mConfig := range fullUpdatedConfig {
		pIdx, errP := strconv.Atoi(pID); if errP != nil { continue }
		if mConfig == nil { continue }
		for mID, bMap := range mConfig {
			mIdx, errM := strconv.Atoi(mID); if errM != nil { continue }
			if bMap == nil { continue }
			for bID, task := range bMap { // Iterate task copies
				bIdx, errB := strconv.Atoi(bID); if errB != nil { continue }

				if TaskType(task.TaskType) != TaskTypeShowAnyWindow { continue }

				currentButtonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if processedButtons[currentButtonKey] { continue } // Skip already processed

				props, err := GetTaskProperties[ShowAnyWindowProperties](task)
				if err != nil || props.WindowHandle != InvalidHandle { continue } // Skip if props fail or already has handle

				// Slot is available
				availableSlots = append(availableSlots, availableSlotInfo{
					ProfileID: pID, MenuID: mID, ButtonID: bID,
					ProfileIdx: pIdx, MenuIdx: mIdx, ButtonIdx: bIdx,
				})
			}
		}
	}

	if len(availableSlots) == 0 {
        log.Println("DEBUG: assignRemainingWindows - No available ShowAny slots found.")
		return // No slots to assign to
	}

	// 2. Sort available slots numerically by Profile, Menu, Button
	sort.SliceStable(availableSlots, func(i, j int) bool {
		// ... (sorting logic as implemented before) ...
		if availableSlots[i].ProfileIdx != availableSlots[j].ProfileIdx {
			return availableSlots[i].ProfileIdx < availableSlots[j].ProfileIdx
		}
		if availableSlots[i].MenuIdx != availableSlots[j].MenuIdx {
			return availableSlots[i].MenuIdx < availableSlots[j].MenuIdx
		}
		return availableSlots[i].ButtonIdx < availableSlots[j].ButtonIdx
	})

	// 3. Gather remaining available windows
	var windowsToAssign []availableWindowInfo // Use helper struct defined earlier
	for handle, info := range availableWindows {
		windowsToAssign = append(windowsToAssign, availableWindowInfo{Handle: handle, Info: info})
	}
	// Optional: Sort windows for deterministic assignment (e.g., by HWND)
	sort.Slice(windowsToAssign, func(i, j int) bool {
		return windowsToAssign[i].Handle < windowsToAssign[j].Handle
	})

	// 4. Assign windows sequentially to sorted slots
	log.Printf("DEBUG: assignRemainingWindows - Assigning %d windows to %d slots.", len(windowsToAssign), len(availableSlots))
	assignedCount := 0
	windowsConsumed := make(map[int]bool)

	for i := 0; i < len(availableSlots) && assignedCount < len(windowsToAssign); i++ {
		slot := availableSlots[i]
		window := windowsToAssign[assignedCount] // Get next window
        slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.ProfileID, slot.MenuID, slot.ButtonID)

        // Get the actual map and task struct to modify
        targetButtonMap := fullUpdatedConfig[slot.ProfileID][slot.MenuID]
        taskToModify := targetButtonMap[slot.ButtonID] // Get task copy
        originalTask := taskToModify // Store original state

		log.Printf("DEBUG: [%s] Assigning window '%s' (H:%d)", slotButtonKey, window.Info.Title, window.Handle)

		err := updateButtonWithWindowInfo(&taskToModify, window.Info, window.Handle) // Modify the copy
		if err != nil {
			log.Printf("ERROR: [%s] Failed update task with assigned window: %v", slotButtonKey, err)
            continue // Try next slot (window remains unassigned for now)
		}

        // Write the modified task copy back to the map
        if !reflect.DeepEqual(originalTask, taskToModify) {
            log.Printf("DEBUG: [%s] Task updated by assignment, writing back.", slotButtonKey)
		    targetButtonMap[slot.ButtonID] = taskToModify
        } else {
            log.Printf("DEBUG: [%s] Task update assignment resulted in no change.", slotButtonKey)
        }

		// Mark window consumed from availableWindows and button as processed
        windowsConsumed[window.Handle] = true
		processedButtons[slotButtonKey] = true // Mark button as processed globally
		assignedCount++                      // Move to next window
	}

	// Remove windows consumed in this step from the original availableWindows map
	// Although technically availableWindows isn't used after this, it's good practice
	for handle := range windowsConsumed {
        delete(availableWindows, handle)
    }

	log.Printf("DEBUG: assignRemainingWindows - Assigned %d windows.", assignedCount)

	// 5. Clear remaining available slots that didn't get a window
	//    (This step ensures consistency - previously assigned windows that closed won't linger)
	if assignedCount < len(availableSlots) {
        log.Printf("DEBUG: Clearing %d remaining empty ShowAny slots.", len(availableSlots)-assignedCount)
		for i := assignedCount; i < len(availableSlots); i++ {
			 slot := availableSlots[i]
			 slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.ProfileID, slot.MenuID, slot.ButtonID)
			 targetButtonMap := fullUpdatedConfig[slot.ProfileID][slot.MenuID]
			 taskToModify := targetButtonMap[slot.ButtonID]
             originalTask := taskToModify

			 err := clearButtonWindowProperties(&taskToModify)
			 if err != nil {
				  log.Printf("ERROR: [%s] Failed to clear remaining empty slot: %v", slotButtonKey, err)
			 } else if !reflect.DeepEqual(originalTask, taskToModify) {
				  log.Printf("DEBUG: [%s] Cleared remaining empty slot.", slotButtonKey)
                  targetButtonMap[slot.ButtonID] = taskToModify // Write cleared copy back
			 }
		}
	}
     log.Println("DEBUG: assignRemainingWindows - Finished.")
} // End assignRemainingWindows


// processLaunchProgramTasks handles updates for LaunchProgram tasks.
// Modifies tasks pointed to by launchProgramButtons and updates buttonMap accordingly.
func (a *ButtonManagerAdapter) processLaunchProgramTasks(profileID, menuID string, launchProgramButtons map[string]*Task, buttonMap ButtonMap) {
	for btnID, taskPtr := range launchProgramButtons {
		// props, err := GetTaskProperties[LaunchProgramProperties](*taskPtr)
		// if err != nil {
		//  log.Printf("WARN: Failed get Launch props (Profile:%s Menu:%s Button:%s): %v", profileID, menuID, btnID, err)
		//  continue
		// }
		buttonMap[btnID] = *taskPtr // Ensure the (potentially modified) task is in the main map
	}
}

// processFunctionCallTasks handles updates for CallFunction tasks.
// Now takes originalButtonMap for more accurate preservation.
func (a *ButtonManagerAdapter) processFunctionCallTasks(
	profileID, menuID string,
	functionCallButtons map[string]*Task,
	buttonMap ButtonMap,
	originalMenuButtonMap ButtonMap, // Pass the specific menu's original button map
) {
	for btnID, taskPtrCurrent := range functionCallButtons {
		// If the original config had this button, prefer its state for function calls
		// as they typically don't change based on window updates.
		if originalTask, exists := originalMenuButtonMap[btnID]; exists && TaskType(originalTask.TaskType) == TaskTypeCallFunction {
			buttonMap[btnID] = originalTask // Restore from original snapshot
			log.Printf("DEBUG: [Profile:%s Menu:%s Button:%s] Restored CallFunction task from original config snapshot.", profileID, menuID, btnID)
		} else {
			// If not in original or type mismatch (shouldn't happen if separateTasksByType is correct),
			// keep the current task from the copied config.
			buttonMap[btnID] = *taskPtrCurrent
			log.Printf("DEBUG: [Profile:%s Menu:%s Button:%s] Kept current CallFunction task (not found or type mismatch in original).", profileID, menuID, btnID)

		}
	}
}


// -------- Helpers -----------

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
