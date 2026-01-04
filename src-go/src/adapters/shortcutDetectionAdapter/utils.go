package shortcutDetectionAdapter

import (
	"slices"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// keyIsPressed checks the current state of a specific virtual key code.
func keyIsPressed(virtualKeyCode int) bool {
	if core.GetKeyState == nil {
		log.Fatal("CRITICAL Error: core.GetKeyState is not initialized.")
		return false
	}
	state, _, _ := core.GetKeyState.Call(uintptr(virtualKeyCode))
	return (state & keyPressedMask) != 0
}

// getActualModifierState checks if a generic modifier (SHIFT, CTRL, ALT) is active
// by checking its left or right specific keys.
func getActualModifierState(genericModifierVKCode int) bool {
	switch genericModifierVKCode {
	case core.VK_SHIFT:
		return keyIsPressed(vkLSHIFT) || keyIsPressed(vkRSHIFT)
	case core.VK_CONTROL:
		return keyIsPressed(vkLCONTROL) || keyIsPressed(vkRCONTROL)
	case core.VK_ALT:
		return keyIsPressed(vkLALT) || keyIsPressed(vkRALT)
	default:
		return keyIsPressed(genericModifierVKCode)
	}
}

// checkShortcutModifiers verifies if all defined modifier keys for a shortcut are currently pressed.
func checkShortcutModifiers(definedModifierCodes []int) bool {
	for _, modCode := range definedModifierCodes {
		isGenericMod := modCode == core.VK_SHIFT || modCode == core.VK_CONTROL || modCode == core.VK_ALT
		if isGenericMod {
			if !getActualModifierState(modCode) {
				return false
			}
		} else {
			if !keyIsPressed(modCode) {
				return false
			}
		}
	}
	return true
}

// isEventKeyInShortcutCodes checks if an event's key code matches any key in a shortcut,
// considering generic vs. specific (L/R) modifier mappings.
func isEventKeyInShortcutCodes(eventVKCode int, shortcutDefinedCodes []int) bool {
	if slices.Contains(shortcutDefinedCodes, eventVKCode) {
		return true
	}
	if genericEquivalent, isSpecificMod := mapSpecificEventModifierToGeneric[eventVKCode]; isSpecificMod {
		if slices.Contains(shortcutDefinedCodes, genericEquivalent) {
			return true
		}
	}
	return false
}

// handleKeyDown processes key down events for shortcut detection.
// Returns true if the event was consumed by a shortcut.
func (adapter *ShortcutDetectionAdapter) handleKeyDown(eventVKCode int) bool {
	// Get current focused app
	adapter.focusedAppMutex.RLock()
	currentFocusedApp := adapter.focusedApp
	adapter.focusedAppMutex.RUnlock()

	// Find all shortcuts that match the pressed keys
	var matchingWithTargetApp []struct {
		index      string
		definition core.ShortcutEntry
	}
	var matchingWithoutTargetApp []struct {
		index      string
		definition core.ShortcutEntry
	}

	for shortcutKeyIndex, shortcutDefinition := range adapter.keyboardHook.shortcuts {
		// Skip shortcuts with no key codes defined
		if len(shortcutDefinition.Codes) == 0 {
			continue
		}
		
		mainShortcutKey := shortcutDefinition.Codes[len(shortcutDefinition.Codes)-1]
		modifierKeys := shortcutDefinition.Codes[:len(shortcutDefinition.Codes)-1]

		if eventVKCode == mainShortcutKey && checkShortcutModifiers(modifierKeys) {
			if shortcutDefinition.TargetApp != nil && *shortcutDefinition.TargetApp != "" {
				matchingWithTargetApp = append(matchingWithTargetApp, struct {
					index      string
					definition core.ShortcutEntry
				}{shortcutKeyIndex, shortcutDefinition})
			} else {
				matchingWithoutTargetApp = append(matchingWithoutTargetApp, struct {
					index      string
					definition core.ShortcutEntry
				}{shortcutKeyIndex, shortcutDefinition})
			}
		}
	}

	// Priority 1: Check if any targetApp matches the focused app
	for _, match := range matchingWithTargetApp {
		if match.definition.TargetApp != nil && *match.definition.TargetApp == currentFocusedApp {
			log.Debug("Shortcut %s triggered for targetApp '%s' (focused)", match.index, *match.definition.TargetApp)
			if adapter.keyboardHook.multiCallback != nil {
				adapter.keyboardHook.multiCallback(match.index, match.definition.Codes, true)
			}
			return true
		}
	}

	// Priority 2: If no targetApp matched, use shortcuts without targetApp
	if len(matchingWithoutTargetApp) > 0 {
		// Only trigger if there are no targetApp shortcuts (already checked above)
		match := matchingWithoutTargetApp[0]
		log.Debug("Shortcut %s triggered (no targetApp restriction)", match.index)
		if adapter.keyboardHook.multiCallback != nil {
			adapter.keyboardHook.multiCallback(match.index, match.definition.Codes, true)
		}
		return true
	}

	// If we have targetApp shortcuts but none matched the focused app, let the key pass through
	if len(matchingWithTargetApp) > 0 {
		targetApp := ""
		if matchingWithTargetApp[0].definition.TargetApp != nil {
			targetApp = *matchingWithTargetApp[0].definition.TargetApp
		}
		log.Debug("Shortcut matched but targetApp '%s' doesn't match focused app '%s', passing through",
			targetApp, currentFocusedApp)
		return false
	}

	return false
}

// handleKeyUp processes key up events for shortcut release detection.
func (adapter *ShortcutDetectionAdapter) handleKeyUp(eventVKCode int) {
	for shortcutKeyIndex, shortcutDefinition := range adapter.keyboardHook.shortcuts {
		// Skip shortcuts with no key codes defined
		if len(shortcutDefinition.Codes) == 0 {
			continue
		}
		
		if adapter.pressedState[shortcutKeyIndex] { // If this shortcut was active
			if isEventKeyInShortcutCodes(eventVKCode, shortcutDefinition.Codes) { // And released key is part of it
				mainShortcutKey := shortcutDefinition.Codes[len(shortcutDefinition.Codes)-1]
				modifierKeys := shortcutDefinition.Codes[:len(shortcutDefinition.Codes)-1]
				mainKeyStillHeld := getActualModifierState(mainShortcutKey)
				allModifiersStillHeld := checkShortcutModifiers(modifierKeys)
				if !(allModifiersStillHeld && mainKeyStillHeld) { // If shortcut no longer fully active
					if adapter.keyboardHook.multiCallback != nil {
						adapter.keyboardHook.multiCallback(shortcutKeyIndex, shortcutDefinition.Codes, false)
					}
				}
			}
		}
	}
}
