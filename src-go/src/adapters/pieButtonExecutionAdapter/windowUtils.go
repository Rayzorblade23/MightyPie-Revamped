package pieButtonExecutionAdapter

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	showWindow          = user32.NewProc("ShowWindow")
	getClassName        = user32.NewProc("GetClassNameW")
	getWindowRect       = user32.NewProc("GetWindowRect")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	enumWindows         = user32.NewProc("EnumWindows")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	switchToThisWindow  = user32.NewProc("SwitchToThisWindow")
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

const (
	SWP_NOACTIVATE = 0x0010
)

const (
	SW_MAXIMIZE = 3
	SW_MINIMIZE = 6
	SW_RESTORE  = 9 // Activates and displays the window. If minimized/maximized, restores to original size/pos.
)

type WindowHandle uintptr

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

func (hwnd WindowHandle) GetClassName() string {
	buf := make([]uint16, 256)
	_, _, _ = getClassName.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

// SetForegroundOrMinimize brings the window to the foreground or minimizes it if it's already in the foreground.
func setForegroundOrMinimize(hwnd uintptr) error {
	foreground, _, callErr := getForegroundWindow.Call()
	if callErr != nil && callErr != syscall.Errno(0) {
		return fmt.Errorf("getForegroundWindow failed: %v", callErr)
	}

	if hwnd == foreground {
		log.Print("forceForeground: window already in foreground, minimizing instead")
		return WindowHandle(hwnd).Minimize()
	}

	ret, _, callErr := switchToThisWindow.Call(hwnd, 1)
	if ret == 0 {
		return fmt.Errorf("switchToThisWindow failed: %v", callErr)
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
	_, _, _ = getClassName.Call( // Using the getClassName global var from your first code snippet
		hwnd,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
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
