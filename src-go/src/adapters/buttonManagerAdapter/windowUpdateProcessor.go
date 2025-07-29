package buttonManagerAdapter



import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	// No reflect needed here if only using JSON compare
)

// processWindowUpdate - Refactored structure (Cleaned)
func (a *ButtonManagerAdapter) processWindowUpdate(currentConfig ConfigData, windows core.WindowsUpdate) (ConfigData, error) {
	if len(currentConfig) == 0 {
		log.Info("Skipping button processing - currentConfig is empty.")
		return nil, nil
	}

	updatedConfig, err := deepCopyConfig(currentConfig)
	if err != nil || updatedConfig == nil || (len(updatedConfig) == 0 && len(currentConfig) > 0) {
		log.Error("processWindowUpdate - Deep copy failed or resulted in invalid state. Error: %v", err)
		return nil, fmt.Errorf("config deep copy failed: %w", err)
	}

	if len(windows) == 0 {
		return a.handleEmptyWindowListAndCompare(currentConfig, updatedConfig) // Use helper
	}

	availableWindows := make(core.WindowsUpdate, len(windows))
	maps.Copy(availableWindows, windows)
	processedButtons := make(map[string]bool)

	// First: run processExistingShowProgramHandles for all pages
	for menuID, menuConfig := range updatedConfig {
		if menuConfig == nil {
			continue
		}
		for pageID, pageConfig := range menuConfig {
			if pageConfig == nil {
				continue
			}
			showProgramButtons, _, _, _, _ := a.separateButtonsByType(pageConfig)
			a.processExistingShowProgramHandles(menuID, pageID, showProgramButtons, availableWindows, processedButtons, pageConfig)
		}
	}
	// Then: run assignMatchingProgramWindows and processExistingShowAnyHandles for all pages
	for menuID, menuConfig := range updatedConfig {
		if menuConfig == nil {
			continue
		}
		for pageID, pageConfig := range menuConfig {
			if pageConfig == nil {
				continue
			}
			_, showAnyButtons, _, _, _ := a.separateButtonsByType(pageConfig)
			a.assignMatchingProgramWindows(availableWindows, processedButtons, updatedConfig)
			a.processExistingShowAnyHandles(menuID, pageID, showAnyButtons, availableWindows, processedButtons, pageConfig)
		}
	}

	a.assignRemainingWindows(availableWindows, processedButtons, updatedConfig)

	jsonSnapshotFinal, errSnapFinal := json.Marshal(currentConfig)
	jsonUpdatedFinal, errUpdateFinal := json.Marshal(updatedConfig)

	if errSnapFinal != nil || errUpdateFinal != nil {
		log.Error("Failed to marshal for final comparison (SnapErr: %v, UpdateErr: %v)", errSnapFinal, errUpdateFinal)
		return nil, fmt.Errorf("final marshal error")
	}

	if bytes.Equal(jsonSnapshotFinal, jsonUpdatedFinal) {
		return nil, nil
	}

	return updatedConfig, nil
}

// Helper for empty window list case (Cleaned)
func (a *ButtonManagerAdapter) handleEmptyWindowListAndCompare(currentConfig, updatedConfig ConfigData) (ConfigData, error) {
	_, err := a.handleEmptyWindowList(updatedConfig) // Modifies updatedConfig directly
	if err != nil {
		log.Error("handleEmptyWindowList returned error: %v", err)
		return nil, fmt.Errorf("error handling empty window list: %w", err)
	}

	// Compare original with potentially modified config
	jsonSnapshot, errSnap := json.Marshal(currentConfig)
	jsonAfterClear, errClear := json.Marshal(updatedConfig)
	if errSnap != nil || errClear != nil {
		log.Error("Failed to marshal for empty list comparison (SnapErr: %v, ClearErr: %v)", errSnap, errClear)
		return nil, fmt.Errorf("marshal error during empty list check")
	}
	if bytes.Equal(jsonSnapshot, jsonAfterClear) {
		return nil, nil
	}
	return updatedConfig, nil
}

// separateButtonsByType (Assuming no DEBUG logs were present)
func (a *ButtonManagerAdapter) separateButtonsByType(pageConfig PageConfig) (
	showProgram map[string]*Button,
	showAny map[string]*Button,
	launchProgram map[string]*Button,
	functionCall map[string]*Button,
	openPageInMenu map[string]*Button) {

	showProgram = make(map[string]*Button)
	showAny = make(map[string]*Button)
	launchProgram = make(map[string]*Button)
	functionCall = make(map[string]*Button)
	openPageInMenu = make(map[string]*Button)

	for btnID := range pageConfig {
		// Create a pointer to the button *in the map* to allow modification by callers
		buttonPtr := pageConfig[btnID] // Get pointer to map value directly

		switch core.ButtonType(buttonPtr.ButtonType) { // Check type via pointer
		case core.ButtonTypeShowProgramWindow:
			showProgram[btnID] = &buttonPtr // Store pointer
		case core.ButtonTypeShowAnyWindow:
			showAny[btnID] = &buttonPtr // Store pointer
		case core.ButtonTypeLaunchProgram:
			launchProgram[btnID] = &buttonPtr // Store pointer
		case core.ButtonTypeCallFunction:
			functionCall[btnID] = &buttonPtr // Store pointer
		case core.ButtonTypeOpenPageInMenu:
			openPageInMenu[btnID] = &buttonPtr // Store pointer
		}
	}
	return
}

// handleEmptyWindowList (Cleaned)
func (a *ButtonManagerAdapter) handleEmptyWindowList(configToModify ConfigData) (ConfigData, error) {
	anyChangeMade := false

	for menuID, menuConfig := range configToModify {
		if menuConfig == nil {
			continue
		}
		for pageID, pageConfig := range menuConfig {
			if pageConfig == nil {
				continue
			}
			for btnID, button := range pageConfig {
				buttonCopy := button
				originalButtonBeforeClear := button

				err := clearButtonWindowProperties(&buttonCopy) // Try to clear the copy
				if err != nil {
					log.Error("Failed to clear properties for button (P:%s M:%s B:%s) on empty window list: %v", menuID, pageID, btnID, err)
					// Continue? Or return error? Continue seems reasonable.
				} else {
					if !reflect.DeepEqual(originalButtonBeforeClear, buttonCopy) {
						pageConfig[btnID] = buttonCopy // Update the actual map
						anyChangeMade = true
					}
				}
			}
		}
	}

	if anyChangeMade {
		return configToModify, nil // Return modified map reference
	}

	return nil, nil // Return nil to signal no changes made *by this function*
}
