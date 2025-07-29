package windowManagementAdapter

import (
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// NewWindowWatcher creates a new window watcher
func NewWindowWatcher() *WindowWatcher {
	return &WindowWatcher{
		stopChan:       make(chan struct{}),
		changeDetected: make(chan struct{}, 1),
		lastEventTime:  time.Now(),
	}
}

// Package-level callback using windows.NewCallback
var winEventProcCallback = windows.NewCallback(func(hWinEventHook windows.Handle, event uint32,
	hwnd windows.HWND, idObject int32, idChild int32,
	idEventThread uint32, dwmsEventTime uint32) uintptr {

	// Access the global watcher safely
	activeWatcherMutex.RLock()
	watcher := activeWindowWatcher
	activeWatcherMutex.RUnlock()

	if watcher == nil {
		log.Debug("[CALLBACK DEBUG] !!! No activeWindowWatcher !!!")
		return 0
	}

	// Filtering
	if idObject != int32(OBJID_WINDOW) || idChild != CHILDID_SELF {
		return 0
	}
	rootOwner, _, _ := procGetAncestor.Call(uintptr(hwnd), GA_ROOTOWNER)
	if uintptr(hwnd) != rootOwner {
		return 0
	}
	isVisibleRet, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	if event == EVENT_OBJECT_SHOW && isVisibleRet == 0 {
		return 0
	}
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 { // Filter empty titles for both show and hide
		return 0
	}

	if event == EVENT_OBJECT_NAMECHANGE {
    // Silenced debug prints for title change detection and signal sent/skipped
    // fmt.Printf("[CALLBACK DEBUG] Title change detected for HWND: 0x%X\n", hwnd)

    // Perform additional actions here, such as updating the window list
    watcher.mutex.Lock()
    watcher.lastEventTime = time.Now()
    watcher.mutex.Unlock()

    select {
    case watcher.changeDetected <- struct{}{}:
        // fmt.Printf("[CALLBACK DEBUG] Title change signal sent for HWND: 0x%X\n", hwnd)
    default:
        // fmt.Printf("[CALLBACK DEBUG] Title change signal skipped (channel busy) for HWND: 0x%X\n", hwnd)
    }
}

	// fmt.Printf("[CALLBACK DEBUG] Filter PASSED! Event=0x%X, Hwnd=0x%X\n", event, hwnd)

	// Debounce and signal logic
	watcher.mutex.RLock() // Use RLock for reading lastEventTime
	lastEvent := watcher.lastEventTime
	watcher.mutex.RUnlock()

	now := time.Now()
	const callbackDebounceDuration = 1000 * time.Millisecond
	if now.Sub(lastEvent) > callbackDebounceDuration {
		watcher.mutex.Lock() // Lock for writing lastEventTime
		watcher.lastEventTime = now
		watcher.mutex.Unlock()

		select {
		case watcher.changeDetected <- struct{}{}:
			// fmt.Printf("[CALLBACK DEBUG] --> Sent signal. Event=0x%X, Hwnd=0x%X\n", event, hwnd)
		default:
			// fmt.Printf("[CALLBACK DEBUG] !! changeDetected channel BUSY. Event=0x%X, Hwnd=0x%X\n", event, hwnd)
		}
	} else {
		// fmt.Printf("[CALLBACK DEBUG] Debounced! Event=0x%X, Hwnd=0x%X\n", event, hwnd)
	}

	return 0
})

// Start launches the dedicated hook-setting and message loop goroutine
func (w *WindowWatcher) Start() error {
	w.mutex.Lock() // Lock for checking/setting isRunning
	if w.isRunning {
		w.mutex.Unlock()
		log.Info("[WATCHER START] Already running.")
		return nil
	}

	// Safely set the active window watcher
	activeWatcherMutex.Lock()
	if activeWindowWatcher != nil {
		activeWatcherMutex.Unlock()
		w.mutex.Unlock()
		log.Error("Another WindowWatcher is already active")
		return fmt.Errorf("another WindowWatcher is already active")
	}
	activeWindowWatcher = w
	activeWatcherMutex.Unlock()

	w.isRunning = true
	w.mutex.Unlock() // Unlock before launching goroutine

	go w.hookAndMessageLoop() // Launch the combined function

	return nil
}

// hookAndMessageLoop sets the hook and runs the message loop on the same thread
func (w *WindowWatcher) hookAndMessageLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Set the hook WITHIN this goroutine
	hookCallbackPtr := winEventProcCallback

hook, _, err := procSetWinEventHook.Call(
    uintptr(EVENT_OBJECT_SHOW),       // Start of event range
    uintptr(EVENT_OBJECT_NAMECHANGE), // End of event range
    0, uintptr(hookCallbackPtr), 0, 0,
    uintptr(WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS),
)

	hookErrString := ""
	if err != nil {
		hookErrString = err.Error()
	}

	if hook == 0 || (hookErrString != "" && !strings.Contains(hookErrString, "operation completed successfully")) {
		w.mutex.Lock()
		w.isRunning = false
		
		activeWatcherMutex.Lock()
		if activeWindowWatcher == w {
			activeWindowWatcher = nil
		}
		activeWatcherMutex.Unlock()
		
		w.mutex.Unlock()
		return
	}

	w.mutex.Lock()
	w.eventHook = HWINEVENTHOOK(hook)
	w.mutex.Unlock()

	// Message Loop
	var loopThreadId uint32 = 0
	retTID, _, _ := procGetCurrentThreadId.Call()
	loopThreadId = uint32(retTID)

	go func(tid uint32) {
		<-w.stopChan
		if tid != 0 {
			ret, _, postErr := procPostThreadMessageW.Call(uintptr(tid), uintptr(WM_QUIT), 0, 0)
			if ret == 0 {
				log.Error("[HOOK LOOP] PostThreadMessageW(WM_QUIT) FAILED. Error: %v", postErr)
			} else {
				log.Info("[HOOK LOOP] WM_QUIT posted via PostThreadMessageW.")
			}
		} else {
			log.Error("[HOOK LOOP] Cannot post WM_QUIT, Thread ID is 0")
		}
	}(loopThreadId)

	// Use locally defined MSG struct
	var msg MSG
	for {
		ret, _, getMsgErr := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0)

		getMsgRet := int32(ret)
		if getMsgRet == -1 {
			log.Error("[HOOK LOOP] GetMessageW ERROR: Ret=-1, Error=%v", getMsgErr)
			break
		}
		if getMsgRet == 0 {
			log.Info("[HOOK LOOP] GetMessageW received WM_QUIT (Ret=0). Exiting loop.")
			break
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	// Unhook logic
	w.mutex.Lock()
	currentHook := w.eventHook
	w.eventHook = 0
	w.isRunning = false
	
	activeWatcherMutex.Lock()
	if activeWindowWatcher == w {
		activeWindowWatcher = nil
	}
	activeWatcherMutex.Unlock()
	
	w.mutex.Unlock()

	if currentHook != 0 {
		ret, _, unhookErr := procUnhookWinEvent.Call(uintptr(currentHook))
		if ret == 0 {
			log.Error("[HOOK LOOP] UnhookWinEvent FAILED: Error=%v", unhookErr)
		} 
	}
}

// Stop signals the hookAndMessageLoop goroutine to exit
func (w *WindowWatcher) Stop() {
	w.mutex.Lock() // Lock for reading/closing stopChan safely
	log.Info("[WATCHER STOP] Stop called.")
	if !w.isRunning {
		log.Info("[WATCHER STOP] Hook loop not running.")
		w.mutex.Unlock()
		return
	}

	select {
	case <-w.stopChan:
		log.Info("[WATCHER STOP] Stop channel already closed.")
	default:
		close(w.stopChan)
		log.Info("[WATCHER STOP] Closed stop channel.")
	}
	w.mutex.Unlock() // Unlock after closing channel

	log.Info("[WATCHER STOP] Stop signal sent.")
}

// GetChangeDetectedChannel returns the channel that signals window changes
func (w *WindowWatcher) GetChangeDetectedChannel() <-chan struct{} {
	return w.changeDetected
}