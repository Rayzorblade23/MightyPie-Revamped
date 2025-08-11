package buttonManagerAdapter

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// processExistingShowProgramHandles (Cleaned)
func (a *ButtonManagerAdapter) processExistingShowProgramHandles(
	menuID, pageID string,
	showProgramButtons map[string]*Button,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	pageConfig PageConfig,
) {
	for btnID, buttonPtr := range showProgramButtons {
		buttonCopy := *buttonPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetButtonProperties[core.ShowProgramWindowProperties](buttonCopy)
		if err != nil {
			log.Warn("[%s] Failed to get ShowProgram props: %v", buttonKey, err)
			continue
		}

		buttonModified := false
		if props.WindowHandle > 0 {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				if winInfo.AppName == props.ButtonTextLower {
					originalButton := buttonCopy
					if err := updateButtonWithWindowInfo(&buttonCopy, winInfo, props.WindowHandle); err != nil {
						log.Error("[%s] Failed update ShowProgram button: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalButton, buttonCopy) {
						buttonModified = true
					}
					delete(availableWindows, props.WindowHandle)
					processedButtons[buttonKey] = true
				} else {
					originalButton := buttonCopy
					if err := clearButtonWindowProperties(&buttonCopy); err != nil {
						log.Error("[%s] Failed clear ShowProgram on mismatch: %v", buttonKey, err)
					} else if !reflect.DeepEqual(originalButton, buttonCopy) {
						buttonModified = true
					}
				}
			} else {
				originalButton := buttonCopy
				if err := clearButtonWindowProperties(&buttonCopy); err != nil {
					log.Error("[%s] Failed clear ShowProgram on invalid handle: %v", buttonKey, err)
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
			pageConfig[btnID] = buttonCopy
		}
	}
}

// processExistingShowAnyHandles (Cleaned)
func (a *ButtonManagerAdapter) processExistingShowAnyHandles(
	menuID, pageID string,
	showAnyButtons map[string]*Button,
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	pageConfig PageConfig,
) {
	for btnID, buttonPtr := range showAnyButtons {
		buttonCopy := *buttonPtr
		buttonKey := fmt.Sprintf("%s:%s:%s", menuID, pageID, btnID)
		if processedButtons[buttonKey] {
			continue
		}

		props, err := GetButtonProperties[core.ShowAnyWindowProperties](buttonCopy)
		if err != nil {
			log.Warn("[%s] Failed get ShowAny props: %v", buttonKey, err)
			continue
		}

		buttonModified := false
		if props.WindowHandle != InvalidHandle {
			if winInfo, exists := availableWindows[props.WindowHandle]; exists {
				originalButton := buttonCopy
				if err := updateButtonWithWindowInfo(&buttonCopy, winInfo, props.WindowHandle); err != nil {
					log.Error("[%s] Failed update ShowAny button: %v", buttonKey, err)
				} else if !reflect.DeepEqual(originalButton, buttonCopy) {
					buttonModified = true
				}
				delete(availableWindows, props.WindowHandle)
				processedButtons[buttonKey] = true
			} else {
				originalButton := buttonCopy
				if err := clearButtonWindowProperties(&buttonCopy); err != nil {
					log.Error("[%s] Failed clear ShowAny on invalid handle: %v", buttonKey, err)
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
			pageConfig[btnID] = buttonCopy
		}
	}
}

// assignMatchingProgramWindows (Cleaned)
func (a *ButtonManagerAdapter) assignMatchingProgramWindows(
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	fullUpdatedConfig ConfigData,
) {

	if len(availableWindows) == 0 {
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
			// Sort button IDs numerically so lowest IDs are assigned first
			btnIDs := make([]string, 0, len(bMap))
			for bID := range bMap {
				btnIDs = append(btnIDs, bID)
			}
			sort.Slice(btnIDs, func(i, j int) bool {
				id1, err1 := strconv.Atoi(btnIDs[i])
				id2, err2 := strconv.Atoi(btnIDs[j])
				if err1 != nil || err2 != nil {
					return btnIDs[i] < btnIDs[j] // fallback to string sort
				}
				return id1 < id2
			})
			for _, bID := range btnIDs {
				button := bMap[bID]
				buttonKey := fmt.Sprintf("%s:%s:%s", pID, mID, bID)
				if core.ButtonType(button.ButtonType) != core.ButtonTypeShowProgramWindow || processedButtons[buttonKey] {
					continue
				}

				props, err := GetButtonProperties[core.ShowProgramWindowProperties](button)
				if err != nil {
					log.Warn("[%s] assignMatchingProgramWindows - Failed to get props: %v", buttonKey, err)
					continue
				}

				if props.WindowHandle == InvalidHandle {
					foundHandle := -1
					var foundWinInfo core.WindowInfo
					isEdgeButton := strings.Contains(strings.ToLower(props.ButtonTextLower), "edge")
					if isEdgeButton {
						log.Info("[DEBUG] ShowProgramWindow button '%s' (text: '%s') looking for window", buttonKey, props.ButtonTextLower)
					}
					for handle, winInfo := range availableWindows {
						if windowsConsumed[handle] {
							continue
						}
						isEdge := winInfo.ExeName == "msedge.exe" || winInfo.AppName == "Microsoft Edge"
						if isEdge {
							log.Info("[DEBUG] Found Edge window - Handle: %X, Title: '%s', AppName: '%s'",
								handle, winInfo.Title, winInfo.AppName)
							if isEdge {
								// Try matching by window title using multiple strategies

								// Strategy 1: Exact title match
								if winInfo.Title == props.ButtonTextLower {
									log.Info("[DEBUG] MATCH: Edge window title '%s' exactly matches button text '%s'",
										winInfo.Title, props.ButtonTextLower)
									foundHandle = handle
									foundWinInfo = winInfo
									break
								}

								// Strategy 2: Button text is contained in window title
								if strings.Contains(winInfo.Title, props.ButtonTextLower) {
									log.Info("[DEBUG] MATCH: Edge window title '%s' contains button text '%s'",
										winInfo.Title, props.ButtonTextLower)
									foundHandle = handle
									foundWinInfo = winInfo
									break
								}

								// Strategy 3: Title prefix match (before first ' - ')
								titleParts := strings.Split(winInfo.Title, " - ")
								if len(titleParts) > 0 && titleParts[0] == props.ButtonTextLower {
									log.Info("[DEBUG] MATCH: Edge window title prefix '%s' matches button text '%s'",
										titleParts[0], props.ButtonTextLower)
									foundHandle = handle
									foundWinInfo = winInfo
									break
								}

								log.Info("[DEBUG] NO MATCH: Edge window title '%s' does not match button text '%s' using any strategy",
									winInfo.Title, props.ButtonTextLower)
							} else {
								if winInfo.AppName == props.ButtonTextLower {
									log.Info("[DEBUG] MATCH: Window AppName '%s' matches button text '%s'",
										winInfo.AppName, props.ButtonTextLower)
									foundHandle = handle
									foundWinInfo = winInfo
									break
								} else if isEdgeButton {
									log.Info("[DEBUG] NO MATCH: Window AppName '%s' does not match button text '%s'",
										winInfo.AppName, props.ButtonTextLower)
								}
							}
						} else {
							if winInfo.AppName == props.ButtonTextLower {
								foundHandle = handle
								foundWinInfo = winInfo
								break
							}
						}
					}

					if foundHandle != InvalidHandle {
						targetButtonMap := fullUpdatedConfig[pID][mID]
						buttonToModify := targetButtonMap[bID]
						originalButton := buttonToModify

						err := updateButtonWithWindowInfo(&buttonToModify, foundWinInfo, foundHandle)
						if err != nil {
							log.Error("[%s] assignMatchingProgramWindows - Failed update button: %v", buttonKey, err)
						} else {
							if !reflect.DeepEqual(originalButton, buttonToModify) {
								targetButtonMap[bID] = buttonToModify
							}
							processedButtons[buttonKey] = true
							windowsConsumed[foundHandle] = true
						}
					}
				}
			}
		}
	}

	if len(windowsConsumed) > 0 {
		for handle := range windowsConsumed {
			delete(availableWindows, handle)
		}
	}
}

// assignRemainingWindows (Cleaned)
func (a *ButtonManagerAdapter) assignRemainingWindows(
	availableWindows core.WindowsUpdate,
	processedButtons map[string]bool,
	fullUpdatedConfig ConfigData,
) {
	if len(availableWindows) == 0 {
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
				if core.ButtonType(button.ButtonType) != core.ButtonTypeShowAnyWindow {
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

	assignedCount := 0
	windowsConsumed := make(map[int]bool)

	for i := 0; i < len(availableSlots) && assignedCount < len(windowsToAssign); i++ {
		slot := availableSlots[i]
		window := windowsToAssign[assignedCount]
		slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.MenuID, slot.PageID, slot.ButtonID)

		targetButtonMap := fullUpdatedConfig[slot.MenuID][slot.PageID]
		buttonToModify := targetButtonMap[slot.ButtonID]
		originalButton := buttonToModify

		err := updateButtonWithWindowInfo(&buttonToModify, window.Info, window.Handle)
		if err != nil {
			log.Error("[%s] Failed update button with assigned window: %v", slotButtonKey, err)
			continue
		}

		if !reflect.DeepEqual(originalButton, buttonToModify) {
			targetButtonMap[slot.ButtonID] = buttonToModify
		}

		windowsConsumed[window.Handle] = true
		processedButtons[slotButtonKey] = true
		assignedCount++
	}

	for handle := range windowsConsumed {
		delete(availableWindows, handle)
	}

	// Clear remaining slots
	if assignedCount < len(availableSlots) {
		for i := assignedCount; i < len(availableSlots); i++ {
			slot := availableSlots[i]
			slotButtonKey := fmt.Sprintf("%s:%s:%s", slot.MenuID, slot.PageID, slot.ButtonID)
			targetButtonMap := fullUpdatedConfig[slot.MenuID][slot.PageID]
			buttonToModify := targetButtonMap[slot.ButtonID]
			originalButton := buttonToModify
			err := clearButtonWindowProperties(&buttonToModify)
			if err != nil {
				log.Error("[%s] Failed to clear remaining empty slot: %v", slotButtonKey, err)
			} else if !reflect.DeepEqual(originalButton, buttonToModify) {
				targetButtonMap[slot.ButtonID] = buttonToModify
			}
		}
	}
}

// updateButtonWithWindowInfo (Cleaned)
func updateButtonWithWindowInfo(button *Button, winInfo core.WindowInfo, newHandle int) error {
	switch core.ButtonType(button.ButtonType) {
	case core.ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}
		props.WindowHandle = newHandle
		props.Instance = winInfo.Instance

		isEdge := winInfo.ExeName == "msedge.exe" || winInfo.AppName == "Microsoft Edge"
		if isEdge {
			// For Edge windows, set ButtonTextUpper to window title but keep ButtonTextLower unchanged
			props.ButtonTextUpper = winInfo.Title
			// ButtonTextLower remains unchanged (keeps the original button text)
		} else {
			props.ButtonTextUpper = winInfo.Title
			props.ButtonTextLower = winInfo.AppName
		}

		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		return SetButtonProperties(button, props) // Returns error from SetButtonProperties
	case core.ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}
		props.WindowHandle = newHandle
		props.Instance = winInfo.Instance
		props.ButtonTextUpper = winInfo.Title
		props.ButtonTextLower = winInfo.AppName
		if winInfo.IconPath != "" {
			props.IconPath = winInfo.IconPath
		}
		return SetButtonProperties(button, props)
	}
	return nil // No update needed for other types
}

// clearButtonWindowProperties (Cleaned)
func clearButtonWindowProperties(button *Button) error {
	switch core.ButtonType(button.ButtonType) {
	case core.ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}

		// Preserve existing properties from buttonConfig
		buttonLower := props.ButtonTextLower
		iconPath := props.IconPath

		// Clear window-specific properties but maintain program identity
		props.WindowHandle = InvalidHandle // Use InvalidHandle (-1) for no window
		props.ButtonTextUpper = ""         // Clear window title

		// Ensure we keep program identity
		props.ButtonTextLower = buttonLower // Keep app name
		props.IconPath = iconPath           // Keep icon from buttonConfig

		return SetButtonProperties(button, props)

	case core.ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*button)
		if err != nil {
			return fmt.Errorf("get_props: %w", err)
		}

		// Clear all properties for ShowAnyWindow
		props.WindowHandle = InvalidHandle
		props.ButtonTextUpper = ""
		props.ButtonTextLower = ""
		props.IconPath = ""

		return SetButtonProperties(button, props)
	}

	return nil // No clearing needed for other types
}
