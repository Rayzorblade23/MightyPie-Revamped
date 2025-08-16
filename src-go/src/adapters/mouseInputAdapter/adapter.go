package mouseInputAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

// Package-level logger instance
var log = logger.New("MouseInput")

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

var (
	lastHeartbeatTime time.Time
	heartbeatTimer    *time.Ticker
	heartbeatDone     chan struct{}
)

func New(natsAdapter *natsAdapter.NatsAdapter) *MouseInputAdapter {
	a := &MouseInputAdapter{
		natsAdapter: natsAdapter,
	}
	natsAdapter.SubscribeToSubject(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_OPENED"), core.GetTypeName(a), func(msg *nats.Msg) {

		var message piemenuOpened_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			log.Error("Failed to decode message: %v", err)
			return
		}

		log.Debug("Pie Menu opened: %+v", message)

		// Set the mouse hook state based on the message
		// The heartbeat monitoring will be started/stopped in SetMouseHookState
		SetMouseHookState(message.PiemenuOpened)
	})
	
	// Subscribe to heartbeat messages
	natsAdapter.SubscribeToSubject(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_HEARTBEAT"), core.GetTypeName(a), func(msg *nats.Msg) {
		var heartbeat heartbeat_Message
		if err := json.Unmarshal(msg.Data, &heartbeat); err != nil {
			log.Error("Failed to decode heartbeat: %v", err)
			return
		}
		
		// Update the last heartbeat time
		lastHeartbeatTime = time.Now()
		log.Debug("Received heartbeat: %v", heartbeat.Timestamp)
	})
	
	return &MouseInputAdapter{
		natsAdapter: natsAdapter,
	}
}

// Start monitoring for missed heartbeats
func startHeartbeatMonitoring() {
    // Stop any existing monitoring
    if heartbeatTimer != nil {
        log.Debug("startHeartbeatMonitoring: pre-stop existing monitoring")
    }
    stopHeartbeatMonitoring()

    log.Debug("Starting heartbeat monitoring")

    // Initialize the last heartbeat time
    lastHeartbeatTime = time.Now()

    // Create a ticker that periodically checks for heartbeats
    ticker := time.NewTicker(HeartbeatCheckIntervalMs * time.Millisecond)
    done := make(chan struct{})
    // Publish to globals so stop can access
    heartbeatTimer = ticker
    heartbeatDone = done
    // ticker created

    // Start a goroutine to monitor heartbeats (capture locals to avoid nil deref)
    go func(t *time.Ticker, d chan struct{}) {
        for {
            select {
            case <-d:
                // goroutine exiting
                return
            case <-t.C:
                // Check if we've exceeded the timeout
                if hookEnabled && time.Since(lastHeartbeatTime) > time.Duration(HeartbeatTimeoutSeconds)*time.Second {
                    timeSinceLastHeartbeat := time.Since(lastHeartbeatTime)
                    log.Warn("No heartbeat received for %v seconds, disabling mouse hook as safety measure", timeSinceLastHeartbeat.Seconds())
                    
                    // Disable the mouse hook as a safety measure
                    SetMouseHookState(false)
                }
            }
        }
    }(ticker, done)
}

// Stop the heartbeat monitoring
func stopHeartbeatMonitoring() {
    if heartbeatTimer != nil {
        heartbeatTimer.Stop()
    }
    if heartbeatDone != nil {
        // Closing will signal the goroutine to exit; safe to close once
        close(heartbeatDone)
    }
    heartbeatTimer = nil
    heartbeatDone = nil
    log.Debug("Stopped heartbeat monitoring")
}

type MouseEvent struct {
	Button string // "left", "right", "middle"
	State  string // "down", "up"
}

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	getMessage          = user32.NewProc("GetMessageW")

	mouseHook   syscall.Handle
	hookEnabled bool
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
	if !hookEnabled {
		ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return ret
	}

	// Gate blocking by heartbeat freshness: if stale, stop blocking and disable hook
	if time.Since(lastHeartbeatTime) > time.Duration(HeartbeatTimeoutSeconds)*time.Second {
		if hookEnabled { // double-check
			log.Warn("Heartbeat stale (%.1fs). Releasing mouse hook immediately.", time.Since(lastHeartbeatTime).Seconds())
			SetMouseHookState(false)
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
	a.natsAdapter.PublishMessage(os.Getenv("PUBLIC_NATSSUBJECT_PIEMENU_CLICK"), "MouseInput", msg)
	log.Info("Mouse %s", msg.Click)
}

// setMouseHookState enables or disables the mouse hook
func SetMouseHookState(enable bool) {
    prev := hookEnabled
    hookEnabled = enable
    log.Debug("SetMouseHookState called. prev=%v new=%v", prev, enable)
    if hookEnabled {
        log.Debug("Mouse hook enabled")
        if heartbeatTimer == nil {
            log.Debug("Heartbeat monitor not running; starting now")
        } else {
            log.Debug("Heartbeat monitor already running; restarting to reset baseline")
        }
        // Reset heartbeat timer when hook is enabled
        startHeartbeatMonitoring()
    } else {
        log.Debug("Mouse hook disabled")
        // Stop heartbeat monitoring when hook is disabled
        stopHeartbeatMonitoring()
    }
}
