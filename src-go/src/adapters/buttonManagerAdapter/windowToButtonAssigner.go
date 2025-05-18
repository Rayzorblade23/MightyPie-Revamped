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
	showProgramButtons map[string]*Button,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap PageConfig,
) {
	// log.Printf("DEBUG: processExistingShowProgramHandles - Starting for P:%s M:%s", menuID, pageID) // Removed DEBUG
	for btnID, buttonPtr := range showProgramButtons {
		buttonCopy := *buttonPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetButtonProperties[core.ShowProgramWindowProperties](buttonCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed to get ShowProgram props: %v", buttonKey, err)
			continue
		}

		buttonModified := false
		if props.WindowHandle > 0 {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				if winInfo.ExePath == props.ExePath {
					// log.Printf("DEBUG: [%s] Found valid existing handle %d for %s.", buttonKey, props.WindowHandle, props.ExePath) // Removed DEBUG
					originalButton := buttonCopy
					if err := updateButtonWithWindowInfo(&buttonCopy, winInfo, props.WindowHandle); err != nil {
						log.Printf("ERROR: [%s] Failed update ShowProgram button: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalButton, buttonCopy) {
						buttonModified = true
					}
					delete(availableWindows, props.WindowHandle)
					processedButtons[buttonKey] = true
				} else {
					// log.Printf("DEBUG: [%s] Handle %d ExePath '%s' mismatch expected '%s'. Clearing.", buttonKey, props.WindowHandle, winInfo.ExePath, props.ExePath) // Removed DEBUG
					originalButton := buttonCopy
					if err := clearButtonWindowProperties(&buttonCopy); err != nil {
						log.Printf("ERROR: [%s] Failed clear ShowProgram on mismatch: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalButton, buttonCopy) {
						buttonModified = true
					}
				}
			} else {
				// log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalButton := buttonCopy
				if err := clearButtonWindowProperties(&buttonCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowProgram on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalButton, buttonCopy) {
					buttonModified = true
				}
			}
		} else {
			originalButton := buttonCopy
			if err := clearButtonWindowProperties(&buttonCopy); err == nil && !reflect.DeepEqual(originalButton, buttonCopy) {
				buttonModified = true
			}
		}

		if buttonModified {
			// log.Printf("DEBUG: [%s] ShowProgram Button modified, updating buttonMap.", buttonKey) // Removed DEBUG
			buttonMap[btnID] = buttonCopy
		}
	}
}

// processExistingShowAnyHandles (Cleaned)
func (a *ButtonManagerAdapter) processExistingShowAnyHandles(
	menuID, pageID string,
	showAnyButtons map[string]*Button,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	buttonMap PageConfig,
) {
	// log.Printf("DEBUG: processExistingShowAnyHandles - Starting Step D logic for P:%s M:%s", menuID, pageID) // Removed DEBUG
	for btnID, buttonPtr := range showAnyButtons {
		buttonCopy := *buttonPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetButtonProperties[core.ShowAnyWindowProperties](buttonCopy)
		if err != nil {
			log.Printf("WARN: [%s] Failed get ShowAny props: %v", buttonKey, err)
			continue
		}

		buttonModified := false
		if props.WindowHandle != InvalidHandle {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				// log.Printf("DEBUG: [%s] Found valid existing handle %d.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalButton := buttonCopy
				if err := updateButtonWithWindowInfo(&buttonCopy, winInfo, props.WindowHandle); err != nil {
					log.Printf("ERROR: [%s] Failed update ShowAny button: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalButton, buttonCopy) {
					buttonModified = true
				}
				delete(availableWindows, props.WindowHandle)
				processedButtons[buttonKey] = true
			} else {
				// log.Printf("DEBUG: [%s] Existing handle %d invalid/closed. Clearing.", buttonKey, props.WindowHandle) // Removed DEBUG
				originalButton := buttonCopy
				if err := clearButtonWindowProperties(&buttonCopy); err != nil {
					log.Printf("ERROR: [%s] Failed clear ShowAny on invalid handle: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalButton, buttonCopy) {
					buttonModified = true
				}
			}
		} else {
			originalButton := buttonCopy
			if err := clearButtonWindowProperties(&buttonCopy); err == nil && !reflect.DeepEqual(originalButton, buttonCopy) {
				buttonModified = true
			}
		}

		if buttonModified {
			// log.Printf("DEBUG: [%s] ShowAny Button modified, updating buttonMap.", buttonKey) // Removed DEBUG
			buttonMap[btnID] = buttonCopy
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
			for bID, button := range bMap {
				buttonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if ButtonType(button.ButtonType) != ButtonTypeShowProgramWindow || processedButtons[buttonKey] {
					continue
				}

				props, err := GetButtonProperties[core.ShowProgramWindowProperties](button)
				if err != nil {
					log.Printf("WARN: [%s] assignMatchingProgramWindows - Failed to get props: %v", buttonKey, err)
					continue
				}

				if props.WindowHandle == InvalidHandle && props.ExePath != "" {
					foundHandle := -1
					var foundWinInfo core.WindowInfo
					for handle, winInfo := range availableWindows {
						if windowsConsumed[handle] {
							continue
						}
						isEdge := winInfo.ExeName == "msedge.exe" || winInfo.AppName == "Microsoft Edge"
						if isEdge {
							// Try matching by window title (ButtonTextUpper)
							if winInfo.Title == props.ButtonTextLower {
								foundHandle = handle
								foundWinInfo = winInfo
								break
							}
						} else {
							// Default: match by AppName (ButtonTextLower)
							if winInfo.AppName == props.ButtonTextUpper {
								foundHandle = handle
								foundWinInfo = winInfo
								break
							}
						}
					}

					if foundHandle != InvalidHandle {
						// log.Printf("DEBUG: [%s] assignMatchingProgramWindows - Found matching window H:%d for ExePath '%s'. Assigning.", buttonKey, foundHandle, props.ExePath) // Removed DEBUG
						targetButtonMap := fullUpdatedConfig[pID][mID]
						buttonToModify := targetButtonMap[bID]
						originalButton := buttonToModify

						err := updateButtonWithWindowInfo(&buttonToModify, foundWinInfo, foundHandle)
						if err != nil {
							log.Printf("ERROR: [%s] assignMatchingProgramWindows - Failed update button: %v", buttonKey, err)
						} else {
							if !reflect.DeepEqual(originalButton, buttonToModify) {
								// log.Printf("DEBUG: [%s] Button updated by assignment, writing back.", buttonKey) // Removed DEBUG
								targetButtonMap[bID] = buttonToModify
							}
							// else { log.Printf("DEBUG: [%s] Button update resulted in no change.", buttonKey) } // Removed DEBUG

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
			for bID, button := range bMap {
				bIdx, errB := strconv.Atoi(bID)
				if errB != nil {
					continue
				}
				if ButtonType(button.ButtonType) != ButtonTypeShowAnyWindow {
					continue
				}
				currentButtonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if processedButtons[currentButtonKey] {
					continue
				}
				props, err := GetButtonProperties[core.ShowAnyWindowProperties](button)
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
		buttonToModify := targetButtonMap[slot.ButtonID]
		originalButton := buttonToModify

		// log.Printf("DEBUG: [%s] Assigning window '%s' (H:%d)", slotButtonKey, window.Info.Title, window.Handle) // Removed DEBUG

		err := updateButtonWithWindowInfo(&buttonToModify, window.Info, window.Handle)
		if err != nil {
			log.Printf("ERROR: [%s] Failed update button with assigned window: %v", slotButtonKey, err)
			continue
		}

		if !reflect.DeepEqual(originalButton, buttonToModify) {
			// log.Printf("DEBUG: [%s] Button updated by assignment, writing back.", slotButtonKey) // Removed DEBUG
			targetButtonMap[slot.ButtonID] = buttonToModify
		}
		// else { log.Printf("DEBUG: [%s] Button update assignment resulted in no change.", slotButtonKey) } // Removed DEBUG

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
			buttonToModify := targetButtonMap[slot.ButtonID]
			originalButton := buttonToModify
			err := clearButtonWindowProperties(&buttonToModify)
			if err != nil {
				log.Printf("ERROR: [%s] Failed to clear remaining empty slot: %v", slotButtonKey, err)
			} else if !reflect.DeepEqual(originalButton, buttonToModify) {
				// log.Printf("DEBUG: [%s] Cleared remaining empty slot.", slotButtonKey) // Removed DEBUG
				targetButtonMap[slot.ButtonID] = buttonToModify
			}
		}
	}
	// log.Println("DEBUG: assignRemainingWindows - Finished.") // Removed DEBUG
}

// updateButtonWithWindowInfo (Cleaned)
func updateButtonWithWindowInfo(button *Button, winInfo core.WindowInfo, newHandle int) error {
	// log.Printf("DEBUG: updateButtonWithWindowInfo called for button type %s with handle %d", button.ButtonType, newHandle) // Removed DEBUG
	switch ButtonType(button.ButtonType) {
	case ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}
		props.WindowHandle = newHandle
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		return SetButtonProperties(button, props) // Returns error from SetButtonProperties
	case ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*button)
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
		return SetButtonProperties(button, props)
	}
	return nil // No update needed for other types
}

// clearButtonWindowProperties (Cleaned)
func clearButtonWindowProperties(button *Button) error {
	switch ButtonType(button.ButtonType) {
	case ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*button)
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

		return SetButtonProperties(button, props)

	case ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}

		// Clear all properties for ShowAnyWindow
		props.WindowHandle = InvalidHandle
		props.ButtonTextUpper = ""
		props.ButtonTextLower = ""
		props.IconPath = ""
		props.ExePath = ""

		return SetButtonProperties(button, props)
	}

	return nil // No clearing needed for other types
}
