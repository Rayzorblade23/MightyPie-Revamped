package shortcutSetterAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("ShortcutSetter")

// ShortcutSetterAdapter is a completely independent adapter for capturing shortcuts dynamically.
type ShortcutSetterAdapter struct {
	natsAdapter  *natsAdapter.NatsAdapter
	keyboardHook *setterKeyboardHook
	updateSubject string
}

// NormalizeShortcut collapses left/right modifiers into their generic VKs and removes duplicates.
// Output ordering: [modifiers..., main]
func NormalizeShortcut(codes []int) []int {
    if len(codes) == 0 {
        return codes
    }
    // Map any side-specific modifier to generic
    normalizeVK := func(vk int) int {
        switch vk {
        case 0xA2, 0xA3: // LCTRL, RCTRL
            return core.VK_CONTROL
        case 0xA4, 0xA5: // LALT, RALT
            return core.VK_MENU
        case 0xA0, 0xA1: // LSHIFT, RSHIFT
            return core.VK_SHIFT
        case 0x5C: // RWIN
            return core.VK_LWIN
        default:
            return vk
        }
    }

    // Normalize all
    tmp := make([]int, 0, len(codes))
    for _, vk := range codes {
        tmp = append(tmp, normalizeVK(vk))
    }

    // Separate modifiers and main
    modsSet := map[int]struct{}{}
    orderedMods := []int{}
    for i := 0; i < len(tmp)-1; i++ {
        v := tmp[i]
        if v == core.VK_CONTROL || v == core.VK_MENU || v == core.VK_SHIFT || v == core.VK_LWIN {
            if _, ok := modsSet[v]; !ok {
                modsSet[v] = struct{}{}
                orderedMods = append(orderedMods, v)
            }
        }
    }
    main := tmp[len(tmp)-1]
    return append(orderedMods, main)
}

 

// Run blocks forever to keep the worker process alive.
func (a *ShortcutSetterAdapter) Run() {
	log.Info("ShortcutSetterAdapter running.")
	select {}
}

// New creates a new instance and sets up the keyboard hook and NATS adapter.
func New(natsAdapter *natsAdapter.NatsAdapter) *ShortcutSetterAdapter {
	shortcutSetterAdapter := &ShortcutSetterAdapter{
		natsAdapter: natsAdapter,
	}

	captureShortcutSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_CAPTURE")
	abortSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_ABORT")
	shortcutSetterAdapter.updateSubject = os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_UPDATE")

	// Button shortcut subjects
	buttonCaptureSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_CAPTURE")
	buttonAbortSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_ABORT")
	buttonUpdateSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_UPDATE")

	// Stateless: do not read or publish existing shortcuts here. Persistence is handled by piemenuConfigManager.

	// Subscribe to requests to record a new shortcut at a given index.
	// When a message is received, begin listening for a shortcut to assign to that index.
	natsAdapter.SubscribeToSubject(captureShortcutSubject, func(msg *nats.Msg) {
		var payload ShortcutIndexMessage
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Error("Failed to decode index: %v", err)
			return
		}
		log.Info("Shortcut pressed index: %d", payload.Index)
		shortcutSetterAdapter.ListenForShortcutAtIndex(payload.Index)
	})

	// Subscribe to abort messages to stop shortcut detection.
	// When a message is received, stop the current keyboard hook if it is running.
	natsAdapter.SubscribeToSubject(abortSubject, func(msg *nats.Msg) {
		log.Info("Received abort message, stopping shortcut detection.")
		if shortcutSetterAdapter.keyboardHook != nil {
			shortcutSetterAdapter.keyboardHook.Stop()
		}
	})

	// Subscribe to button shortcut capture requests
	natsAdapter.SubscribeToSubject(buttonCaptureSubject, func(msg *nats.Msg) {
		log.Info("Received button shortcut capture request")
		shortcutSetterAdapter.ListenForButtonShortcut(buttonUpdateSubject)
	})

	// Subscribe to button shortcut abort messages
	natsAdapter.SubscribeToSubject(buttonAbortSubject, func(msg *nats.Msg) {
		log.Info("Received button shortcut abort request")
		if shortcutSetterAdapter.keyboardHook != nil {
			shortcutSetterAdapter.keyboardHook.Stop()
		}
	})

	// No delete subscription here; UI publishes delete directly to piemenuConfigManager.

	return shortcutSetterAdapter
}

func (a *ShortcutSetterAdapter) ListenForShortcutAtIndex(index int) {
	var once sync.Once

	// Stop any previous hook before starting a new one
	if a.keyboardHook != nil {
		a.keyboardHook.Stop()
	}

	a.keyboardHook = newSetterKeyboardHook(func(shortcut []int) {
		once.Do(func() {
			// Always stop the hook after the first detected shortcut.
			defer a.keyboardHook.Stop()

			if !IsValidShortcut(shortcut) {
				log.Debug("Invalid shortcut, ignoring")
				return
			}

			// Publish partial update directly; piemenuConfigManager will merge and persist
			update := map[string]core.ShortcutEntry{
				strconv.Itoa(index): {Codes: shortcut, Label: ShortcutCodesToString(shortcut)},
			}
			if a.updateSubject == "" {
				log.Error("Update subject is empty; cannot publish shortcut update")
				return
			}
			a.natsAdapter.PublishMessage(a.updateSubject, update)
			log.Info("Shortcut detected and published for index %d", index)
		})
	})
	go func() {
		if err := a.keyboardHook.Run(); err != nil {
			log.Error("Keyboard hook error: %v", err)
		}
	}()
}


// ListenForButtonShortcut captures a keyboard shortcut for button use and converts it to RobotGo format
func (a *ShortcutSetterAdapter) ListenForButtonShortcut(buttonUpdateSubject string) {
	var once sync.Once

	// Stop any previous hook before starting a new one
	if a.keyboardHook != nil {
		a.keyboardHook.Stop()
	}

	a.keyboardHook = newSetterKeyboardHook(func(shortcut []int) {
		once.Do(func() {
			// Always stop the hook after the first detected shortcut
			defer a.keyboardHook.Stop()

			if !IsValidShortcut(shortcut) {
				log.Debug("Invalid button shortcut, ignoring")
				return
			}

			// Normalize modifiers and deduplicate before processing
			norm := NormalizeShortcut(shortcut)
			// Convert key codes to RobotGo-compatible format and build display label
			robotGoKeys := ConvertToRobotGoFormat(norm)
			displayLabel := ShortcutCodesToString(norm)
			if robotGoKeys == "" {
				log.Warn("Button shortcut could not be converted to RobotGo format; ignoring. Label=%s", displayLabel)
				// Notify frontend to close dialog gracefully
				if buttonUpdateSubject != "" {
					a.natsAdapter.PublishMessage(buttonUpdateSubject, map[string]string{
						"error": "unmappable",
						"label": displayLabel,
					})
				}
				return
			}
			log.Info("Button shortcut captured: %s (label: %s)", robotGoKeys, displayLabel)

			// Publish the captured shortcut with execution keys and display label
			update := map[string]string{
				"keys":  robotGoKeys,
				"label": displayLabel,
			}

			if buttonUpdateSubject == "" {
				log.Error("Button shortcut update subject is empty; cannot publish update")
				return
			}

			a.natsAdapter.PublishMessage(buttonUpdateSubject, update)
			log.Info("Button shortcut published: keys=%s, label=%s", robotGoKeys, displayLabel)
		})
	})

	go func() {
		if err := a.keyboardHook.Run(); err != nil {
			log.Error("Button shortcut keyboard hook error: %v", err)
		}
	}()
}

// ConvertToRobotGoFormat converts Windows virtual key codes to RobotGo-compatible format
func ConvertToRobotGoFormat(keyCodes []int) string {
    if len(keyCodes) == 0 {
        return ""
    }

    var parts []string
    
    // Process all keys except the last one as modifiers
    for i := 0; i < len(keyCodes)-1; i++ {
        keyCode := keyCodes[i]
        if modifier := keyCodeToRobotGoModifier(keyCode); modifier != "" {
            parts = append(parts, modifier)
        } else {
            // If a supposed modifier cannot be converted, treat as invalid shortcut for RobotGo
            return ""
        }
    }
    
    // Process the last key as the main key
    mainKeyCode := keyCodes[len(keyCodes)-1]
    if mainKey := keyCodeToRobotGoKey(mainKeyCode); mainKey != "" {
        parts = append(parts, mainKey)
    } else {
        // Unmappable main key -> invalid for RobotGo
        return ""
    }
    
    return strings.Join(parts, "+")
}

// (removed duplicate NormalizeShortcut)

// keyCodeToRobotGoModifier converts a Windows virtual key code to RobotGo modifier format
func keyCodeToRobotGoModifier(keyCode int) string {
    switch keyCode {
    // Ctrl
    case core.VK_CONTROL, 0xA2, 0xA3:
        return "ctrl"
    // Alt (menu)
    case core.VK_MENU, 0xA4, 0xA5:
        return "alt"
    // Shift
    case core.VK_SHIFT, 0xA0, 0xA1:
        return "shift"
    // Win
    case core.VK_LWIN, core.VK_RWIN:
        return "cmd"
    default:
        // Not a modifier in RobotGo terms
        return ""
    }
}

// keyCodeToRobotGoKey converts a Windows virtual key code to RobotGo key format
func keyCodeToRobotGoKey(keyCode int) string {
    // Resolve canonical key name from our core map
    name := core.FindKeyByValue(keyCode)
    if name != "" {
        if token, ok := core.RobotGoKeyName[name]; ok {
            return token
        }
        // Not explicitly supported by RobotGo mapping
        return ""
    }
    // Not found: do not emit unsupported tokens
    return ""
}

func ShortcutCodesToString(codes []int) string {
	names := []string{}
	for _, k := range codes {
		name := core.FindKeyByValue(k)
		if name == "" {
			name = fmt.Sprintf("VK_%d", k)
		}
		names = append(names, name)
	}
	return strings.Join(names, " + ")
}
