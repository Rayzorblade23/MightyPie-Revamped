package mouseInputAdapter

import (
	"encoding/json"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

const subject = "mightyPie.events.shortcut.detected"

type EventMessage struct {
	ShortcutDetected int `json:"shortcutDetected"`
}

type MouseInputAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New (natsAdapter *natsAdapter.NatsAdapter) *MouseInputAdapter {
	natsAdapter.SubscribeToSubject(subject, func(msg *nats.Msg) {
		
		var message EventMessage
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			println("Failed to decode message: %v", err)
			return
		}
		
		fmt.Printf("Shortcut detected: %+v", message)
		
		if message.ShortcutDetected == 1 {
			SetMouseHookState(true)	
		}
	})
	return &MouseInputAdapter{
		natsAdapter: natsAdapter,
	}
}


var (
	user32               = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx     = user32.NewProc("SetWindowsHookExW")
	callNextHookEx       = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx  = user32.NewProc("UnhookWindowsHookEx")
	getMessage           = user32.NewProc("GetMessageW")

	mouseHook syscall.Handle
	hookEnabled bool
)

const (
	WH_MOUSE_LL = 14

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
)


func (a *MouseInputAdapter) Run() {
	

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
	if !hookEnabled {
		ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return ret
	}

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
	fmt.Println("Left click detected and blocked!")
}

func handleRightClick() {
	fmt.Println("Right click detected and passed!")
}

// setMouseHookState enables or disables the mouse hook
func SetMouseHookState(enable bool) {
	hookEnabled = enable
	if hookEnabled {
		fmt.Println("Mouse hook enabled")
	} else {
		fmt.Println("Mouse hook disabled")
	}
}
