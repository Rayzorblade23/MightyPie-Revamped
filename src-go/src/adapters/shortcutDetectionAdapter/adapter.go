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

type ShortcutDetectionAdapter struct {
    natsAdapter  *natsAdapter.NatsAdapter
    keyboardHook *KeyboardHook
    hook         syscall.Handle
    shortcuts    map[string]ShortcutEntry // index as string -> ShortcutEntry
    pressedState map[string]bool
}

type ShortcutEntry struct {
	Codes []int  `json:"codes"`
	Label string `json:"label"`
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ShortcutDetectionAdapter {
    shortcutDetectionAdapter := &ShortcutDetectionAdapter{
        natsAdapter:  natsAdapter,
        shortcuts:    make(map[string]ShortcutEntry),
        pressedState: make(map[string]bool),
    }

	// Request the initial set of shortcuts
	requestUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_REQUEST_UPDATE")
	natsAdapter.PublishMessage(requestUpdateSubject, nil)

	// Subscribe to shortcut updates from the setter adapter
	setterUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	natsAdapter.SubscribeToSubject(setterUpdateSubject, core.GetTypeName(shortcutDetectionAdapter), func(msg *nats.Msg) {
		var shortcuts map[string]ShortcutEntry
		if err := json.Unmarshal(msg.Data, &shortcuts); err != nil {
			fmt.Printf("Failed to decode shortcuts update: %v\n", err)
			return
		}
		shortcutDetectionAdapter.shortcuts = shortcuts
		fmt.Printf("Updated shortcuts: %+v\n", shortcuts)
		shortcutDetectionAdapter.updateKeyboardHook()
	})

	subject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")

	natsAdapter.SubscribeToSubject(subject, core.GetTypeName(shortcutDetectionAdapter), func(msg *nats.Msg) {

		var message shortcutPressed_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			println("Failed to decode message: %v", err)
			return
		}

		fmt.Printf("[ShortcutDetector] Shortcut pressed: %+v\n", message)

	})

	return shortcutDetectionAdapter
}

func (a *ShortcutDetectionAdapter) updateKeyboardHook() {
    // Stop and unhook previous hook if needed
    if a.keyboardHook != nil && a.hook != 0 {
        core.UnhookWindowsHookEx.Call(uintptr(a.hook))
        a.hook = 0
        a.keyboardHook = nil
    }
    // Create a new KeyboardHook that checks all shortcuts
a.keyboardHook = NewKeyboardHookForShortcuts(a.shortcuts, func(index string, codes []int, pressed bool) bool {
    idxInt := 0
    fmt.Sscanf(index, "%d", &idxInt)

    // Only publish on state change
    prev := a.pressedState[index]
	if pressed && !prev {
		a.publishMessage(idxInt, codes, true)
		a.pressedState[index] = true
	} else if !pressed && prev {
		a.publishMessage(idxInt, codes, false)
		a.pressedState[index] = false
	}
    return true
})

    // Set the new hook
    hookProc := syscall.NewCallback(a.hookProc)
    ret, _, err := core.SetWindowsHookEx.Call(
        uintptr(core.WH_KEYBOARD_LL),
        hookProc,
        0,
        0,
    )
    a.hook = syscall.Handle(ret)
    if a.hook == 0 {
        fmt.Printf("Failed to set keyboard hook: %v\n", err)
        return
    }

    // Start the message loop in a goroutine
    go func() {
        var msg core.MSG
        for {
            core.GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        }
    }()
}



// New function to support multiple shortcuts
func NewKeyboardHookForShortcuts(shortcuts map[string]ShortcutEntry, callback func(index string, codes []int, pressed bool) bool) *KeyboardHook {
	return &KeyboardHook{
		multiCallback: callback,
		shortcuts:     shortcuts,
	}
}

type KeyboardHook struct {
    multiCallback func(index string, codes []int, pressed bool) bool
    shortcuts     map[string]ShortcutEntry
}

func getMousePosition() (int, int, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getCursorPos := user32.NewProc("GetCursorPos")

	var pt core.POINT
	ret, _, err := getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, err
	}
	return int(pt.X), int(pt.Y), nil
}

// In hookProc, check all shortcuts
func (a *ShortcutDetectionAdapter) hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
    if nCode == 0 && a.keyboardHook != nil && a.keyboardHook.shortcuts != nil {
        kbd := (*core.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
        var pressed bool
        if wParam == core.WM_KEYDOWN || wParam == core.WM_SYSKEYDOWN {
            pressed = true
        } else if wParam == core.WM_KEYUP || wParam == core.WM_SYSKEYUP {
            pressed = false
        } else {
            ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
            return uintptr(ret)
        }
        for idx, entry := range a.keyboardHook.shortcuts {
            if len(entry.Codes) < 2 {
                continue
            }
            mainKey := entry.Codes[len(entry.Codes)-1]
            modifiers := entry.Codes[:len(entry.Codes)-1]
            if int(kbd.VKCode) == mainKey && checkModifiers(modifiers) {
                if a.keyboardHook.multiCallback != nil {
                    a.keyboardHook.multiCallback(idx, entry.Codes, pressed)
                }
                return 1
            }
        }
    }
    ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
    return uintptr(ret)
}

func checkModifiers(modifiers []int) bool {
	getKeyState := core.User32.NewProc("GetKeyState")
	for _, mod := range modifiers {
		state, _, _ := getKeyState.Call(uintptr(mod))
		if (state & 0x8000) == 0 {
			return false
		}
	}
	return true
}

type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}

func (a *ShortcutDetectionAdapter) publishMessage(shortcutPressed int, codes []int, pressed bool) {
    x, y, err := getMousePosition()
    if err != nil {
        fmt.Printf("Failed to get mouse position: %v\n", err)
        x, y = 0, 0
    }

    // Find the label from the shortcuts map
    label := ""
    indexStr := fmt.Sprintf("%d", shortcutPressed)
    if entry, ok := a.shortcuts[indexStr]; ok {
        label = entry.Label
    }

    msg := shortcutPressed_Message{
        ShortcutPressed: shortcutPressed,
        MouseX:          x,
        MouseY:          y,
    }

    if pressed {
        fmt.Printf("Publishing PRESSED for shortcut %d (%s) at (%d, %d)\n", shortcutPressed, label, x, y)
        a.natsAdapter.PublishMessage(env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED"), msg)
    } else {
        fmt.Printf("Publishing RELEASED for shortcut %d (%s) at (%d, %d)\n", shortcutPressed, label, x, y)
        a.natsAdapter.PublishMessage(env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED"), msg)
    }
}