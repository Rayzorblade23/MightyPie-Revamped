package mouseInputAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/go-vgo/robotgo"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("MouseInputHandler")

type piemenuOpened_Message struct {
	PiemenuOpened bool `json:"piemenuOpened"`
}

type piemenuClick_Message struct {
	Click string `json:"click"`
}

type heartbeat_Message struct {
	Timestamp int64 `json:"timestamp"`
}

type MouseInputAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

const (
	// Maximum time to wait for a heartbeat before disabling the hook
	HeartbeatTimeoutSeconds = 9 // Allow for some network delays

	// How often to check for missed heartbeats (more efficient than resetting timer)
	HeartbeatCheckIntervalMs = 500 // Check every 500ms
)

// lastHeartbeatUnix stores the last heartbeat time as UnixNano for atomic access across goroutines
var lastHeartbeatUnix int64

// control messages to a single manager goroutine that owns state/ticker
type controlMsgKind int

const (
	msgSetOpen controlMsgKind = iota
	msgHeartbeat
)

type controlMsg struct {
	kind    controlMsgKind
	open    bool
	hbTime  time.Time
	reason  string
}

var controlCh chan controlMsg

func New(natsAdapter *natsAdapter.NatsAdapter) *MouseInputAdapter {

	// initialize control channel and start manager goroutine once
	if controlCh == nil {
		controlCh = make(chan controlMsg, 32)
		go stateManager()
	}

	natsAdapter.SubscribeToSubject(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_OPENED"), func(msg *nats.Msg) {

		var message piemenuOpened_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			log.Error("Failed to decode message: %v", err)
			return
		}

		if message.PiemenuOpened {
			log.Debug("Pie Menu opened!")
		} else {
			log.Debug("Pie Menu closed!")
		}

		// Serialize state change via manager goroutine
		controlCh <- controlMsg{kind: msgSetOpen, open: message.PiemenuOpened, reason: "NATS opened msg"}
	})

	// Subscribe to heartbeat messages
	natsAdapter.SubscribeToSubject(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_HEARTBEAT"), func(msg *nats.Msg) {
		var heartbeat heartbeat_Message
		if err := json.Unmarshal(msg.Data, &heartbeat); err != nil {
			log.Error("Failed to decode heartbeat: %v", err)
			return
		}

		// Send heartbeat to manager goroutine; also update atomic for hook fast-path
		now := time.Now()
		atomic.StoreInt64(&lastHeartbeatUnix, now.UnixNano())
		controlCh <- controlMsg{kind: msgHeartbeat, hbTime: now}
		log.Debug("Received heartbeat: %v", heartbeat.Timestamp)
	})

	return &MouseInputAdapter{
		natsAdapter: natsAdapter,
	}
}

// stateManager is the single goroutine that owns hook state and heartbeat ticker
func stateManager() {
	var enabled bool
	var ticker *time.Ticker

	stopTicker := func() {
		if ticker != nil {
			ticker.Stop()
			ticker = nil
		}
	}

	for {
		select {
		case msg := <-controlCh:
			switch msg.kind {
			case msgSetOpen:
				if msg.open {
					if !enabled {
						enabled = true
						atomic.StoreUint32(&hookEnabledFlag, 1)
						// reset heartbeat baseline
						now := time.Now()
						atomic.StoreInt64(&lastHeartbeatUnix, now.UnixNano())
						if ticker == nil {
							ticker = time.NewTicker(HeartbeatCheckIntervalMs * time.Millisecond)
						}
						log.Debug("Mouse hook enabled (manager). reason=%s", msg.reason)
					}
				} else {
					if enabled {
						enabled = false
						atomic.StoreUint32(&hookEnabledFlag, 0)
						stopTicker()
						log.Debug("Mouse hook disabled (manager). reason=%s", msg.reason)
					}
				}
			case msgHeartbeat:
				// already updated atomic in subscriber; no-op needed here beyond optional diagnostics
			}
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			// return a typed nil channel that never fires when no ticker
			var nilCh <-chan time.Time
			return nilCh
		}():
			if enabled {
				ns := atomic.LoadInt64(&lastHeartbeatUnix)
				if ns == 0 {
					// no heartbeat seen yet; allow grace period handled by initial baseline
					continue
				}
				if time.Since(time.Unix(0, ns)) > time.Duration(HeartbeatTimeoutSeconds)*time.Second {
					log.Warn("No heartbeat received within timeout, disabling mouse hook as safety measure (manager)")
					enabled = false
					atomic.StoreUint32(&hookEnabledFlag, 0)
					stopTicker()
				}
			}
		}
	}
}

type MouseEvent struct {
	Button string // "left", "right", "middle"
	State  string // "down", "up"
}

// Local definitions for low-level mouse hook data structures
type point struct {
	x int32
	y int32
}

type msllHookStruct struct {
	pt          point
	mouseData   uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	getMessage          = user32.NewProc("GetMessageW")

	mouseHook   syscall.Handle
	// hookEnabledFlag is read by the hook callback; only written by the manager or a fast-path disable
	hookEnabledFlag uint32 // 0=false, 1=true
	adapter     *MouseInputAdapter
)

const (
	WH_MOUSE_LL = 14

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
	WM_MOUSEWHEEL  = 0x020A
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
		log.Debug("Waiting for mouse input...")
		getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
	}
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if atomic.LoadUint32(&hookEnabledFlag) == 0 {
		ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return ret
	}

	// Gate blocking by heartbeat freshness: if stale, stop blocking and disable hook
	if ns := atomic.LoadInt64(&lastHeartbeatUnix); ns > 0 && time.Since(time.Unix(0, ns)) > time.Duration(HeartbeatTimeoutSeconds)*time.Second {
		if atomic.LoadUint32(&hookEnabledFlag) == 1 { // double-check
			log.Warn("Heartbeat stale (%.1fs). Releasing mouse hook immediately.", time.Since(time.Unix(0, ns)).Seconds())
			// Fast-path: immediately flip atomic flag to stop blocking, and notify manager non-blocking
			atomic.StoreUint32(&hookEnabledFlag, 0)
			select {
			case controlCh <- controlMsg{kind: msgSetOpen, open: false, reason: "stale from hookProc"}:
			default:
				// drop if manager busy; it'll also disable on next tick
			}
		}
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
		case WM_MOUSEWHEEL:
			// Determine wheel direction from MSLLHOOKSTRUCT.mouseData high word (signed)
			ms := (*msllHookStruct)(unsafe.Pointer(lParam))
			delta := int16((ms.mouseData >> 16) & 0xFFFF)
			if delta > 0 {
				// Wheel up -> Volume Up
				if err := robotgo.KeyTap("audio_vol_up"); err != nil {
					log.Error("robotgo.KeyTap audio_vol_up failed: %v", err)
				} else {
					log.Debug("Wheel up detected -> Volume Up")
				}
			} else if delta < 0 {
				// Wheel down -> Volume Down
				if err := robotgo.KeyTap("audio_vol_down"); err != nil {
					log.Error("robotgo.KeyTap audio_vol_down failed: %v", err)
				} else {
					log.Debug("Wheel down detected -> Volume Down")
				}
			}
			return 1 // block scroll while pie menu open
		}
	}
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func (a *MouseInputAdapter) handleClick(button string, state string) {
	log.Debug("%s button %s detected and blocked!", button, state)
	a.publishMessage(MouseEvent{Button: button, State: state})
}

// Update publishMessage to handle the new MouseEvent type
func (a *MouseInputAdapter) publishMessage(event MouseEvent) {
	msg := piemenuClick_Message{
		Click: fmt.Sprintf("%s_%s", event.Button, event.State),
	}
	a.natsAdapter.PublishMessage(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_CLICK"), msg)
	log.Debug("Mouse %s", msg.Click)
}

// SetMouseHookState requests enabling/disabling of the hook via the manager goroutine.
// When disabling, we also flip the atomic flag immediately to stop blocking in the hook callback.
func SetMouseHookState(enable bool) {
	if enable {
		controlCh <- controlMsg{kind: msgSetOpen, open: true, reason: "SetMouseHookState"}
	} else {
		atomic.StoreUint32(&hookEnabledFlag, 0)
		select {
		case controlCh <- controlMsg{kind: msgSetOpen, open: false, reason: "SetMouseHookState"}:
		default:
		}
	}
}
