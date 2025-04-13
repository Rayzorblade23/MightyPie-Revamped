package inputDetectionAdapter

import (
	"fmt"
	"syscall"
)

var (
    user32               = syscall.NewLazyDLL("user32.dll")
    getAsyncKeyStateProc = user32.NewProc("GetAsyncKeyState")
)

// Map of key names to virtual key codes
var KeyMap = map[string]int{
    // Modifier keys
    "Ctrl":   0x11,
    "Shift":  0x10,
    "Alt":    0x12,
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


type KeyCodeEvalutaor func(vkCode int) bool

func IsKeyPressed(vkCode int) bool {
    // Call GetAsyncKeyState to check if the key is pressed
    ret, _, _ := getAsyncKeyStateProc.Call(uintptr(vkCode))
    return ret&0x8000 != 0
}

type Shortcut []int

var (
    isModAPressed = false
    isModBPressed = false
	isPressed = false
)

// TODO: Implement the use of more or no modifiers
func MyInputDetector(isKeyPressed_checker KeyCodeEvalutaor, shortcut Shortcut) bool {

	// for {
		// Check if the primary key (A) is pressed
		otherKey := isKeyPressed_checker(shortcut[2])

		// Handle shortcut press
		if otherKey && isModAPressed && isModBPressed {
			if !isPressed {
				fmt.Println("Shortcut pressed!")
			}
			isPressed = true
		} else {
			// Handle shortcut release
			if isPressed {
				fmt.Println("Shortcut released!")
			}
			isPressed = false
		}

		// Update modifier key states
		if otherKey {
			return isPressed
		}
		isModAPressed = isKeyPressed_checker(shortcut[0])
		isModBPressed = isKeyPressed_checker(shortcut[1])
	// }
    return false
}

