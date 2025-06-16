package shortcutDetectionAdapter

import (
    "github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)


type KeyboardHook struct {
	multiCallback func(string, []int, bool) bool
	shortcuts     map[string]core.ShortcutEntry
}

func NewKeyboardHookForShortcuts(s map[string]core.ShortcutEntry, cb func(string, []int, bool) bool) *KeyboardHook {
	return &KeyboardHook{cb, s}
}