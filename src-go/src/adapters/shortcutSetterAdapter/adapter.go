package shortcutSetterAdapter

import (
	"encoding/json"
	"fmt"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

// ShortcutSetterAdapter is a completely independent adapter for capturing shortcuts dynamically.
type ShortcutSetterAdapter struct {
	natsAdapter  *natsAdapter.NatsAdapter
	keyboardHook *setterKeyboardHook
}

// New creates a new instance and sets up the keyboard hook and NATS adapter.
func New(natsAdapter *natsAdapter.NatsAdapter) *ShortcutSetterAdapter {
	shortcutSetterAdapter := &ShortcutSetterAdapter{
		natsAdapter: natsAdapter,
	}

	captureShortcutSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE")
	updateSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	abortSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT")

	// Load and print existing shortcuts
	shortcuts, err := LoadShortcuts()
	if err != nil {
		fmt.Println("Failed to load shortcuts for initial update:", err)
	} else {
		natsAdapter.PublishMessage(updateSubject, shortcuts)
	}

	// Subscribe to requests to record a new shortcut at a given index.
	// When a message is received, begin listening for a shortcut to assign to that index.
	natsAdapter.SubscribeToSubject(captureShortcutSubject, core.GetTypeName(shortcutSetterAdapter), func(msg *nats.Msg) {
		var payload ShortcutIndexMessage
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			fmt.Printf("Failed to decode index: %v\n", err)
			return
		}
		fmt.Printf("[ShortcutSetter] Shortcut pressed index: %d\n", payload.Index)
		shortcutSetterAdapter.ListenForShortcutAtIndex(payload.Index)
	})

	// Subscribe to abort messages to stop shortcut detection.
	// When a message is received, stop the current keyboard hook if it is running.
	natsAdapter.SubscribeToSubject(abortSubject, core.GetTypeName(shortcutSetterAdapter), func(msg *nats.Msg) {
		fmt.Println("Received abort message, stopping shortcut detection.")
		if shortcutSetterAdapter.keyboardHook != nil {
			shortcutSetterAdapter.keyboardHook.Stop()
		}
	})

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
			if !IsValidShortcut(shortcut) {
				fmt.Println("DEBUG: Invalid shortcut, ignoring")
				return
			}
			if err := a.SaveShortcut(index, shortcut); err != nil {
				fmt.Println("Failed to save shortcut:", err)
			} else {
				fmt.Printf("Shortcut detected and saved for index %d\n", index)
			}
			a.keyboardHook.Stop()
		})
	})
	go func() {
		if err := a.keyboardHook.Run(); err != nil {
			fmt.Println("Keyboard hook error:", err)
		}
	}()
}
