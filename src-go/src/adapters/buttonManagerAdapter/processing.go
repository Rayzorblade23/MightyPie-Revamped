package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"reflect"
	// No reflect needed here if only using JSON compare
)

// processWindowUpdate - Refactored structure (Cleaned)
func (a *ButtonManagerAdapter) processWindowUpdate(currentConfig ConfigData, windows WindowsUpdate) (ConfigData, error) {
	if len(currentConfig) == 0 {
		// INFO level might be appropriate if this state is unusual
		log.Println("INFO: Skipping button processing - currentConfig is empty.")
		return nil, nil
	}

	// 1. Deep Copy Config
	updatedConfig, err := deepCopyConfig(currentConfig)
	if err != nil || updatedConfig == nil || (len(updatedConfig) == 0 && len(currentConfig) > 0) {
		log.Printf("ERROR: processWindowUpdate - Deep copy failed or resulted in invalid state. Error: %v", err)
		return nil, fmt.Errorf("config deep copy failed: %w", err)
	}

	// 2. Handle Empty Window List Case
	if len(windows) == 0 {
		return a.handleEmptyWindowListAndCompare(currentConfig, updatedConfig) // Use helper
	}

	// 3. Setup for Processing - Shared State
	availableWindows := make(WindowsUpdate, len(windows))
	maps.Copy(availableWindows, windows)
	processedButtons := make(map[string]bool)

	// 4. === Phase 1: Process Existing Handles and Non-Window Tasks ===
	// log.Println("DEBUG: processWindowUpdate - Starting Phase 1: Process existing state...") // Removed DEBUG
	for profileID, menuConfig := range updatedConfig {
		if menuConfig == nil { continue }
		for menuID, buttonMap := range menuConfig {
			if buttonMap == nil { continue }

			var originalButtonMap ButtonMap
			if currentConfig[profileID] != nil && currentConfig[profileID][menuID] != nil {
				originalButtonMap = currentConfig[profileID][menuID]
			}

			showProgramButtons, showAnyButtons, launchProgramButtons, functionCallButtons :=
				a.separateTasksByType(buttonMap)

			// Process tasks
			a.processLaunchProgramTasks(profileID, menuID, launchProgramButtons, buttonMap)
			a.processFunctionCallTasks(profileID, menuID, functionCallButtons, buttonMap, originalButtonMap)
			a.processExistingShowProgramHandles(profileID, menuID, showProgramButtons, availableWindows, processedButtons, buttonMap)
			a.processExistingShowAnyHandles(profileID, menuID, showAnyButtons, availableWindows, processedButtons, buttonMap)
		}
	}
	// log.Printf("DEBUG: processWindowUpdate - Finished Phase 1. Remaining windows: %d", len(availableWindows)) // Removed DEBUG


	// 5. === Phase 1.5 (NEW): Assign Matching Program Windows ===
	// log.Println("DEBUG: processWindowUpdate - Starting Phase 1.5: Assign matching program windows...") // Removed DEBUG
	a.assignMatchingProgramWindows(availableWindows, processedButtons, updatedConfig)
	// log.Printf("DEBUG: processWindowUpdate - Finished Phase 1.5. Remaining windows: %d", len(availableWindows)) // Removed DEBUG


	// 6. === Phase 2: Assign Remaining Windows to Available ShowAny Slots ===
	// log.Println("DEBUG: processWindowUpdate - Starting Phase 2: Assign remaining windows to ShowAny slots...") // Removed DEBUG
	a.assignRemainingWindows(availableWindows, processedButtons, updatedConfig)
	// log.Println("DEBUG: processWindowUpdate - Finished Phase 2.") // Removed DEBUG


	// 7. Final Comparison (Use JSON)
	// log.Println("DEBUG: processWindowUpdate - Performing final JSON comparison...") // Removed DEBUG
	jsonSnapshotFinal, errSnapFinal := json.Marshal(currentConfig)
	jsonUpdatedFinal, errUpdateFinal := json.Marshal(updatedConfig)

	if errSnapFinal != nil || errUpdateFinal != nil {
		log.Printf("ERROR: Failed to marshal for final comparison (SnapErr: %v, UpdateErr: %v)", errSnapFinal, errUpdateFinal)
		return nil, fmt.Errorf("final marshal error")
	}

	if bytes.Equal(jsonSnapshotFinal, jsonUpdatedFinal) {
		// log.Println("DEBUG: Final JSON comparison shows configurations ARE equal. Returning nil.") // Removed DEBUG
		return nil, nil
	}

	log.Println("INFO: Final JSON comparison shows configurations ARE different. Returning updated config.") // Keep INFO
	return updatedConfig, nil
}

// Helper for empty window list case (Cleaned)
func (a *ButtonManagerAdapter) handleEmptyWindowListAndCompare(currentConfig, updatedConfig ConfigData) (ConfigData, error) {
	// log.Println("DEBUG: Handling empty window list.") // Removed DEBUG
	_, err := a.handleEmptyWindowList(updatedConfig) // Modifies updatedConfig directly
	if err != nil {
		log.Printf("ERROR: handleEmptyWindowList returned error: %v", err)
		return nil, fmt.Errorf("error handling empty window list: %w", err)
	}

	// Compare original with potentially modified config
	jsonSnapshot, errSnap := json.Marshal(currentConfig)
	jsonAfterClear, errClear := json.Marshal(updatedConfig)
	if errSnap != nil || errClear != nil {
		 log.Printf("ERROR: Failed to marshal for empty list comparison (SnapErr: %v, ClearErr: %v)", errSnap, errClear)
		 return nil, fmt.Errorf("marshal error during empty list check")
	}
	if bytes.Equal(jsonSnapshot, jsonAfterClear) {
		// log.Println("DEBUG: JSON comparison shows config unchanged after clearing. Returning nil.") // Removed DEBUG
		return nil, nil
	}
	// log.Println("DEBUG: JSON comparison shows config *changed* after clearing. Returning modified config.") // Removed DEBUG
	return updatedConfig, nil
}

// separateTasksByType (Assuming no DEBUG logs were present)
func (a *ButtonManagerAdapter) separateTasksByType(buttonMap ButtonMap) (
	showProgram map[string]*Task,
	showAny map[string]*Task,
	launchProgram map[string]*Task,
	functionCall map[string]*Task) {

	showProgram = make(map[string]*Task)
	showAny = make(map[string]*Task)
	launchProgram = make(map[string]*Task)
	functionCall = make(map[string]*Task)

	for btnID := range buttonMap {
		// Create a pointer to the task *in the map* to allow modification by callers
        taskPtr := buttonMap[btnID] // Get pointer to map value directly

		switch TaskType(taskPtr.TaskType) { // Check type via pointer
		case TaskTypeShowProgramWindow:
			showProgram[btnID] = &taskPtr // Store pointer
		case TaskTypeShowAnyWindow:
			showAny[btnID] = &taskPtr // Store pointer
		case TaskTypeLaunchProgram:
			launchProgram[btnID] = &taskPtr // Store pointer
		case TaskTypeCallFunction:
			functionCall[btnID] = &taskPtr // Store pointer
		}
	}
	return
}

// handleEmptyWindowList (Cleaned)
func (a *ButtonManagerAdapter) handleEmptyWindowList(configToModify ConfigData) (ConfigData, error) {
	// log.Println("DEBUG: Entering handleEmptyWindowList...") // Removed DEBUG
	anyChangeMade := false

	for profileID, menuConfig := range configToModify {
		if menuConfig == nil { continue }
		for menuID, buttonMap := range menuConfig {
			if buttonMap == nil { continue }
			for btnID, task := range buttonMap {
				taskCopy := task
				originalTaskBeforeClear := task

				err := clearButtonWindowProperties(&taskCopy) // Try to clear the copy
				if err != nil {
					log.Printf("ERROR: Failed to clear properties for task (P:%s M:%s B:%s) on empty window list: %v", profileID, menuID, btnID, err)
					// Continue? Or return error? Continue seems reasonable.
				} else {
					if !reflect.DeepEqual(originalTaskBeforeClear, taskCopy) {
						buttonMap[btnID] = taskCopy // Update the actual map
						anyChangeMade = true
					}
				}
			}
		}
	}

	if anyChangeMade {
		// log.Printf("DEBUG: handleEmptyWindowList detected internal changes.") // Removed DEBUG
		return configToModify, nil // Return modified map reference
	}

	// log.Printf("DEBUG: handleEmptyWindowList detected no internal changes.") // Removed DEBUG
	return nil, nil // Return nil to signal no changes made *by this function*
}