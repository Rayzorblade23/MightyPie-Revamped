package shortcutSetterAdapter

import "github.com/Rayzorblade23/MightyPie-Revamped/src/core"

type ShortcutIndexMessage struct {
	Index int `json:"index"`
}

type ShortcutMap map[string]core.ShortcutEntry

// IsValidShortcut checks if a shortcut is valid (at least one modifier and a main key).
func IsValidShortcut(shortcut []int) bool {
	if len(shortcut) < 2 {
		return false
	}
	hasMain := false
	for _, k := range shortcut {
		if !core.IsModifier(k) {
			hasMain = true
			break
		}
	}
	if !hasMain {
		return false
	}
	mainKey := shortcut[len(shortcut)-1]
	if core.IsModifier(mainKey) {
		return false
	}
	for _, k := range shortcut[:len(shortcut)-1] {
		if !core.IsModifier(k) {
			return false
		}
	}
	return true
}