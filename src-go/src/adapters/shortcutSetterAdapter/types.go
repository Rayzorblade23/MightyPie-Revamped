package shortcutSetterAdapter

import "slices"

import "github.com/Rayzorblade23/MightyPie-Revamped/src/core"

type ShortcutIndexMessage struct {
	Index int `json:"index"`
}

type ShortcutMap map[string]core.ShortcutEntry

// IsValidShortcut checks if a shortcut is valid (not just modifiers, not escape, etc.).
func IsValidShortcut(shortcut []int) bool {
	if len(shortcut) < 1 {
		return false
	}

	// The Escape key is used for cancellation and is not a valid shortcut.
	if slices.Contains(shortcut, core.KeyMap["Esc"]) {
			return false
		}

	// A shortcut must contain at least one non-modifier key.
	mainKey := shortcut[len(shortcut)-1]
	return !core.IsModifier(mainKey)
}