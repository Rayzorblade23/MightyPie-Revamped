package mouseInputAdapter

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx     = user32.NewProc("SetWindowsHookExW")
	callNextHookEx       = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx  = user32.NewProc("UnhookWindowsHookEx")
	getMessage           = user32.NewProc("GetMessageW")

	mouseHook syscall.Handle
)

const (
	WH_MOUSE_LL = 14

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
)


func Run() {
	hookProc := syscall.NewCallback(mouseHookProc)
	h, _, _ := setWindowsHookEx.Call(uintptr(WH_MOUSE_LL), hookProc, 0, 0)
	mouseHook = syscall.Handle(h)

	defer unhookWindowsHookEx.Call(uintptr(mouseHook))

	var msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}
	for {
		println("Waiting for mouse input...")
		getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
	}
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == 0 {
		switch wParam {
		case WM_LBUTTONDOWN:
			handleLeftClick()
			return 1 // block
		case WM_RBUTTONDOWN:
			handleRightClick()
		}
	}
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// You can define these handlers however you want
func handleLeftClick() {
	fmt.Println("Left click detected and blocked")
}

func handleRightClick() {
	fmt.Println("Right click detected and blocked")
}
