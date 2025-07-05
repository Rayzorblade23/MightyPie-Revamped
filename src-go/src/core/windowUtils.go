package core

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// CenterWindowUnderCursor centers the given window under the monitor where the cursor is,
// and resizes it to half the monitor's width and height.
func CenterWindowUnderCursor(hwnd win.HWND) error {
	// Define necessary Win32 structs
	type POINT struct {
		X, Y int32
	}
	type RECT struct {
		Left, Top, Right, Bottom int32
	}
	type MONITORINFO struct {
		CbSize    uint32
		RcMonitor RECT
		RcWork    RECT
		DwFlags   uint32
	}

	// Get cursor position
	var pt POINT
	user32 := syscall.NewLazyDLL("user32.dll")
	getCursorPos := user32.NewProc("GetCursorPos")
	monitorFromPoint := user32.NewProc("MonitorFromPoint")
	getMonitorInfo := user32.NewProc("GetMonitorInfoW")
	setWindowPos := user32.NewProc("SetWindowPos")

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
		uintptr(hwnd),
		0,
		uintptr(winLeft),
		uintptr(winTop),
		uintptr(winWidth),
		uintptr(winHeight),
		SWP_NOZORDER|SWP_NOACTIVATE,
	)
	if r3 == 0 {
		return fmt.Errorf("SetWindowPos failed: %v", err)
	}

	return nil
}
