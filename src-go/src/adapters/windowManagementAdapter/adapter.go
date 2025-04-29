package windowManagementAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	// Use windows package for callback and types where appropriate
	"golang.org/x/sys/windows"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	// Use lxn/win for HWND type consistency if desired, otherwise use windows.HWND
	"github.com/lxn/win" // Assuming you want to keep this for HWND
	"github.com/nats-io/nats.go"
)

// --- Structs remain the same ---
type WindowManagementAdapter struct {
	natsAdapter   *natsAdapter.NatsAdapter
	winManager    *WindowManager
	stopChan      chan struct{} // Adapter's overall stop
	windowWatcher *WindowWatcher
}
type shortcutPressed_Message struct {
	ShortcutPressed int `json:"shortcutPressed"`
	MouseX          int `json:"mouseX"`
	MouseY          int `json:"mouseY"`
}
type WindowInfo struct {
	Title    string
	ExeName  string
	Instance int
}
type WindowMapping map[win.HWND]WindowInfo // Using lxn/win HWND
type WindowManager struct {
	openWindowsInfo WindowMapping
	mutex           sync.RWMutex
}
type WindowEvents struct {
	WindowsChanged bool              `json:"windowsChanged"`
	Windows        map[string]string `json:"windows"`
}

// Use windows.Handle for hooks, consistent with windows package
type HWINEVENTHOOK windows.Handle

type WindowWatcher struct {
	mutex          sync.RWMutex // Use RWMutex if reading hook handle elsewhere
	eventHook      HWINEVENTHOOK
	changeDetected chan struct{}
	stopChan       chan struct{} // Watcher's specific stop
	lastEventTime  time.Time
	// winEventProc uintptr // No longer need to store callback pointer here
	isRunning bool // Track if the hook loop goroutine is active
}

// Manually defined MSG struct (matching C struct)
type MSG struct {
	HWnd    windows.HWND // Use windows.HWND type
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      Point // Defined below
	// lPrivate uint32 // Typically not needed/used in Go message loops
}

// Manually defined POINT struct used within MSG
type Point struct {
	X, Y int32
}

// --- Global variables / Constants ---
var (
	user32   = windows.NewLazySystemDLL("user32.dll") // Use windows package DLL loading
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	// Load procs using windows package style
	procSetWinEventHook            = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent             = user32.NewProc("UnhookWinEvent")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW         = user32.NewProc("PostThreadMessageW") // Need this for clean exit
	procGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procGetClassNameW              = user32.NewProc("GetClassNameW")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procDwmGetWindowAttribute      = dwmapi.NewProc("DwmGetWindowAttribute")
	// procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	// procOpenProcess                = kernel32.NewProc("OpenProcess")
	// procCloseHandle                = kernel32.NewProc("CloseHandle")

	hwndToExclude      []win.HWND // Use lxn/win HWND consistently if desired
	excludedClassNames = map[string]bool{"Progman": true, "AutoHotkeyGUI": true, "RainmeterMeterWindow": true}
	logger             = log.New(os.Stdout, "[WindowManager] ", log.LstdFlags)

	activeWindowWatcher *WindowWatcher // Still use this for callback access
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	EVENT_OBJECT_SHOW                 = 0x8002
	EVENT_OBJECT_HIDE                 = 0x8003
	WINEVENT_OUTOFCONTEXT             = 0x0000
	WINEVENT_SKIPOWNPROCESS           = 0x0002 // Changed back
	OBJID_WINDOW                      = 0      // Use 0 directly
	CHILDID_SELF                      = 0
	GA_ROOTOWNER                      = 3
	WM_QUIT                           = 0x0012
	DWMWA_CLOAKED                     = 14 // Standard name for cloaked state attribute
	MAX_PATH                          = 260
)

// --- Adapter New, Run, Stop methods ---
func New(natsAdapter *natsAdapter.NatsAdapter) *WindowManagementAdapter {
	winManager := &WindowManager{openWindowsInfo: make(WindowMapping)}
	changeDetected := make(chan struct{}, 1) // Buffered channel might be good
	windowWatcher := &WindowWatcher{
		stopChan:       make(chan struct{}),
		changeDetected: changeDetected,
		lastEventTime:  time.Now(),
	}
	a := &WindowManagementAdapter{
		natsAdapter:   natsAdapter,
		winManager:    winManager,
		stopChan:      make(chan struct{}),
		windowWatcher: windowWatcher,
	}

	// NATS Subscription... (keep as is)
	natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_SHORTCUT_PRESSED"), func(msg *nats.Msg) {
		var message shortcutPressed_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			fmt.Printf("Failed to decode command: %v\n", err)
			return
		}
		fmt.Printf("Shortcut pressed: %d\n", message.ShortcutPressed)
		windows := a.GetFilteredListOfWindows(0)
		fmt.Println("\nCurrent Window Mapping:")
		for hwnd, info := range windows {
			fmt.Printf("  HWND: %v, Title: %s, Exe: %s, Instance: %d\n", hwnd, info.Title, info.ExeName, info.Instance)
		}
	})

	return a
}

func (a *WindowManagementAdapter) Run() error {
	fmt.Println("WindowManagementAdapter starting...")
	initialWindows := a.GetFilteredListOfWindows(0)
	a.winManager.UpdateOpenWindowsInfo(initialWindows)
	logger.Println("Initial window list created with", len(initialWindows), "windows")

	// Start the window watcher (which now launches the hook+loop goroutine)
	err := a.windowWatcher.Start()
	if err != nil {
		return fmt.Errorf("failed to start window watcher: %w", err)
	}

	// Start the separate window monitoring goroutine
	go a.monitorWindows()
	fmt.Println("WindowManagementAdapter running.")

	// Wait until adapter is stopped
	<-a.stopChan
	fmt.Println("WindowManagementAdapter received stop signal.")

	// Clean up: Stop the watcher first
	a.windowWatcher.Stop() // Ensure this signals the hook loop to exit

	fmt.Println("WindowManagementAdapter finished.")
	return nil
}

func (a *WindowManagementAdapter) Stop() {
	fmt.Println("[ADAPTER STOP] Stopping WindowManagementAdapter...")
	// Signal the main Run loop to exit FIRST
	select {
	case <-a.stopChan:
		// Already closed
		fmt.Println("[ADAPTER STOP] Adapter stopChan already closed.")
	default:
		close(a.stopChan)
		fmt.Println("[ADAPTER STOP] Closed adapter stopChan.")
	}

	// Then signal the watcher to stop its loop and unhook
	if a.windowWatcher != nil {
		fmt.Println("[ADAPTER STOP] Calling windowWatcher.Stop()...")
		a.windowWatcher.Stop()
	} else {
		fmt.Println("[ADAPTER STOP] windowWatcher is nil.")
	}
	fmt.Println("[ADAPTER STOP] WindowManagementAdapter stopped signal sent.")
}

// monitorWindows monitors window changes and updates accordingly
func (a *WindowManagementAdapter) monitorWindows() {
	// Store previous window state for comparison
	var previousWindows WindowMapping

	// Track last update time to throttle updates
	var lastUpdateTime time.Time

	// Set minimum interval between updates (1 second)
	const minUpdateInterval = 1 * time.Second

	// Timer to handle deferred updates
	var updateTimer *time.Timer
	var updatePending bool

	for {
		select {
		case <-a.stopChan:
			// Stop any pending timer
			if updateTimer != nil {
				updateTimer.Stop()
			}
			return

		case <-a.windowWatcher.changeDetected:
			// A window change was detected

			// Check if we need to throttle
			timeSinceLastUpdate := time.Since(lastUpdateTime)
			if timeSinceLastUpdate < minUpdateInterval {
				// Too soon for another update
				if !updatePending {
					// Schedule a deferred update
					delay := minUpdateInterval - timeSinceLastUpdate
					if updateTimer == nil {
						updateTimer = time.NewTimer(delay)
					} else {
						updateTimer.Reset(delay)
					}
					updatePending = true

					go func() {
						<-updateTimer.C
						// Signal that it's time to process the deferred update
						select {
						case a.windowWatcher.changeDetected <- struct{}{}:
							// Signal sent
						default:
							// Channel full, no need to send again
						}
						updatePending = false
					}()
				}
				// Skip immediate update
				continue
			}

			// Time to update - process changes
			currentWindows := a.GetFilteredListOfWindows(0)

			// Check if windows have actually changed
			if !reflect.DeepEqual(currentWindows, previousWindows) {
				a.winManager.UpdateOpenWindowsInfo(currentWindows)
				previousWindows = currentWindows

				// Update the timestamp
				lastUpdateTime = time.Now()
				logger.Println("Windows list updated due to detected change")

				// Print the window list whenever it is updated
				fmt.Println("Updated Window List:")
				for hwnd, info := range currentWindows {
					fmt.Printf("Window Handle: %v\n", hwnd)
					fmt.Printf("  Title: %s\n", info.Title)
					fmt.Printf("  ExeName: %s\n", info.ExeName)
					fmt.Printf("  Instance: %d\n", info.Instance)
					fmt.Println()
				}
			}
		}
	}
}

// --- Window Watcher ---

// Package-level callback using windows.NewCallback and correct signature
var winEventProcCallback = windows.NewCallback(func(hWinEventHook windows.Handle, event uint32,
	hwnd windows.HWND, idObject int32, idChild int32,
	idEventThread uint32, dwmsEventTime uint32) uintptr {

	if activeWindowWatcher == nil {
		fmt.Println("[CALLBACK DEBUG] !!! No activeWindowWatcher !!!")
		return 0
	}

	// Filtering (bring back from minimal script if needed)
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
	// ---- End Filtering ----

	fmt.Printf("[CALLBACK DEBUG] Filter PASSED! Event=0x%X, Hwnd=0x%X\n", event, hwnd)

	// Debounce and signal logic
	activeWindowWatcher.mutex.RLock() // Use RLock for reading lastEventTime
	lastEvent := activeWindowWatcher.lastEventTime
	activeWindowWatcher.mutex.RUnlock()

	now := time.Now()
	const callbackDebounceDuration = 1000 * time.Millisecond
	if now.Sub(lastEvent) > callbackDebounceDuration {
		activeWindowWatcher.mutex.Lock() // Lock for writing lastEventTime
		activeWindowWatcher.lastEventTime = now
		activeWindowWatcher.mutex.Unlock()

		select {
		case activeWindowWatcher.changeDetected <- struct{}{}:
			fmt.Printf("[CALLBACK DEBUG] --> Sent signal. Event=0x%X, Hwnd=0x%X\n", event, hwnd)
		default:
			fmt.Printf("[CALLBACK DEBUG] !! changeDetected channel BUSY. Event=0x%X, Hwnd=0x%X\n", event, hwnd)
		}
	} else {
		fmt.Printf("[CALLBACK DEBUG] Debounced! Event=0x%X, Hwnd=0x%X\n", event, hwnd)
	}

	return 0
})

// Start launches the dedicated hook-setting and message loop goroutine
func (w *WindowWatcher) Start() error {
	w.mutex.Lock() // Lock for checking/setting isRunning
	if w.isRunning {
		w.mutex.Unlock()
		fmt.Println("[WATCHER START] Already running.")
		return nil
	}

	// Assign the active watcher *before* launching the goroutine
	// Consider potential race conditions if Start can be called concurrently
	if activeWindowWatcher != nil {
		w.mutex.Unlock()
		return fmt.Errorf("another WindowWatcher is already active")
	}
	activeWindowWatcher = w

	w.isRunning = true
	// Reset stopChan if it was closed previously (allow restarting)
	// Or better: Ensure Start is only called once per watcher instance
	// For now, assume it's called once. If restart is needed, re-create the channel.
	// w.stopChan = make(chan struct{}) // Re-create if allowing restart

	w.mutex.Unlock() // Unlock before launching goroutine

	fmt.Println("[WATCHER START] Launching hook and message loop goroutine...")
	go w.hookAndMessageLoop() // Launch the combined function

	return nil
}

// hookAndMessageLoop sets the hook and runs the message loop on the same thread
func (w *WindowWatcher) hookAndMessageLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fmt.Println("[HOOK LOOP] Goroutine started, thread locked.")

	// --- Set the hook WITHIN this goroutine ---
	hookCallbackPtr := winEventProcCallback
	fmt.Printf("[HOOK LOOP] Attempting SetWinEventHook with callback: %v\n", hookCallbackPtr)

	hook, _, err := procSetWinEventHook.Call(
		uintptr(EVENT_OBJECT_SHOW),
		uintptr(EVENT_OBJECT_HIDE),
		0, hookCallbackPtr, 0, 0,
		uintptr(WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS),
	)

	hookErrString := ""
	if err != nil {
		hookErrString = err.Error()
	}

	if hook == 0 || (hookErrString != "" && !strings.Contains(hookErrString, "operation completed successfully")) {
		lastErr := syscall.GetLastError()
		fmt.Printf("[HOOK LOOP] !!!!! SetWinEventHook FAILED! Hook=0x%X, Error='%s', LastError=%d !!!!!\n", hook, hookErrString, lastErr)
		w.mutex.Lock()
		w.isRunning = false
		if activeWindowWatcher == w {
			activeWindowWatcher = nil
		}
		w.mutex.Unlock()
		return
	}

	w.mutex.Lock()
	w.eventHook = HWINEVENTHOOK(hook)
	w.mutex.Unlock()
	fmt.Printf("[HOOK LOOP] SetWinEventHook SUCCESSFUL. Hook Handle: %X\n", hook)
	// -----------------------------------------

	// --- Message Loop ---
	var loopThreadId uint32 = 0
	retTID, _, _ := procGetCurrentThreadId.Call()
	loopThreadId = uint32(retTID)
	fmt.Printf("[HOOK LOOP] Message loop starting on Thread ID: %d (0x%X)\n", loopThreadId, loopThreadId)

	go func(tid uint32) {
		<-w.stopChan
		fmt.Printf("[HOOK LOOP] Stop signal received for Thread ID: %d\n", tid)
		if tid != 0 {
			ret, _, postErr := procPostThreadMessageW.Call(uintptr(tid), uintptr(WM_QUIT), 0, 0)
			if ret == 0 {
				fmt.Printf("[HOOK LOOP] !!!!! PostThreadMessageW(WM_QUIT) FAILED. Error: %v !!!!!\n", postErr)
			} else {
				fmt.Println("[HOOK LOOP] WM_QUIT posted via PostThreadMessageW.")
			}
		} else {
			fmt.Println("[HOOK LOOP] !!! Cannot post WM_QUIT, Thread ID is 0 !!!")
		}
	}(loopThreadId)

	// --- Use locally defined MSG struct ---
	var msg MSG
	// ------------------------------------
	for {
		ret, _, getMsgErr := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0)

		getMsgRet := int32(ret)
		if getMsgRet == -1 {
			fmt.Fprintf(os.Stderr, "[HOOK LOOP] !!!!! GetMessageW ERROR: Ret=-1, Error=%v !!!!!\n", getMsgErr)
			break
		}
		if getMsgRet == 0 {
			fmt.Println("[HOOK LOOP] GetMessageW received WM_QUIT (Ret=0). Exiting loop.")
			break
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	// --------------------

	// --- Unhook logic remains the same ---
	w.mutex.Lock()
	currentHook := w.eventHook
	w.eventHook = 0
	w.isRunning = false
	if activeWindowWatcher == w {
		activeWindowWatcher = nil
	}
	w.mutex.Unlock()

	if currentHook != 0 {
		fmt.Printf("[HOOK LOOP] Unhooking event hook: %X\n", currentHook)
		ret, _, unhookErr := procUnhookWinEvent.Call(uintptr(currentHook))
		if ret == 0 {
			fmt.Printf("[HOOK LOOP] !!!!! UnhookWinEvent FAILED: Error=%v !!!!!\n", unhookErr)
		} else {
			fmt.Println("[HOOK LOOP] UnhookWinEvent successful.")
		}
	}
	// ----------------------------------

	fmt.Println("[HOOK LOOP] Goroutine finished.")
}

// Stop signals the hookAndMessageLoop goroutine to exit
func (w *WindowWatcher) Stop() {
	w.mutex.Lock() // Lock for reading/closing stopChan safely
	fmt.Println("[WATCHER STOP] Stop called.")
	if !w.isRunning {
		fmt.Println("[WATCHER STOP] Hook loop not running.")
		w.mutex.Unlock()
		return
	}

	select {
	case <-w.stopChan:
		fmt.Println("[WATCHER STOP] Stop channel already closed.")
	default:
		close(w.stopChan)
		fmt.Println("[WATCHER STOP] Closed stop channel.")
	}
	w.mutex.Unlock() // Unlock after closing channel

	// Note: Unhooking is now handled *inside* the hookAndMessageLoop goroutine
	// We just signal it to stop here.
	// Maybe add a short wait or a WaitGroup if synchronous stop is needed.
	fmt.Println("[WATCHER STOP] Stop signal sent.")
}

// GetFilteredListOfWindows returns a list of filtered windows
func (a *WindowManagementAdapter) GetFilteredListOfWindows(thisWindow win.HWND) WindowMapping {
	tempWindowMapping := make(WindowMapping) // Target map

	enumFunc := windows.NewCallback(func(hwnd windows.HWND, lparam uintptr) uintptr {

		// --- Get window properties ---
		tempIsVisible := IsWindowVisible(hwnd)
		if !tempIsVisible {
			return 1
		} // Early exit if not visible

		tempTitle := GetWindowText(hwnd)
		tempClassName := GetClassName(hwnd)

		var tempIsCloaked int32
		ret, _, _ := procDwmGetWindowAttribute.Call(
			uintptr(hwnd), uintptr(DWMWA_CLOAKED), uintptr(unsafe.Pointer(&tempIsCloaked)), uintptr(unsafe.Sizeof(tempIsCloaked)),
		)
		// cloakCheckResult := "OK" // No longer needed for debug print
		if ret != 0 {
			tempIsCloaked = 0
		} // Default to not cloaked on failure

		// Call shouldIncludeWindow
		if shouldIncludeWindow(hwnd, tempTitle, tempClassName, int(tempIsCloaked), thisWindow) {
			infoMap, appName := getWindowInfo(hwnd)
			cleanWindowTitles(tempWindowMapping, infoMap, appName)
		}
		return 1 // TRUE
	})

	procEnumWindows.Call(enumFunc, 0)
	runtime.KeepAlive(enumFunc)

	tempWindowMapping = assignInstanceNumbers(tempWindowMapping, a.winManager.GetOpenWindowsInfo())
	return tempWindowMapping
}

// Wrapper for IsWindowVisible using windows package
func IsWindowVisible(hwnd windows.HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

// UpdateOpenWindowsInfo updates the manager's window information
func (wm *WindowManager) UpdateOpenWindowsInfo(mapping WindowMapping) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	wm.openWindowsInfo = mapping
}

// GetOpenWindowsInfo returns a copy of the open windows info
func (wm *WindowManager) GetOpenWindowsInfo() WindowMapping {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	// Create a copy of the map to avoid concurrent modification issues
	result := make(WindowMapping)
	for hwnd, info := range wm.openWindowsInfo {
		result[hwnd] = info
	}
	return result
}

// AddHwndToExclude adds a window handle to the exclusion list
func AddHwndToExclude(hwnd win.HWND) {
	hwndToExclude = append(hwndToExclude, hwnd)
}

// Update shouldIncludeWindow signature to use windows.HWND
func shouldIncludeWindow(hwnd windows.HWND, windowTitle, className string, isCloaked int, thisWindow win.HWND) bool {
	// Cast thisWindow (lxn) to windows.HWND for comparison if needed,
	// OR change thisWindow parameter type too if feasible.
	// For now, let's cast hwnd (windows) to lxn for comparison with excluded list/thisWindow
	lxnHwnd := win.HWND(hwnd)

	// Check specific HWND exclusion list (using lxn/win HWND)
	for _, excluded := range hwndToExclude {
		if lxnHwnd == excluded {
			return false
		}
	}

	// Check main properties
	return isCloaked == 0 &&
		strings.TrimSpace(windowTitle) != "" &&
		!excludedClassNames[className] &&
		lxnHwnd != thisWindow // Compare lxn HWNDs
}

// cleanWindowTitles updates window titles in the mapping
func cleanWindowTitles(mapping WindowMapping, entry WindowMapping, appName string) {
	for hwnd, info := range entry {
		cleanTitle := info.Title

		if info.ExeName == "explorer.exe" && strings.Contains(info.Title, " - File Explorer") {
			cleanTitle = strings.Replace(info.Title, " - File Explorer", "", -1)
		} else if strings.Contains(info.Title, " - "+appName) {
			cleanTitle = strings.Replace(info.Title, " - "+appName, "", -1)
		}

		mapping[hwnd] = WindowInfo{
			Title:    cleanTitle,
			ExeName:  info.ExeName,
			Instance: 0,
		}
	}
}

// assignInstanceNumbers assigns instance numbers to windows with the same title and exe
func assignInstanceNumbers(tempMapping WindowMapping, existingMapping WindowMapping) WindowMapping {
	resultMapping := make(WindowMapping)

	// Track used instance numbers for each title/exe pair
	titleExeMapping := make(map[string]map[int]bool)

	// First register all instances from existing mapping
	for _, info := range existingMapping {
		key := fmt.Sprintf("%s|%s", info.Title, info.ExeName)
		if _, exists := titleExeMapping[key]; !exists {
			titleExeMapping[key] = make(map[int]bool)
		}
		titleExeMapping[key][info.Instance] = true
	}

	// Process each window
	for hwnd, info := range tempMapping {
		// If window exists in existing mapping, update title and exe but keep instance number
		if existingInfo, exists := existingMapping[hwnd]; exists {
			newInfo := tempMapping[hwnd]
			if newInfo.Title != existingInfo.Title {
				newInfo.Instance = 0
			} else {
				newInfo.Instance = existingInfo.Instance
				resultMapping[hwnd] = newInfo
				continue
			}
		}

		key := fmt.Sprintf("%s|%s", info.Title, info.ExeName)
		if _, exists := titleExeMapping[key]; !exists {
			titleExeMapping[key] = make(map[int]bool)
		}

		// Find next available instance number
		newInstance := 0
		for titleExeMapping[key][newInstance] {
			newInstance++
		}

		// Add new instance to tracking
		titleExeMapping[key][newInstance] = true
		info.Instance = newInstance
		resultMapping[hwnd] = info
	}

	return resultMapping
}

// getWindowInfo gets information about a window
func getWindowInfo(hwnd windows.HWND) (WindowMapping, string) {
	result := make(WindowMapping) // Still map[win.HWND]WindowInfo
	windowTitle := GetWindowText(hwnd)
	lxnHwnd := win.HWND(hwnd) // Cast ONCE for map key

	if hwnd != 0 {
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		if pid != 0 {
			exePath, err := getProcessExePath(pid)
			if err != nil {
				result[lxnHwnd] = WindowInfo{Title: windowTitle, ExeName: "Unknown App", Instance: 0}
				return result, "Unknown App"
			}
			if fileExists(exePath) {
				exeName := strings.ToLower(filepath.Base(exePath))
				appName := getFriendlyAppName(exePath, exeName)
				result[lxnHwnd] = WindowInfo{Title: windowTitle, ExeName: exeName, Instance: 0}
				return result, appName
			}
		}
		result[lxnHwnd] = WindowInfo{Title: windowTitle, ExeName: "Unknown App", Instance: 0}
	}
	return result, "Unknown App"
}

// GetWindowText gets the title of a window
func GetWindowText(hwnd windows.HWND) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if textLen == 0 {
		return ""
	}
	buf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(textLen+1))
	// Add basic error check for UTF16ToString if needed, though usually okay if length > 0
	return windows.UTF16ToString(buf)
}

// GetClassName gets the class name of a window
func GetClassName(hwnd windows.HWND) string {
	var buf [256]uint16 // Use a fixed-size buffer, class names are usually not excessively long
	// Use procGetClassNameW defined earlier
	len, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if len == 0 {
		return "" // Return empty string if call failed or class name is empty
	}
	// Note: The returned 'len' is the number of characters copied, *not* including null terminator
	return windows.UTF16ToString(buf[:len])
}

func getProcessExePath(pid uint32) (string, error) {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("failed to open process: %w", err)
	}
	defer syscall.CloseHandle(handle)

	var buf [syscall.MAX_PATH]uint16
	size := uint32(syscall.MAX_PATH)

	queryFullProcessImageName := syscall.NewLazyDLL("kernel32.dll").NewProc("QueryFullProcessImageNameW")
	ret, _, err := queryFullProcessImageName.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to query process image name: %w", err)
	}

	return syscall.UTF16ToString(buf[:size]), nil
}

// getFriendlyAppName gets a friendly name from file version info
func getFriendlyAppName(exePath, exeName string) string {
	// This is a simplified version - Windows version info retrieval is complex in Go
	// A full implementation would use Windows APIs to get FileDescription

	// For brevity, just return the capitalized exe name without extension
	baseName := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	return strings.Title(baseName)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
