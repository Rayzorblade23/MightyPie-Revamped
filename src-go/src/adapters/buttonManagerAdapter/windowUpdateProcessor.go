package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"reflect"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	// No reflect needed here if only using JSON compare
)

// processWindowUpdate - Refactored structure (Cleaned)
func (a *ButtonManagerAdapter) processWindowUpdate(currentConfig ConfigData, windows core.WindowsUpdate) (ConfigData, error) {
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
	availableWindows := make(core.WindowsUpdate, len(windows))
	maps.Copy(availableWindows, windows)
	processedButtons := make(map[string]bool)

	// 4. === Phase 1: Process Existing Handles and Non-Window Buttons ===
	// log.Println("DEBUG: processWindowUpdate - Starting Phase 1: Process existing state...") // Removed DEBUG
	for menuID, menuConfig := range updatedConfig {
		if menuConfig == nil {
			continue
		}
		for pageID, buttonMap := range menuConfig {
			if buttonMap == nil {
				continue
			}

			showProgramButtons, showAnyButtons, _, _ :=
				a.separateButtonsByType(buttonMap)

			// Process buttons
			a.processExistingShowProgramHandles(menuID, pageID, showProgramButtons, availableWindows, processedButtons, buttonMap)
			a.assignMatchingProgramWindows(availableWindows, processedButtons, updatedConfig)
			a.processExistingShowAnyHandles(menuID, pageID, showAnyButtons, availableWindows, processedButtons, buttonMap)
		}
	}
	// log.Printf("DEBUG: processWindowUpdate - Finished Phase 1. Remaining windows: %d", len(availableWindows)) // Removed DEBUG

	// 5. === Phase 1.5 (NEW): Assign Matching Program Windows ===
	// log.Println("DEBUG: processWindowUpdate - Starting Phase 1.5: Assign matching program windows...") // Removed DEBUG
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

// separateButtonsByType (Assuming no DEBUG logs were present)
func (a *ButtonManagerAdapter) separateButtonsByType(buttonMap PageConfig) (
	showProgram map[string]*Button,
	showAny map[string]*Button,
	launchProgram map[string]*Button,
	functionCall map[string]*Button) {

	showProgram = make(map[string]*Button)
	showAny = make(map[string]*Button)
	launchProgram = make(map[string]*Button)
	functionCall = make(map[string]*Button)

	for btnID := range buttonMap {
		// Create a pointer to the button *in the map* to allow modification by callers
		buttonPtr := buttonMap[btnID] // Get pointer to map value directly

		switch ButtonType(buttonPtr.ButtonType) { // Check type via pointer
		case ButtonTypeShowProgramWindow:
			showProgram[btnID] = &buttonPtr // Store pointer
		case ButtonTypeShowAnyWindow:
			showAny[btnID] = &buttonPtr // Store pointer
		case ButtonTypeLaunchProgram:
			launchProgram[btnID] = &buttonPtr // Store pointer
		case ButtonTypeCallFunction:
			functionCall[btnID] = &buttonPtr // Store pointer
		}
	}
	return
}

// handleEmptyWindowList (Cleaned)
func (a *ButtonManagerAdapter) handleEmptyWindowList(configToModify ConfigData) (ConfigData, error) {
	// log.Println("DEBUG: Entering handleEmptyWindowList...") // Removed DEBUG
	anyChangeMade := false

	for menuID, menuConfig := range configToModify {
		if menuConfig == nil {
			continue
		}
		for pageID, buttonMap := range menuConfig {
			if buttonMap == nil {
				continue
			}
			for btnID, button := range buttonMap {
				buttonCopy := button
				originalButtonBeforeClear := button

				err := clearButtonWindowProperties(&buttonCopy) // Try to clear the copy
				if err != nil {
					log.Printf("ERROR: Failed to clear properties for button (P:%s M:%s B:%s) on empty window list: %v", menuID, pageID, btnID, err)
					// Continue? Or return error? Continue seems reasonable.
				} else {
					if !reflect.DeepEqual(originalButtonBeforeClear, buttonCopy) {
						buttonMap[btnID] = buttonCopy // Update the actual map
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
