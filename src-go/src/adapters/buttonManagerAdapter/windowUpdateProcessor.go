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
			// Use cached separated buttons instead of re-separating
			separated := getSeparatedButtons(menuID, pageID)
			if separated == nil {
				continue
			}
			a.processExistingShowProgramHandles(menuID, pageID, separated.ShowProgram, availableWindows, processedButtons, pageConfig)
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
			// Use cached separated buttons instead of re-separating
			separated := getSeparatedButtons(menuID, pageID)
			if separated == nil {
				continue
			}
			a.assignMatchingProgramWindows(availableWindows, processedButtons, updatedConfig)
			a.processExistingShowAnyHandles(menuID, pageID, separated.ShowAny, availableWindows, processedButtons, pageConfig)
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

// buildSeparatedButtonsCache creates a cache of buttons separated by type for the entire config
func buildSeparatedButtonsCache(config ConfigData) SeparatedButtonsCache {
	cache := make(SeparatedButtonsCache)

	for menuID, menuConfig := range config {
		if menuConfig == nil {
			continue
		}
		cache[menuID] = make(map[string]*SeparatedButtons)

		for pageID, pageConfig := range menuConfig {
			if pageConfig == nil {
				continue
			}
			cache[menuID][pageID] = separateButtonsByType(pageConfig)
		}
	}

	return cache
}

// separateButtonsByType separates buttons by type for a single page
func separateButtonsByType(pageConfig PageConfig) *SeparatedButtons {
	separated := &SeparatedButtons{
		ShowProgram:      make(map[string]*Button),
		ShowAny:          make(map[string]*Button),
		LaunchProgram:    make(map[string]*Button),
		FunctionCall:     make(map[string]*Button),
		OpenPageInMenu:   make(map[string]*Button),
		OpenResource:     make(map[string]*Button),
		KeyboardShortcut: make(map[string]*Button),
	}

	for btnID := range pageConfig {
		buttonPtr := pageConfig[btnID]

		switch core.ButtonType(buttonPtr.ButtonType) {
		case core.ButtonTypeShowProgramWindow:
			separated.ShowProgram[btnID] = &buttonPtr
		case core.ButtonTypeShowAnyWindow:
			separated.ShowAny[btnID] = &buttonPtr
		case core.ButtonTypeLaunchProgram:
			separated.LaunchProgram[btnID] = &buttonPtr
		case core.ButtonTypeCallFunction:
			separated.FunctionCall[btnID] = &buttonPtr
		case core.ButtonTypeOpenPageInMenu:
			separated.OpenPageInMenu[btnID] = &buttonPtr
		case core.ButtonTypeOpenResource:
			separated.OpenResource[btnID] = &buttonPtr
		case core.ButtonTypeKeyboardShortcut:
			separated.KeyboardShortcut[btnID] = &buttonPtr
		}
	}
	return separated
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
