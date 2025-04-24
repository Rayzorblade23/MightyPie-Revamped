package mouseInputAdapter

import (
	"encoding/json"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

const piemenuOpened_Subject = "mightyPie.events.piemenu.opened"

const piemenuClick_Subject = "mightyPie.events.piemenu.click"

type piemenuOpened_Message struct {
	PiemenuOpened bool `json:"piemenuOpened"`
}

type piemenuClick_Message struct {
	Click string `json:"click"`
}

type MouseInputAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New (natsAdapter *natsAdapter.NatsAdapter) *MouseInputAdapter {
	natsAdapter.SubscribeToSubject(piemenuOpened_Subject, func(msg *nats.Msg) {
		
		var message piemenuOpened_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			println("Failed to decode message: %v", err)
			return
		}
		
		fmt.Printf("Shortcut detected: %+v\n", message)
		
		if message.PiemenuOpened {
			SetMouseHookState(true)	
		} else if !message.PiemenuOpened {
			SetMouseHookState(false)	
		}
	})
	return &MouseInputAdapter{
		natsAdapter: natsAdapter,
	}
}

type MouseEvent struct {
    Button string // "left", "right", "middle"
    State  string // "down", "up"
}

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx     = user32.NewProc("SetWindowsHookExW")
	callNextHookEx       = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx  = user32.NewProc("UnhookWindowsHookEx")
	getMessage           = user32.NewProc("GetMessageW")

	mouseHook syscall.Handle
	hookEnabled bool
	adapter *MouseInputAdapter
)

const (
	WH_MOUSE_LL = 14

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
)


func (a *MouseInputAdapter) Run() {
	adapter = a

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
            adapter.handleClick("left", "down")
            return 1 // block
        case WM_LBUTTONUP:
            adapter.handleClick("left", "up")
            return 1 // block
        case WM_RBUTTONDOWN:
            adapter.handleClick("right", "down")
            return 1 // block
        case WM_RBUTTONUP:
            adapter.handleClick("right", "up")
            return 1 // block
        case WM_MBUTTONDOWN:
            adapter.handleClick("middle", "down")
            return 1 // block
        case WM_MBUTTONUP:
            adapter.handleClick("middle", "up")
            return 1 // block
        }
	}
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func (a *MouseInputAdapter) handleClick(button string, state string) {
    fmt.Printf("%s button %s detected and blocked!\n", button, state)
    a.publishMessage(MouseEvent{Button: button, State: state})
}

// Update publishMessage to handle the new MouseEvent type
func (a *MouseInputAdapter) publishMessage(event MouseEvent) {
    msg := piemenuClick_Message{
        Click: fmt.Sprintf("%s_%s", event.Button, event.State),
    }
    a.natsAdapter.PublishMessage(piemenuClick_Subject, msg)
    println("Message published to NATS")
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
