package shortcutSetterAdapter

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("ShortcutSetter")

// ShortcutSetterAdapter is a completely independent adapter for capturing shortcuts dynamically.
type ShortcutSetterAdapter struct {
	natsAdapter  *natsAdapter.NatsAdapter
	keyboardHook *setterKeyboardHook
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

	captureShortcutSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE")
	updateSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	abortSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT")
	deleteSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_DELETE")

	// Load and print existing shortcuts
	shortcuts, err := LoadShortcuts()
	if err != nil {
		log.Error("Failed to load shortcuts for initial update: %v", err)
	} else {
		natsAdapter.PublishMessage(updateSubject, shortcuts)
	}

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

	// Subscribe to delete shortcut messages
	natsAdapter.SubscribeToSubject(deleteSubject, func(msg *nats.Msg) {
		var payload ShortcutIndexMessage
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Error("Failed to decode index for delete: %v", err)
			return
		}
		log.Info("Deleting shortcut at index: %d", payload.Index)
		if err := shortcutSetterAdapter.DeleteShortcut(payload.Index); err != nil {
			log.Error("Failed to delete shortcut: %v", err)
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
			// Always stop the hook after the first detected shortcut.
			defer a.keyboardHook.Stop()

			if !IsValidShortcut(shortcut) {
				log.Debug("Invalid shortcut, ignoring")
				return
			}

			if err := a.SaveShortcut(index, shortcut); err != nil {
				log.Error("Failed to save shortcut: %v", err)
			} else {
				log.Info("Shortcut detected and saved for index %d", index)
			}
		})
	})
	go func() {
		if err := a.keyboardHook.Run(); err != nil {
			log.Error("Keyboard hook error: %v", err)
		}
	}()
}
