package core

import (
	"syscall"
	"unsafe"
)

func FindKeyByValue(value int) string {
	for key, val := range KeyMap {
		if val == value {
			return key
		}
	}
	return ""
}

func IsModifier(key int) bool {
	switch key {
	case 0x10, // VK_SHIFT
		0xA0, // VK_LSHIFT
		0xA1, // VK_RSHIFT
		0x11, // VK_CONTROL
		0xA2, // VK_LCONTROL
		0xA3, // VK_RCONTROL
		0x12, // VK_ALT
		0xA4, // VK_LMENU (Left Alt)
		0xA5, // VK_RMENU (Right Alt)
		0x5B, // VK_LWIN
		0x5C: // VK_RWIN
		return true
	}
	return false
}

func GetMousePosition() (int, int, error) {
	var pt POINT
	retValue, _, errSyscall := GetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if retValue == 0 {
		return 0, 0, errSyscall
	}
	return int(pt.X), int(pt.Y), nil
}

type POINT struct {
    X int32
    Y int32
}

type MSG struct {
    Hwnd   uintptr
    Msg    uint32
    WParam uintptr
    LParam uintptr
    Time   uint32
    Pt     POINT
}

const (
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_ALT     = 0x12
	VK_MENU    = 0x12 // Alt
	VK_LWIN    = 0x5B
	VK_RWIN    = 0x5C
)

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1
	INPUT_HARDWARE = 2

	KEYEVENTF_EXTENDEDKEY = 0x0001
	KEYEVENTF_KEYUP       = 0x0002
)

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type HARDWAREINPUT struct {
	UMsg    uint32
	WParamL uint16
	WParamH uint16
}

type INPUT struct {
	Type    uint32
	_       [4]byte    // explicit padding to align Ki to 8-byte boundary (x64)
	Ki      KEYBDINPUT // 24 bytes
	_       [8]byte    // explicit padding to reach 40 bytes total (Windows x64 expects sizeof(INPUT)==40)
}

var (
	SendInput = User32.NewProc("SendInput")
)

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105
)

type ShortcutEntry struct {
	Codes []int  `json:"codes"`
	Label string `json:"label"`
}

type KBDLLHOOKSTRUCT struct {
	VKCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

var (
	User32              = syscall.NewLazyDLL("user32.dll")
	SetWindowsHookEx    = User32.NewProc("SetWindowsHookExW")
	CallNextHookEx      = User32.NewProc("CallNextHookEx")
	UnhookWindowsHookEx = User32.NewProc("UnhookWindowsHookEx")
	PostQuitMessage     = User32.NewProc("PostQuitMessage")

	// Keyboard/Mouse state - ADD THESE
	GetKeyState        = User32.NewProc("GetKeyState")
	GetCursorPos       = User32.NewProc("GetCursorPos")

	// Clipboard
	OpenClipboard      = User32.NewProc("OpenClipboard")
	CloseClipboard     = User32.NewProc("CloseClipboard")

	// Message loop - ADD/MODIFY THESE
	GetMessage          = User32.NewProc("GetMessageW")
	TranslateMessage   = User32.NewProc("TranslateMessage")
	DispatchMessage    = User32.NewProc("DispatchMessageW")
)

var KeyMap = map[string]int{
	// Modifier keys
	"Shift": VK_SHIFT,
	"Ctrl":  VK_CONTROL,
	"Alt":   VK_ALT,
	"Win":   0x5B, // Left Windows key

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
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
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
	"Esc":       0x1B,
	"Tab":       0x09,
	"CapsLock":  0x14,
	"Space":     0x20,
	"Enter":     0x0D,
	"Backspace": 0x08,
	"Delete":    0x2E,
	"Insert":    0x2D,
	"Home":      0x24,
	"End":       0x23,
	"PageUp":    0x21,
	"PageDown":  0x22,

	// Symbols
	"Semicolon": 0xBA, "Equal": 0xBB, "Comma": 0xBC,
	"Minus": 0xBD, "Period": 0xBE, "Slash": 0xBF,
	"Backtick": 0xC0, "BracketOpen": 0xDB,
	"Backslash": 0xDC, "BracketClose": 0xDD,
	"Quote": 0xDE,
}
