package shortcutSetterAdapter

import "github.com/Rayzorblade23/MightyPie-Revamped/src/core"

type ShortcutIndexMessage struct {
	Index int `json:"index"`
}

type ShortcutMap map[string]core.ShortcutEntry

// IsValidShortcut checks if a shortcut is valid (at least one non-modifier key).
func IsValidShortcut(shortcut []int) bool {
	if len(shortcut) < 1 {
		return false
	}
	mainKey := shortcut[len(shortcut)-1]
	if core.IsModifier(mainKey) {
		return false
	}
	return true
}