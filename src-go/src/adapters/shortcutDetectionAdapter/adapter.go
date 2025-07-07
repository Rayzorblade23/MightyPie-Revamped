// Production-Ready: Minimal Comments, Refactored hookProc, Debug hardcoding commented
package shortcutDetectionAdapter

import (
	"encoding/json"
	"fmt"
	"syscall"

	"unsafe"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

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

	setterUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	err := adapter.natsAdapter.SubscribeJetStreamPull(setterUpdateSubject, "", func(natsMessage *nats.Msg) {
		var receivedShortcuts map[string]core.ShortcutEntry
		if err := json.Unmarshal(natsMessage.Data, &receivedShortcuts); err != nil {
			fmt.Printf("Error: Failed to decode shortcuts update from JetStream: %v\n", err)
			return
		}

		fmt.Printf("Info: Received shortcuts update from JetStream: %+v\n", receivedShortcuts)

		adapter.shortcuts = receivedShortcuts
		newPressedState := make(map[string]bool)
		for shortcutKey := range adapter.shortcuts {
			newPressedState[shortcutKey] = false
		}
		adapter.pressedState = newPressedState
		select {
		case adapter.updateHookChan <- struct{}{}:
		default:
		}
	})

	if err != nil {
		fmt.Printf("Error: Failed to subscribe to JetStream subject: %v\n", err)
	}


	pressedEventSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	adapter.natsAdapter.SubscribeToSubject(pressedEventSubject, core.GetTypeName(adapter), func(natsMessage *nats.Msg) {
		var eventData shortcutPressed_Message
		if err := json.Unmarshal(natsMessage.Data, &eventData); err != nil {
			fmt.Printf("Error (NATS Listener): Failed to decode pressed event: %v\n", err)
		}
		// Optional: fmt.Printf("Info (NATS Listener): Shortcut pressed event observed: %+v\n", eventData)
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
		fmt.Println("CRITICAL Error: core.SetWindowsHookEx is nil!")
		return
	}

	hookHandle, _, errOriginal := core.SetWindowsHookEx.Call(uintptr(core.WH_KEYBOARD_LL), hookProcCallback, 0, 0)
	adapter.hook = syscall.Handle(hookHandle)

	if adapter.hook == 0 {
		fmt.Printf("Error: Failed to set keyboard hook: %v (GetLastError: %v)\n", errOriginal, syscall.GetLastError())
		return
	}

	go func() {
		var msg core.MSG
		if core.GetMessage == nil {
			fmt.Println("CRITICAL Error: GetMessage nil in msg loop!")
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

		// Always pass through injected events (from SendInput)
		const LLKHF_INJECTED = 0x00000010
		if (eventFlags & LLKHF_INJECTED) != 0 {
			if core.CallNextHookEx != nil {
				r1, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
				return r1
			}
			return 0
		}

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

type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}

func (adapter *ShortcutDetectionAdapter) publishMessage(shortcutIndexInt int, isPressedEvent bool) {
	xPos, yPos, errMouse := core.GetMousePosition()
	if errMouse != nil {
		fmt.Printf("Error: Failed to get mouse position: %v\n", errMouse)
		xPos, yPos = 0, 0
	}
	shortcutLabel := ""
	stringifiedIndex := fmt.Sprintf("%d", shortcutIndexInt)
	if shortcutDetails, found := adapter.shortcuts[stringifiedIndex]; found {
		shortcutLabel = shortcutDetails.Label
	}
	outgoingMessage := shortcutPressed_Message{ShortcutPressed: shortcutIndexInt, MouseX: xPos, MouseY: yPos}
	actionString := "RELEASED"
	if isPressedEvent {
		actionString = "PRESSED"
	}
	natsSubject := ""
	if isPressedEvent {
		natsSubject = env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	} else {
		natsSubject = env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED")
	}
	fmt.Printf("Publishing %s for shortcut %d (%s) at (%d, %d)\n", actionString, shortcutIndexInt, shortcutLabel, xPos, yPos)
	adapter.natsAdapter.PublishMessage(natsSubject, outgoingMessage)
}
