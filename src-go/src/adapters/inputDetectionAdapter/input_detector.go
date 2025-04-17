package inputDetectionAdapter

import (
	"fmt"
	"strings"
	"syscall"

	"unsafe"
)

const (
    WH_KEYBOARD_LL = 13
    WM_KEYDOWN     = 0x0100
)

const (
    WM_KEYUP     = 0x0101
    WM_SYSKEYUP  = 0x0105
)


type KBDLLHOOKSTRUCT struct {
    VKCode    uint32
    ScanCode  uint32
    Flags     uint32
    Time      uint32
    ExtraInfo uintptr
}

var (
    user32           = syscall.NewLazyDLL("user32.dll")
    setWindowsHookEx = user32.NewProc("SetWindowsHookExW")
    callNextHookEx   = user32.NewProc("CallNextHookEx")
    getMessage       = user32.NewProc("GetMessageW")
    hook            syscall.Handle
)

// KeyboardHook represents a keyboard hook instance
type KeyboardHook struct {
    callback   func(key int) bool
    isPressed  bool
    modKeys    []int
    targetKey  int
}

type point struct {
	X int32
	Y int32
}

func getMousePosition() (int, int, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getCursorPos := user32.NewProc("GetCursorPos")

	var pt point
	ret, _, err := getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, err
	}
	return int(pt.X), int(pt.Y), nil
}

// TODO: Make input detector work with more or no modifiers
func MyInputDetector(shortcut Shortcut) bool {
    hook := NewKeyboardHook(
        []int{shortcut[0], shortcut[1]}, // modifiers
        shortcut[2],                     // target key
        func(key int) bool {
			// Print the shortcut name
            printShortcut(shortcut[:])

            x, y, _ := getMousePosition()
	        fmt.Printf("Mouse position: x=%d, y=%d\n", x, y)
            
            // Publish the message to NATS
            publishMessage(1, MousePosition{X: x, Y: y})


			return true
        },
    )

    err := hook.Start()
    if err != nil {
        fmt.Printf("Error starting keyboard hook: %v\n", err)
        return false
    }

    return true
}

func NewKeyboardHook(modifiers []int, key int, callback func(key int) bool) *KeyboardHook {
    return &KeyboardHook{
        callback:  callback,
        modKeys:   modifiers,
        targetKey: key,
    }
}

func (kh *KeyboardHook) hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
    if nCode == 0 {
        kbd := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
        
        if wParam == WM_KEYDOWN {
            if int(kbd.VKCode) == kh.targetKey && kh.checkModifiers() {
                if !kh.isPressed {
                    kh.isPressed = true
                    if kh.callback != nil {
                        kh.callback(int(kbd.VKCode))
                    }
                }
                // Prevent the key event from being passed to other applications
                return 1
            }
        } else if wParam == WM_KEYUP || wParam == WM_SYSKEYUP {
            if int(kbd.VKCode) == kh.targetKey && kh.isPressed {
                kh.isPressed = false

                // Send a message here on shortcut release
                fmt.Printf("Shortcut released!\n")

                // Pusblish message with default mouse position
                publishMessage(0)
            }
        }
    }
    
    ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
    return ret
}

func (kh *KeyboardHook) Start() error {
    hookProc := syscall.NewCallback(kh.hookProc)
    
    ret, _, err := setWindowsHookEx.Call(
        uintptr(WH_KEYBOARD_LL),
        hookProc,
        0,
        0,
    )
    
    hook = syscall.Handle(ret)
    if hook == 0 {
        return fmt.Errorf("failed to set keyboard hook: %v", err)
    }

    var msg struct {
        hwnd   uintptr
        msg    uint32
        wParam uintptr
        lParam uintptr
        time   uint32
        pt     struct{ x, y int32 }
    }

    // Message loop
    for {
        getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
    }
}

const (
    VK_SHIFT   = 0x10
    VK_CONTROL = 0x11
    VK_ALT     = 0x12
    // Add any other virtual key codes you need
)

type Shortcut [3]int

func (kh *KeyboardHook) checkModifiers() bool {
    getKeyState := user32.NewProc("GetKeyState")
    for _, mod := range kh.modKeys {
        state, _, _ := getKeyState.Call(uintptr(mod))
        if (state & 0x8000) == 0 {
            return false
        }
    }
    return true
}

func FindKeyByValue(value int) string {
    for key, val := range KeyMap {
        if val == value {
            return key
        }
    }
    return ""
}

func printShortcut(shortcut []int) {
    shortcutNames := []string{}
    for _, keyValue := range shortcut {
        shortcutNames = append(shortcutNames, FindKeyByValue(keyValue))
    }
    shortcutString := strings.Join(shortcutNames, " + ")
    fmt.Printf("Shortcut %s pressed!\n", shortcutString)
}

func publishMessage(shortcutDetected int, mousePos ...MousePosition) {
    msg := EventMessage{
        ShortcutDetected: shortcutDetected,
    }

    if len(mousePos) > 0 {
        msg.MousePosition = mousePos[0]
    } else {
        msg.MousePosition = MousePosition{X: 100, Y: 100}
    }

    PublishMessage("mightyPie.events.pie_menu.open", msg)
    println("Message published to NATS")
}

var KeyMap = map[string]int{
    // Modifier keys
    "Shift": VK_SHIFT,
    "Ctrl":  VK_CONTROL,
    "Alt":   VK_ALT,
    "Win":    0x5B, // Left Windows key

    // Letters
    "A": 0x41, "B": 0x42, "C": 0x43, "D": 0x44, "E": 0x45,
    "F": 0x46, "G": 0x47, "H": 0x48, "I": 0x49, "J": 0x4A,
    "K": 0x4B, "L": 0x4C, "M": 0x4D, "N": 0x4E, "O": 0x4F,
    "P": 0x50, "Q": 0x51, "R": 0x52, "S": 0x53, "T": 0x54,
    "U": 0x55, "V": 0x56, "W": 0x57, "X": 0x58, "Y": 0x59,
    "Z": 0x5A,

    // Numbers (top row)
    "0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
    "5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,

    // Function keys
    "F1":  0x70, "F2":  0x71, "F3":  0x72, "F4":  0x73,
    "F5":  0x74, "F6":  0x75, "F7":  0x76, "F8":  0x77,
    "F9":  0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
	"F13": 0x7C, "F14": 0x7D, "F15": 0x7E, "F16": 0x7F,
    "F17": 0x80, "F18": 0x81, "F19": 0x82, "F20": 0x83,
    "F21": 0x84, "F22": 0x85, "F23": 0x86, "F24": 0x87,


    // Numpad keys
    "Num0": 0x60, "Num1": 0x61, "Num2": 0x62, "Num3": 0x63,
    "Num4": 0x64, "Num5": 0x65, "Num6": 0x66, "Num7": 0x67,
    "Num8": 0x68, "Num9": 0x69,
    "NumLock": 0x90, "Divide": 0x6F, "Multiply": 0x6A,
    "Subtract": 0x6D, "Add": 0x6B, "Decimal": 0x6E,

    // Arrow keys
    "Up":    0x26,
    "Down":  0x28,
    "Left":  0x25,
    "Right": 0x27,

    // Special keys
    "Esc":      0x1B,
    "Tab":      0x09,
    "CapsLock": 0x14,
    "Space":    0x20,
    "Enter":    0x0D,
    "Backspace": 0x08,
    "Delete":   0x2E,
    "Insert":   0x2D,
    "Home":     0x24,
    "End":      0x23,
    "PageUp":   0x21,
    "PageDown": 0x22,

    // Symbols
    "Semicolon": 0xBA, "Equal": 0xBB, "Comma": 0xBC,
    "Minus":     0xBD, "Period": 0xBE, "Slash": 0xBF,
    "Backtick":  0xC0, "BracketOpen": 0xDB,
    "Backslash": 0xDC, "BracketClose": 0xDD,
    "Quote":     0xDE,
}