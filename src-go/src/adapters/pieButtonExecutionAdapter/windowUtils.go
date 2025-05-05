package pieButtonExecutionAdapter

import (
    "fmt"
    "syscall"
    "unsafe"
)

var (
    user32              = syscall.NewLazyDLL("user32.dll")
    showWindow          = user32.NewProc("ShowWindow")
    getClassName        = user32.NewProc("GetClassNameW")
    getWindowRect       = user32.NewProc("GetWindowRect")
    setForegroundWindow = user32.NewProc("SetForegroundWindow")
    enumWindows         = user32.NewProc("EnumWindows")
)

type RECT struct {
    Left, Top, Right, Bottom int32
}

const (
    SW_MAXIMIZE = 3
    SW_MINIMIZE = 6
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