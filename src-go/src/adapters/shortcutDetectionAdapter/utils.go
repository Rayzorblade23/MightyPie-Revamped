package shortcutDetectionAdapter

import (
	"fmt"
	"slices"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// keyIsPressed checks the current state of a specific virtual key code.
func keyIsPressed(virtualKeyCode int) bool {
	if core.GetKeyState == nil {
		fmt.Println("CRITICAL Error: core.GetKeyState is not initialized.")
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
	for shortcutKeyIndex, shortcutDefinition := range adapter.keyboardHook.shortcuts {
		if len(shortcutDefinition.Codes) < 2 {
			continue
		} // Original logic: requires at least 2 keys for a shortcut.
		mainShortcutKey := shortcutDefinition.Codes[len(shortcutDefinition.Codes)-1]
		modifierKeys := shortcutDefinition.Codes[:len(shortcutDefinition.Codes)-1]
		if eventVKCode == mainShortcutKey && checkShortcutModifiers(modifierKeys) {
			if adapter.keyboardHook.multiCallback != nil {
				adapter.keyboardHook.multiCallback(shortcutKeyIndex, shortcutDefinition.Codes, true)
			}
			return true
		}
	}
	return false
}

// handleKeyUp processes key up events for shortcut release detection.
func (adapter *ShortcutDetectionAdapter) handleKeyUp(eventVKCode int) {
	for shortcutKeyIndex, shortcutDefinition := range adapter.keyboardHook.shortcuts {
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