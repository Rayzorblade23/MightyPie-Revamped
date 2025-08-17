package pieButtonExecutionAdapter

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// --- Windows API constants and helpers ---
var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procMouseEvent = user32.NewProc("mouse_event")

	showWindow               = user32.NewProc("ShowWindow")
	getWindowRect            = user32.NewProc("GetWindowRect")
	setForegroundWindow      = user32.NewProc("SetForegroundWindow")
	enumWindows              = user32.NewProc("EnumWindows")
	getWindowTextW           = user32.NewProc("GetWindowTextW")
	getForegroundWindow      = user32.NewProc("GetForegroundWindow")
	getWindowPlacement       = user32.NewProc("GetWindowPlacement")
	procGetClassNameW        = user32.NewProc("GetClassNameW")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	getCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
	attachThreadInput        = user32.NewProc("AttachThreadInput")
	bringWindowToTop         = user32.NewProc("BringWindowToTop")
	setWindowPos             = user32.NewProc("SetWindowPos")
	isIconic                 = user32.NewProc("IsIconic")
	getCursorPos             = user32.NewProc("GetCursorPos")
	monitorFromPoint         = user32.NewProc("MonitorFromPoint")
	getMonitorInfo           = user32.NewProc("GetMonitorInfoW")
	isZoomed                 = user32.NewProc("IsZoomed")
)

const (
	SWP_NOACTIVATE = 0x0010
	SW_MAXIMIZE    = 3
	SW_MINIMIZE    = 6
	SW_RESTORE     = 9

	HWND_TOPMOST   = ^uintptr(0)     // 0xFFFFFFFF
	HWND_NOTOPMOST = ^uintptr(0) - 1 // 0xFFFFFFFE
	SWP_NOMOVE     = 0x0002
	SWP_NOSIZE     = 0x0001
	SWP_SHOWWINDOW = 0x0040
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

// enumContext holds per-call state for EnumWindows in GetWindowAtPoint.
type enumContext struct {
    x, y           int
    managedWindows map[int]core.WindowInfo
    hwnd           WindowHandle
    found          bool
}

// Single package-level callback to avoid repeated allocations.
var enumWindowsProc = syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
    ctx := (*enumContext)(unsafe.Pointer(lparam))

    handle := int(hwnd)

    winInfo, exists := ctx.managedWindows[handle]
    if !exists || winInfo.ExeName == "mightypie-revamped.exe" {
        return 1
    }

    var rect RECT
    _, _, _ = getWindowRect.Call(
        uintptr(hwnd),
        uintptr(unsafe.Pointer(&rect)),
    )

    if int32(ctx.x) >= rect.Left && int32(ctx.x) <= rect.Right &&
        int32(ctx.y) >= rect.Top && int32(ctx.y) <= rect.Bottom {
        ctx.hwnd = WindowHandle(hwnd)
        ctx.found = true
        return 0 // stop enumeration
    }
    return 1 // continue
})

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
	// First set foreground
	_, _, err := setForegroundWindow.Call(uintptr(hwnd))
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("failed to set foreground window: %v", err)
	}
	
	// Then maximize
	_, _, err = showWindow.Call(uintptr(hwnd), uintptr(SW_MAXIMIZE))
	
	// Windows API often returns non-nil error even on success
	if err != nil {
		if err.Error() == "The operation completed successfully." {
			log.Info("Successfully maximized window HWND %X", hwnd)
			return nil
		}
		
		// Ignore quota errors as they often occur even when the window is successfully maximized
		if strings.Contains(err.Error(), "Not enough quota") {
			// Check if the window is actually maximized despite the error
			isMax, _ := hwnd.IsMaximized()
			if isMax { // Window is maximized
				log.Info("Successfully maximized window HWND %X (despite quota error)", hwnd)
				return nil
			}
		}
		
		return fmt.Errorf("failed to maximize window: %v", err)
	}
	
	log.Info("Successfully maximized window HWND %X", hwnd)
	return nil
}

func (hwnd WindowHandle) Minimize() error {
	_, _, err := showWindow.Call(uintptr(hwnd), uintptr(SW_MINIMIZE))
	
	// Windows API often returns non-nil error even on success
	// "The operation completed successfully" is a known case
	// "Not enough quota" can also happen when the operation actually succeeds
	if err != nil {
		if err.Error() == "The operation completed successfully." {
			log.Info("Successfully minimized window HWND %X", hwnd)
			return nil
		}
		
		// Ignore quota errors as they often occur even when the window is successfully minimized
		if strings.Contains(err.Error(), "Not enough quota") {
			// Check if the window is actually minimized despite the error
			ret, _, _ := isIconic.Call(uintptr(hwnd))
			if ret != 0 { // Window is minimized
				log.Info("Successfully minimized window HWND %X (despite quota error)", hwnd)
				return nil
			}
		}
		
		return fmt.Errorf("failed to minimize window: %v", err)
	}
	
	log.Info("Successfully minimized window HWND %X", hwnd)
	return nil
}

func (hwnd WindowHandle) Restore() error {
	_, _, err := showWindow.Call(uintptr(hwnd), uintptr(SW_RESTORE))
	
	// Windows API often returns non-nil error even on success
	if err != nil {
		if err.Error() == "The operation completed successfully." {
			log.Info("Successfully restored window HWND %X", hwnd)
			return nil
		}
		
		// Ignore quota errors as they often occur even when the window is successfully restored
		if strings.Contains(err.Error(), "Not enough quota") {
			// Check if the window is actually restored despite the error
			ret, _, _ := isIconic.Call(uintptr(hwnd))
			isMax, _ := hwnd.IsMaximized()
			if ret == 0 && !isMax { // Window is neither minimized nor maximized, so it's restored
				log.Info("Successfully restored window HWND %X (despite quota error)", hwnd)
				return nil
			}
		}
		
		return fmt.Errorf("failed to restore window: %v", err)
	}
	
	log.Info("Successfully restored window HWND %X", hwnd)
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

    ctx := enumContext{x: x, y: y, managedWindows: managedWindows}
    enumWindows.Call(enumWindowsProc, uintptr(unsafe.Pointer(&ctx)))
    runtime.KeepAlive(&ctx)

    if !ctx.found {
        return 0, fmt.Errorf("no managed window found at coordinates")
    }

    return ctx.hwnd, nil
}

// setForegroundOrMinimize brings the window to the foreground or minimizes it if it's already in the foreground.
// Now a method so it can access installedAppsInfo and windowsList for special handling.
func (a *PieButtonExecutionAdapter) setForegroundOrMinimize(hwnd uintptr) error {
	// --- Special handling for Task Manager ---
	a.mu.RLock()
	winInfo, ok := a.windowsList[int(hwnd)]
	appInfo, appOk := a.installedAppsInfo["Task Manager"]
	a.mu.RUnlock()
	if ok && strings.EqualFold(winInfo.ExeName, "taskmgr.exe") && appOk {
		// Always launch Task Manager, as foregrounding is unreliable and only one instance runs
		if err := LaunchApp("Task Manager", appInfo); err != nil {
			log.Error("Failed to launch Task Manager: %v", err)
			return err
		}
		return nil
	}

	foreground, _, callErr := getForegroundWindow.Call()
	if callErr != nil && callErr != syscall.Errno(0) {
		log.Error("getForegroundWindow failed: %v", callErr)
		return fmt.Errorf("getForegroundWindow failed: %v", callErr)
	}

	if hwnd == foreground {
		// Check if the window is minimized (iconic)
		ret, _, _ := isIconic.Call(hwnd)
		if ret != 0 {
			showWindow.Call(hwnd, SW_RESTORE)
			// Continue to rest of logic to bring to foreground
		} else {
			// Minimize if already foreground and not minimized
			if err := WindowHandle(hwnd).Minimize(); err != nil {
				return err
			}
			return nil
		}
	}

	// --- Input join (AttachThreadInput) method ---
	var tmp uint32
	fgThread, _, _ := getWindowThreadProcessId.Call(foreground, uintptr(unsafe.Pointer(&tmp)))
	thisThread, _, _ := getCurrentThreadId.Call()

	// Only attach thread input if threads differ
	attached := false
	if thisThread != fgThread {
		attachThreadInput.Call(thisThread, fgThread, 1)
		attached = true
	}

	// Check if window is maximized
	wasMaximized := false
	var placement windowPlacement
	placement.Length = uint32(unsafe.Sizeof(placement))
	ret, _, _ := getWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if ret != 0 && placement.ShowCmd == SW_MAXIMIZE {
		wasMaximized = true
	}

	// Restore or maximize as needed
	if wasMaximized {
		showWindow.Call(hwnd, SW_MAXIMIZE)
	} else {
		showWindow.Call(hwnd, SW_RESTORE)
	}

	// Bring window to top
	bringWindowToTop.Call(hwnd)

	// Set window topmost, then notopmost
	_, _, _ = setWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0, uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW))
	_, _, _ = setWindowPos.Call(hwnd, HWND_NOTOPMOST, 0, 0, 0, 0, uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW))

	// Now try to bring to foreground
	ret, _, _ = setForegroundWindow.Call(hwnd)
	if ret == 0 {
		log.Error("setForegroundWindow failed after restore. This is common when another window is already in the foreground.")
		if attached {
			attachThreadInput.Call(thisThread, fgThread, 0)
		}
		return fmt.Errorf("setForegroundWindow failed after restore")
	}

	if attached {
		attachThreadInput.Call(thisThread, fgThread, 0)
	}

	return nil
}

func logWindowContext(index int, text string, hwnd uintptr) {
	title := GetWindowTitle(hwnd)
	class := GetWindowClassName(hwnd)
	log.Info("show_any_window: Button %d (%s), HWND %X (%d), Title: '%s', Class: '%s'",
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

// CenterWindowUnderCursor centers the given window under the monitor where the cursor is,
// and resizes it to half the monitor's width and height.
func CenterWindowOnMonitor(hwnd uintptr) error {

	// Check if maximized
	zoomed, _, _ := isZoomed.Call(hwnd)
	if zoomed != 0 {
		ret, _, err := showWindow.Call(hwnd, SW_RESTORE)
		if ret == 0 {
			return fmt.Errorf("ShowWindow(SW_RESTORE) failed: %v", err)
		}
	}

	type POINT struct {
		X, Y int32
	}
	type MONITORINFO struct {
		CbSize    uint32
		RcMonitor RECT
		RcWork    RECT
		DwFlags   uint32
	}

	// Get cursor position
	var pt POINT
	r1, _, err := getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r1 == 0 {
		return fmt.Errorf("GetCursorPos failed: %v", err)
	}

	// Get monitor under cursor
	const MONITOR_DEFAULTTONEAREST = 2
	hMonitor, _, _ := monitorFromPoint.Call(uintptr(*(*int64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	if hMonitor == 0 {
		return fmt.Errorf("MonitorFromPoint failed")
	}

	// Get monitor info
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	r2, _, err := getMonitorInfo.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if r2 == 0 {
		return fmt.Errorf("GetMonitorInfo failed: %v", err)
	}

	monWidth := mi.RcMonitor.Right - mi.RcMonitor.Left
	monHeight := mi.RcMonitor.Bottom - mi.RcMonitor.Top
	winWidth := monWidth / 2
	winHeight := monHeight / 2
	winLeft := mi.RcMonitor.Left + (monWidth-winWidth)/2
	winTop := mi.RcMonitor.Top + (monHeight-winHeight)/2

	// Set window position and size
	const SWP_NOZORDER = 0x0004
	const SWP_NOACTIVATE = 0x0010
	r3, _, err := setWindowPos.Call(
		hwnd,
		0,
		uintptr(winLeft),
		uintptr(winTop),
		uintptr(winWidth),
		uintptr(winHeight),
		SWP_NOZORDER|SWP_NOACTIVATE,
	)
	if r3 == 0 {
		log.Error("[CenterWindowOnMonitor] SetWindowPos failed: %v", err)
		return fmt.Errorf("SetWindowPos failed: %v", err)
	}

	return nil
}
