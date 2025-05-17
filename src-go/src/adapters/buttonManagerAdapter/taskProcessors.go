package buttonManagerAdapter

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// --- Helper types assumed from types.go ---
// type availableSlotInfo struct { ... }
// type availableWindowInfo struct { ... }

// processExistingShowProgramHandles (Cleaned)
func (a *ButtonManagerAdapter) processExistingShowProgramHandles(
	menuID, pageID string,
	showProgramButtons map[string]*Task,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap PageConfig,
) {
	// log.Printf("DEBUG: processExistingShowProgramHandles - Starting for P:%s M:%s", menuID, pageID) // Removed DEBUG
	for btnID, taskPtr := range showProgramButtons {
		taskCopy := *taskPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetTaskProperties[ShowProgramWindowProperties](taskCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed to get ShowProgram props: %v", buttonKey, err)
			continue
		}

		taskModified := false
		if props.WindowHandle > 0 {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				if winInfo.ExePath == props.ExePath {
					// log.Printf("DEBUG: [%s] Found valid existing handle %d for %s.", buttonKey, props.WindowHandle, props.ExePath) // Removed DEBUG
					originalTask := taskCopy
					if err := updateButtonWithWindowInfo(&taskCopy, winInfo, props.WindowHandle); err != nil {
						log.Printf("ERROR: [%s] Failed update ShowProgram task: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalTask, taskCopy) {
						taskModified = true
					}
					delete(availableWindows, props.WindowHandle)
					processedButtons[buttonKey] = true
				} else {
					// log.Printf("DEBUG: [%s] Handle %d ExePath '%s' mismatch expected '%s'. Clearing.", buttonKey, props.WindowHandle, winInfo.ExePath, props.ExePath) // Removed DEBUG
					originalTask := taskCopy
					if err := clearButtonWindowProperties(&taskCopy); err != nil {
						log.Printf("ERROR: [%s] Failed clear ShowProgram on mismatch: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalTask, taskCopy) {
						taskModified = true
					}
				}
			} else {
				// log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalTask := taskCopy
				if err := clearButtonWindowProperties(&taskCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowProgram on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
			}
		} else {
			originalTask := taskCopy
			if err := clearButtonWindowProperties(&taskCopy); err == nil && !reflect.DeepEqual(originalTask, taskCopy) {
				taskModified = true
			}
		}

		if taskModified {
			// log.Printf("DEBUG: [%s] ShowProgram Task modified, updating buttonMap.", buttonKey) // Removed DEBUG
			buttonMap[btnID] = taskCopy
		}
	}
}

// processExistingShowAnyHandles (Cleaned)
func (a *ButtonManagerAdapter) processExistingShowAnyHandles(
	menuID, pageID string,
	showAnyButtons map[string]*Task,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap PageConfig,
) {
	// log.Printf("DEBUG: processExistingShowAnyHandles - Starting Step D logic for P:%s M:%s", menuID, pageID) // Removed DEBUG
	for btnID, taskPtr := range showAnyButtons {
		taskCopy := *taskPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetTaskProperties[ShowAnyWindowProperties](taskCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed get ShowAny props: %v", buttonKey, err)
			continue
		}

		taskModified := false
		if props.WindowHandle != InvalidHandle {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				// log.Printf("DEBUG: [%s] Found valid existing handle %d.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalTask := taskCopy
				if err := updateButtonWithWindowInfo(&taskCopy, winInfo, props.WindowHandle); err != nil {
					log.Printf("ERROR: [%s] Failed update ShowAny task: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
				delete(availableWindows, props.WindowHandle)
				processedButtons[buttonKey] = true
			} else {
				// log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalTask := taskCopy
				if err := clearButtonWindowProperties(&taskCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowAny on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalTask, taskCopy) {
					taskModified = true
				}
			}
		} else {
			originalTask := taskCopy
			if err := clearButtonWindowProperties(&taskCopy); err == nil && !reflect.DeepEqual(originalTask, taskCopy) {
				taskModified = true
			}
		}

		if taskModified {
			// log.Printf("DEBUG: [%s] ShowAny Task modified, updating buttonMap.", buttonKey) // Removed DEBUG
			buttonMap[btnID] = taskCopy
		}
	}
}

// assignMatchingProgramWindows (Cleaned)
func (a *ButtonManagerAdapter) assignMatchingProgramWindows(
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	fullUpdatedConfig ConfigData,
) {
	// log.Println("DEBUG: assignMatchingProgramWindows - Starting assignment based on ExePath match.") // Removed DEBUG
	if len(availableWindows) == 0 {
		// log.Println("DEBUG: assignMatchingProgramWindows - No windows available to assign.") // Removed DEBUG
		return
	}

	windowsConsumed := make(map[int]bool)

	for pID, mConfig := range fullUpdatedConfig {
		if mConfig == nil {
			continue
		}
		for mID, bMap := range mConfig {
			if bMap == nil {
				continue
			}
			for bID, task := range bMap {
				buttonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if TaskType(task.TaskType) != TaskTypeShowProgramWindow || processedButtons[buttonKey] {
					continue
				}

				props, err := GetTaskProperties[ShowProgramWindowProperties](task)
				if err != nil {
					log.Printf("WARN: [%s] assignMatchingProgramWindows - Failed to get props: %v", buttonKey, err)
					continue
				}

				if props.WindowHandle == InvalidHandle && props.ExePath != "" {
					foundHandle := -1
					var foundWinInfo core.WindowInfo
					for handle, winInfo := range availableWindows {
						if !windowsConsumed[handle] && winInfo.ExePath == props.ExePath {
							foundHandle = handle
							foundWinInfo = winInfo
							break
						}
					}

					if foundHandle != InvalidHandle {
						// log.Printf("DEBUG: [%s] assignMatchingProgramWindows - Found matching window H:%d for ExePath '%s'. Assigning.", buttonKey, foundHandle, props.ExePath) // Removed DEBUG
						targetButtonMap := fullUpdatedConfig[pID][mID]
						taskToModify := targetButtonMap[bID]
						originalTask := taskToModify

						err := updateButtonWithWindowInfo(&taskToModify, foundWinInfo, foundHandle)
						if err != nil {
							log.Printf("ERROR: [%s] assignMatchingProgramWindows - Failed update task: %v", buttonKey, err)
						} else {
							if !reflect.DeepEqual(originalTask, taskToModify) {
								// log.Printf("DEBUG: [%s] Task updated by assignment, writing back.", buttonKey) // Removed DEBUG
								targetButtonMap[bID] = taskToModify
							}
							// else { log.Printf("DEBUG: [%s] Task update resulted in no change.", buttonKey) } // Removed DEBUG

							processedButtons[buttonKey] = true
							windowsConsumed[foundHandle] = true
						}
					}
				}
			}
		}
	}

	if len(windowsConsumed) > 0 {
		// log.Printf("DEBUG: assignMatchingProgramWindows - Consumed %d windows.", len(windowsConsumed)) // Removed DEBUG
		for handle := range windowsConsumed {
			delete(availableWindows, handle)
		}
	}
	// else { log.Println("DEBUG: assignMatchingProgramWindows - Consumed 0 windows.") } // Removed DEBUG
	// log.Println("DEBUG: assignMatchingProgramWindows - Finished.") // Removed DEBUG
}

// assignRemainingWindows (Cleaned)
func (a *ButtonManagerAdapter) assignRemainingWindows(
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	fullUpdatedConfig ConfigData,
) {
	// log.Printf("DEBUG: assignRemainingWindows - Starting. Windows to assign: %d", len(availableWindows)) // Removed DEBUG
	if len(availableWindows) == 0 {
		// log.Println("DEBUG: assignRemainingWindows - No windows left to assign.") // Removed DEBUG
		return
	}

	var availableSlots []availableSlotInfo
	for pID, mConfig := range fullUpdatedConfig {
		pIdx, errP := strconv.Atoi(pID)
		if errP != nil {
			continue
		}
		if mConfig == nil {
			continue
		}
		for mID, bMap := range mConfig {
			mIdx, errM := strconv.Atoi(mID)
			if errM != nil {
				continue
			}
			if bMap == nil {
				continue
			}
			for bID, task := range bMap {
				bIdx, errB := strconv.Atoi(bID)
				if errB != nil {
					continue
				}
				if TaskType(task.TaskType) != TaskTypeShowAnyWindow {
					continue
				}
				currentButtonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if processedButtons[currentButtonKey] {
					continue
				}
				props, err := GetTaskProperties[ShowAnyWindowProperties](task)
				if err != nil || props.WindowHandle != InvalidHandle {
					continue
				}
				availableSlots = append(availableSlots, availableSlotInfo{
					MenuID: pID, PageID: mID, ButtonID: bID,
					MenuIdx: pIdx, PageIdx: mIdx, ButtonIdx: bIdx,
				})
			}
		}
	}

	if len(availableSlots) == 0 {
		// log.Println("DEBUG: assignRemainingWindows - No available ShowAny slots found.") // Removed DEBUG
		return
	}

	sort.SliceStable(availableSlots, func(i, j int) bool {
		if availableSlots[i].MenuIdx != availableSlots[j].MenuIdx {
			return availableSlots[i].MenuIdx < availableSlots[j].MenuIdx
		}
		if availableSlots[i].PageIdx != availableSlots[j].PageIdx {
			return availableSlots[i].PageIdx < availableSlots[j].PageIdx
		}
		return availableSlots[i].ButtonIdx < availableSlots[j].ButtonIdx
	})

	var windowsToAssign []availableWindowInfo
	for handle, info := range availableWindows {
		windowsToAssign = append(windowsToAssign, availableWindowInfo{Handle: handle, Info: info})
	}
	sort.Slice(windowsToAssign, func(i, j int) bool { return windowsToAssign[i].Handle < windowsToAssign[j].Handle })

	// log.Printf("DEBUG: assignRemainingWindows - Assigning %d windows to %d slots.", len(windowsToAssign), len(availableSlots)) // Removed DEBUG
	assignedCount := 0
	windowsConsumed := make(map[int]bool)

	for i := 0; i < len(availableSlots) && assignedCount < len(windowsToAssign); i++ {
		slot := availableSlots[i]
		window := windowsToAssign[assignedCount]
		slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.MenuID, slot.PageID, slot.ButtonID)

		targetButtonMap := fullUpdatedConfig[slot.MenuID][slot.PageID]
		taskToModify := targetButtonMap[slot.ButtonID]
		originalTask := taskToModify

		// log.Printf("DEBUG: [%s] Assigning window '%s' (H:%d)", slotButtonKey, window.Info.Title, window.Handle) // Removed DEBUG

		err := updateButtonWithWindowInfo(&taskToModify, window.Info, window.Handle)
		if err != nil {
			log.Printf("ERROR: [%s] Failed update task with assigned window: %v", slotButtonKey, err)
			continue
		}

		if !reflect.DeepEqual(originalTask, taskToModify) {
			// log.Printf("DEBUG: [%s] Task updated by assignment, writing back.", slotButtonKey) // Removed DEBUG
			targetButtonMap[slot.ButtonID] = taskToModify
		}
		// else { log.Printf("DEBUG: [%s] Task update assignment resulted in no change.", slotButtonKey) } // Removed DEBUG

		windowsConsumed[window.Handle] = true
		processedButtons[slotButtonKey] = true
		assignedCount++
	}

	for handle := range windowsConsumed {
		delete(availableWindows, handle)
	}
	// log.Printf("DEBUG: assignRemainingWindows - Assigned %d windows.", assignedCount) // Removed DEBUG

	// Clear remaining slots
	if assignedCount < len(availableSlots) {
		// log.Printf("DEBUG: Clearing %d remaining empty ShowAny slots.", len(availableSlots)-assignedCount) // Removed DEBUG
		for i := assignedCount; i < len(availableSlots); i++ {
			slot := availableSlots[i]
			slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.MenuID, slot.PageID, slot.ButtonID)
			targetButtonMap := fullUpdatedConfig[slot.MenuID][slot.PageID]
			taskToModify := targetButtonMap[slot.ButtonID]
			originalTask := taskToModify
			err := clearButtonWindowProperties(&taskToModify)
			if err != nil {
				log.Printf("ERROR: [%s] Failed to clear remaining empty slot: %v", slotButtonKey, err)
			} else if !reflect.DeepEqual(originalTask, taskToModify) {
				// log.Printf("DEBUG: [%s] Cleared remaining empty slot.", slotButtonKey) // Removed DEBUG
				targetButtonMap[slot.ButtonID] = taskToModify
			}
		}
	}
	// log.Println("DEBUG: assignRemainingWindows - Finished.") // Removed DEBUG
}

// processLaunchProgramTasks (Assuming no DEBUG logs added)
func (a *ButtonManagerAdapter) processLaunchProgramTasks(menuID, pageID string, launchProgramButtons map[string]*Task, buttonMap PageConfig) {
	for btnID, taskPtr := range launchProgramButtons {
		// No window-based updates typically needed, ensure task is present
		if _, ok := buttonMap[btnID]; !ok {
			buttonMap[btnID] = *taskPtr // Ensure original task state is preserved
		}
	}
}

// processFunctionCallTasks (Assuming no DEBUG logs added)
func (a *ButtonManagerAdapter) processFunctionCallTasks(menuID, pageID string, functionCallButtons map[string]*Task, buttonMap PageConfig, originalMenuButtonMap PageConfig) {
	for btnID, taskPtrCurrent := range functionCallButtons {
		if originalTask, exists := originalMenuButtonMap[btnID]; exists && TaskType(originalTask.TaskType) == TaskTypeCallFunction {
			buttonMap[btnID] = originalTask // Restore from original snapshot
		} else {
			// Keep current task if not found in original (shouldn't happen often)
			buttonMap[btnID] = *taskPtrCurrent
		}
	}
}

// updateButtonWithWindowInfo (Cleaned)
func updateButtonWithWindowInfo(task *Task, winInfo core.WindowInfo, newHandle int) error {
	// log.Printf("DEBUG: updateButtonWithWindowInfo called for task type %s with handle %d", task.TaskType, newHandle) // Removed DEBUG
	switch TaskType(task.TaskType) {
	case TaskTypeShowProgramWindow:
		props, err := GetTaskProperties[ShowProgramWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}
		props.WindowHandle = newHandle
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		return SetTaskProperties(task, props) // Returns error from SetTaskProperties
	case TaskTypeShowAnyWindow:
		props, err := GetTaskProperties[ShowAnyWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}
		props.WindowHandle = newHandle
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		props.ExePath = winInfo.ExePath // Update ExePath for ShowAny
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		return SetTaskProperties(task, props)
	}
	return nil // No update needed for other types
}

// clearButtonWindowProperties (Cleaned)
func clearButtonWindowProperties(task *Task) error {
	switch TaskType(task.TaskType) {
	case TaskTypeShowProgramWindow:
		props, err := GetTaskProperties[ShowProgramWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}

		// Preserve existing properties from buttonConfig
		exePath := props.ExePath
		buttonLower := props.ButtonTextLower
		iconPath := props.IconPath

		// Clear window-specific properties but maintain program identity
		props.WindowHandle = InvalidHandle // Use InvalidHandle (-1) for no window
		props.ButtonTextUpper = ""         // Clear window title

		// Ensure we keep program identity
		props.ExePath = exePath             // Keep the program path
		props.ButtonTextLower = buttonLower // Keep app name
		props.IconPath = iconPath           // Keep icon from buttonConfig

		return SetTaskProperties(task, props)

	case TaskTypeShowAnyWindow:
		props, err := GetTaskProperties[ShowAnyWindowProperties](*task)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}

		// Clear all properties for ShowAnyWindow
		props.WindowHandle = InvalidHandle
		props.ButtonTextUpper = ""
		props.ButtonTextLower = ""
		props.IconPath = ""
		props.ExePath = ""

		return SetTaskProperties(task, props)
	}

	return nil // No clearing needed for other types
}
