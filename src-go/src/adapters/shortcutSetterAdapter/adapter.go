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
	abortSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT")
	shortcutSetterAdapter.updateSubject = os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")

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
