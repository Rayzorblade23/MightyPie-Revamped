package pieButtonExecutionAdapter

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
	"strings"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)


// --- Windows API constants and helpers ---
var (
	user32 = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	procMouseEvent = user32.NewProc("mouse_event")

	showWindow          = user32.NewProc("ShowWindow")
	getWindowRect       = user32.NewProc("GetWindowRect")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	enumWindows         = user32.NewProc("EnumWindows")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	switchToThisWindow  = user32.NewProc("SwitchToThisWindow")
	getWindowPlacement  = user32.NewProc("GetWindowPlacement")
	procGetClassNameW   = user32.NewProc("GetClassNameW")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	getCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
	attachThreadInput        = user32.NewProc("AttachThreadInput")
)

const (
	SWP_NOACTIVATE = 0x0010
	SW_MAXIMIZE    = 3
	SW_MINIMIZE    = 6
	SW_RESTORE     = 9
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

type windowPlacement struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    [2]int32
	PtMaxPosition    [2]int32
	RcNormalPosition RECT
}

type WindowHandle uintptr

// --- Window manipulation methods ---



// isExplorerWindow returns true if the given HWND belongs to explorer.exe using the cached windowsList
func isExplorerWindow(hwnd uintptr, windowsList map[int]core.WindowInfo) bool {
	if hwnd == 0 {
		return false
	}
	winInfo, ok := windowsList[int(hwnd)]
	if !ok {
		return false
	}
	return strings.ToLower(winInfo.ExeName) == "explorer.exe"
}

const WM_CLOSE = 0x0010

// Close sends a WM_CLOSE message to the window to request it to close.
func (hwnd WindowHandle) Close() error {
	_, _, err := setForegroundWindow.Call(uintptr(hwnd))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to set foreground window: %v", err)
	}
	ret, _, err := user32.NewProc("PostMessageW").Call(uintptr(hwnd), uintptr(WM_CLOSE), 0, 0)
	if ret == 0 {
		return fmt.Errorf("failed to send WM_CLOSE to window: %v", err)
	}
	return nil
}

func (hwnd WindowHandle) Maximize() error {
	_, _, err := setForegroundWindow.Call(uintptr(hwnd))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to set foreground window: %v", err)
	}
	_, _, err = showWindow.Call(uintptr(hwnd), uintptr(SW_MAXIMIZE))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to maximize window: %v", err)
	}
	return nil
}

func (hwnd WindowHandle) Minimize() error {
	_, _, err := showWindow.Call(uintptr(hwnd), uintptr(SW_MINIMIZE))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to minimize window: %v", err)
	}
	return nil
}

func (hwnd WindowHandle) Restore() error {
	_, _, err := showWindow.Call(uintptr(hwnd), uintptr(SW_RESTORE))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to restore window: %v", err)
	}
	return nil
}

// Returns true if the window is maximized, false otherwise
func (hwnd WindowHandle) IsMaximized() (bool, error) {
	var placement windowPlacement
	placement.Length = uint32(unsafe.Sizeof(placement))
	ret, _, err := getWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		return false, fmt.Errorf("GetWindowPlacement failed: %w", err)
	}
	return placement.ShowCmd == SW_MAXIMIZE, nil
}

func (a *PieButtonExecutionAdapter) GetWindowAtPoint(x, y int) (WindowHandle, error) {
	a.mu.RLock()
	managedWindows := a.windowsList
	a.mu.RUnlock()

	type windowInfo struct {
		hwnd  WindowHandle
		found bool
	}
	result := windowInfo{}

	cb := func(hwnd syscall.Handle, lparam uintptr) uintptr {
		handle := int(hwnd)

		winInfo, exists := managedWindows[handle]
		if !exists || winInfo.ExeName == "mightypie-revamped.exe" {
			return 1
		}

		var rect RECT
		_, _, _ = getWindowRect.Call(
			uintptr(hwnd),
			uintptr(unsafe.Pointer(&rect)),
		)

		if int32(x) >= rect.Left && int32(x) <= rect.Right &&
			int32(y) >= rect.Top && int32(y) <= rect.Bottom {
			result.hwnd = WindowHandle(hwnd)
			result.found = true
			return 0
		}
		return 1
	}

	syscallCallback := syscall.NewCallback(cb)
	enumWindows.Call(syscallCallback, 0)

	if !result.found {
		return 0, fmt.Errorf("no managed window found at coordinates")
	}

	return result.hwnd, nil
}


// SetForegroundOrMinimize brings the window to the foreground or minimizes it if it's already in the foreground.
func setForegroundOrMinimize(hwnd uintptr) error {
	log.Printf("[setForegroundOrMinimize] Called for HWND: 0x%X", hwnd)
	foreground, _, callErr := getForegroundWindow.Call()
	if callErr != nil && callErr != syscall.Errno(0) {
		log.Printf("[setForegroundOrMinimize] getForegroundWindow failed: %v", callErr)
		return fmt.Errorf("getForegroundWindow failed: %v", callErr)
	}

	if hwnd == foreground {
		log.Printf("[setForegroundOrMinimize] HWND is already foreground, minimizing instead.")
		return WindowHandle(hwnd).Minimize()
	}

	// --- Input join (AttachThreadInput) method ---
	var tmp uint32
	fgThread, _, _ := getWindowThreadProcessId.Call(foreground, uintptr(unsafe.Pointer(&tmp)))
	thisThread, _, _ := getCurrentThreadId.Call()
	log.Printf("[setForegroundOrMinimize] thisThread: %d, fgThread: %d", thisThread, fgThread)

	// Attach input of our thread and foreground thread
	res, _, err := attachThreadInput.Call(thisThread, fgThread, 1)
	log.Printf("[setForegroundOrMinimize] AttachThreadInput result: %d, err: %v", res, err)
	
	// Always restore the window first
	showRet, _, showErr := showWindow.Call(hwnd, SW_RESTORE)
	if showRet == 0 {
		log.Printf("[setForegroundOrMinimize] showWindow(SW_RESTORE) failed: %v", showErr)
		attachThreadInput.Call(thisThread, fgThread, 0)
		return fmt.Errorf("showWindow(SW_RESTORE) failed: %v", showErr)
	}

	// Now try to bring to foreground
	ret, _, callErr := setForegroundWindow.Call(hwnd)
	log.Printf("[setForegroundOrMinimize] setForegroundWindow result: %d, err: %v", ret, callErr)

	// Detach input
	detachRes, _, detachErr := attachThreadInput.Call(thisThread, fgThread, 0)
	log.Printf("[setForegroundOrMinimize] DetachThreadInput result: %d, err: %v", detachRes, detachErr)

	if ret == 0 {
		return fmt.Errorf("setForegroundWindow failed after restore: %v", callErr)
	}

	return nil
}


func logWindowContext(index int, text string, hwnd uintptr) {
	title := GetWindowTitle(hwnd)
	class := GetWindowClassName(hwnd)
	log.Printf("show_any_window: Button %d (%s), HWND %X (%d), Title: '%s', Class: '%s'",
		index, text, hwnd, hwnd, title, class)
}

// Helper to get window title
func GetWindowTitle(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	// Max length for window title
	// Adjust if necessary, but 256 characters should be sufficient for most titles
	const maxTitleLength = 256
	buf := make([]uint16, maxTitleLength)
	lenCopied, _, _ := getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(maxTitleLength))
	if lenCopied == 0 {
		return "" // No title or error
	}
	return syscall.UTF16ToString(buf[:lenCopied])
}

// Your existing GetClassName for WindowHandle can be adapted or a similar one for uintptr
func GetWindowClassName(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	_, _, _ = procGetClassNameW.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
