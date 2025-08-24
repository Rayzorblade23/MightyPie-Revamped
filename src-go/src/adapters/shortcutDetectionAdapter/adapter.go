// Production-Ready: Minimal Comments, Refactored hookProc, Debug hardcoding commented
package shortcutDetectionAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("ShortcutDetector")

const (
	vkLSHIFT          = 0xA0
	vkRSHIFT          = 0xA1
	vkLCONTROL        = 0xA2
	vkRCONTROL        = 0xA3
	vkLALT            = 0xA4
	vkRALT            = 0xA5
	keyPressedMask    = 0x8000
	keyAutorepeatFlag = 0x40000000
)

var mapSpecificEventModifierToGeneric = map[int]int{
	vkLSHIFT: core.VK_SHIFT, vkRSHIFT: core.VK_SHIFT,
	vkLCONTROL: core.VK_CONTROL, vkRCONTROL: core.VK_CONTROL,
	vkLALT: core.VK_ALT, vkRALT: core.VK_ALT,
}

type ShortcutDetectionAdapter struct {
	natsAdapter    *natsAdapter.NatsAdapter
	keyboardHook   *KeyboardHook
	hook           syscall.Handle
	shortcuts      map[string]core.ShortcutEntry
	pressedState   map[string]bool
	updateHookChan chan struct{}
}

// Run blocks forever to keep the worker process alive.
func (a *ShortcutDetectionAdapter) Run() {
	log.Info("ShortcutDetectionAdapter running.")
	select {}
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ShortcutDetectionAdapter {
	adapter := &ShortcutDetectionAdapter{
		natsAdapter:    natsAdapter,
		shortcuts:      make(map[string]core.ShortcutEntry),
		pressedState:   make(map[string]bool),
		updateHookChan: make(chan struct{}, 1),
	}

	go func() {
		for range adapter.updateHookChan {
			adapter.updateKeyboardHook()
		}
	}()

	// Listen to unified backend config updates; update detector shortcuts only on explicit save
	backendSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE")
	adapter.natsAdapter.SubscribeToSubject(backendSubject, func(natsMessage *nats.Msg) {
		var payload struct {
			Shortcuts map[string]core.ShortcutEntry `json:"shortcuts"`
		}
		if err := json.Unmarshal(natsMessage.Data, &payload); err != nil {
			log.Error("Failed to decode backend config update: %v", err)
			return
		}
		// Apply shortcuts from full config
		if payload.Shortcuts == nil {
			payload.Shortcuts = make(map[string]core.ShortcutEntry)
		}
		adapter.shortcuts = payload.Shortcuts
		newPressedState := make(map[string]bool)
		for shortcutKey := range adapter.shortcuts {
			newPressedState[shortcutKey] = false
		}
		adapter.pressedState = newPressedState
		select {
		case adapter.updateHookChan <- struct{}{}:
		default:
		}
		log.Info("[ShortcutDetector] Applied shortcuts from unified config (%d entries)", len(adapter.shortcuts))
	})

	pressedEventSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	adapter.natsAdapter.SubscribeToSubject(pressedEventSubject, func(natsMessage *nats.Msg) {
		var eventData core.ShortcutPressed_Message
		if err := json.Unmarshal(natsMessage.Data, &eventData); err != nil {
			log.Error("NATS Listener: Failed to decode pressed event: %v", err)
		}
		// Optional: log.Debug("NATS Listener: Shortcut pressed event observed: %+v", eventData)
	})
	return adapter
}

func (adapter *ShortcutDetectionAdapter) updateKeyboardHook() {

	if adapter.hook != 0 {
		if core.UnhookWindowsHookEx != nil {
			core.UnhookWindowsHookEx.Call(uintptr(adapter.hook))
		}
		adapter.hook = 0
		adapter.keyboardHook = nil
	}
	if len(adapter.shortcuts) == 0 {
		return
	}

	adapter.keyboardHook = NewKeyboardHookForShortcuts(adapter.shortcuts, func(shortcutKeyIndex string, shortcutVKCodes []int, isPressedEvent bool) bool {
		shortcutIndexInt := 0
		fmt.Sscanf(shortcutKeyIndex, "%d", &shortcutIndexInt) // Assuming index is always numeric string.
		previousState := adapter.pressedState[shortcutKeyIndex]
		if isPressedEvent && !previousState {
			adapter.publishMessage(shortcutIndexInt, true)
			adapter.pressedState[shortcutKeyIndex] = true
		} else if !isPressedEvent && previousState {
			adapter.publishMessage(shortcutIndexInt, false)
			adapter.pressedState[shortcutKeyIndex] = false
		}
		return true
	})

	hookProcCallback := syscall.NewCallback(adapter.hookProc)
	if core.SetWindowsHookEx == nil {
		log.Fatal("CRITICAL Error: core.SetWindowsHookEx is nil!")
		return
	}

	hookHandle, _, errOriginal := core.SetWindowsHookEx.Call(uintptr(core.WH_KEYBOARD_LL), hookProcCallback, 0, 0)
	adapter.hook = syscall.Handle(hookHandle)

	if adapter.hook == 0 {
		log.Error("Failed to set keyboard hook: %v (GetLastError: %v)", errOriginal, syscall.GetLastError())
		return
	}

	go func() {
		var msg core.MSG
		if core.GetMessage == nil {
			log.Fatal("CRITICAL Error: GetMessage nil in msg loop!")
			return
		}
		for {
			core.GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		}
	}()
}

// hookProc is the callback for Windows keyboard events.
func (adapter *ShortcutDetectionAdapter) hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == 0 && adapter.keyboardHook != nil && adapter.keyboardHook.shortcuts != nil {
		keyboardHookStruct := (*core.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		eventVKCode := int(keyboardHookStruct.VKCode)
		eventFlags := keyboardHookStruct.Flags // Store flags for use in helpers

		isKeyDownEvent := wParam == core.WM_KEYDOWN || wParam == core.WM_SYSKEYDOWN
		isKeyUpEvent := wParam == core.WM_KEYUP || wParam == core.WM_SYSKEYUP

		// Filter auto-repeat events (where previous key state was also down).
		if (eventFlags & keyAutorepeatFlag) != 0 {
			if core.CallNextHookEx != nil {
				r1, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
				return r1
			}
			return 0
		}

		// Publish Escape key down so UI can close pie menu regardless of focus
		if isKeyDownEvent && eventVKCode == 0x1B { // VK_ESCAPE
			subject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_ESCAPE")
			if subject != "" {
				// Keep payload simple
				adapter.natsAdapter.PublishMessage(subject, map[string]any{"pressed": true})
				log.Debug("Published Escape keydown to %s", subject)
			} else {
				log.Warn("PUBLIC_NATSSUBJECT_PIEMENU_ESCAPE not set; skipping Escape publish")
			}
			// Do not consume the event globally; let it propagate
		}

		if isKeyDownEvent {
			if adapter.handleKeyDown(eventVKCode) { // Pass only eventVKCode
				return 1 // Event consumed
			}
		} else if isKeyUpEvent {
			adapter.handleKeyUp(eventVKCode) // Pass only eventVKCode
		}
	}
	if core.CallNextHookEx != nil {
		r1, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return r1
	}
	return 0
}

func (adapter *ShortcutDetectionAdapter) publishMessage(shortcutIndexInt int, isPressedEvent bool) {
	xPos, yPos, errMouse := core.GetMousePosition()
	if errMouse != nil {
		log.Error("Failed to get mouse position: %v", errMouse)
		xPos, yPos = 0, 0
	}
	shortcutLabel := ""
	stringifiedIndex := fmt.Sprintf("%d", shortcutIndexInt)
	if shortcutDetails, found := adapter.shortcuts[stringifiedIndex]; found {
		shortcutLabel = shortcutDetails.Label
	}
	outgoingMessage := core.ShortcutPressed_Message{ShortcutPressed: shortcutIndexInt, MouseX: xPos, MouseY: yPos, OpenSpecificPage: false, PageID: 0}
	actionString := "RELEASED"
	if isPressedEvent {
		actionString = "PRESSED"
	}
	natsSubject := ""
	if isPressedEvent {
		natsSubject = os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	} else {
		natsSubject = os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED")
	}
    log.Info("Publishing %s for shortcut %d (%s) at (%d, %d)", actionString, shortcutIndexInt, shortcutLabel, xPos, yPos)
	adapter.natsAdapter.PublishMessage(natsSubject, outgoingMessage)
}
