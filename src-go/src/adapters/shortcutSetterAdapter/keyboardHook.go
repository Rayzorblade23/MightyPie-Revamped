package shortcutSetterAdapter

import (
    "fmt"
    "sync"
    "syscall"
    "unsafe"

    "github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

type setterKeyboardHook struct {
    callback func(shortcut []int)
    pressed  map[int]bool
    stopChan chan struct{}
    stopped  bool
    mu       sync.Mutex
    hook     syscall.Handle
}

func newSetterKeyboardHook(callback func(shortcut []int)) *setterKeyboardHook {
    return &setterKeyboardHook{
        callback: callback,
        pressed:  make(map[int]bool),
        stopChan: make(chan struct{}),
        stopped:  false,
    }
}

func (kh *setterKeyboardHook) Run() error {
    hookProc := syscall.NewCallback(kh.hookProc)
    ret, _, err := core.SetWindowsHookEx.Call(
        uintptr(core.WH_KEYBOARD_LL),
        hookProc,
        0,
        0,
    )
    kh.hook = syscall.Handle(ret)
    if kh.hook == 0 {
        return fmt.Errorf("failed to set keyboard hook: %v", err)
    }

	var msg core.MSG

    for {
        select {
        case <-kh.stopChan:
            return nil
        default:
            core.GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        }
    }
}

func (kh *setterKeyboardHook) Stop() {
    kh.mu.Lock()
    defer kh.mu.Unlock()
    if !kh.stopped {
        close(kh.stopChan)
        kh.stopped = true
        if kh.hook != 0 {
            core.UnhookWindowsHookEx.Call(uintptr(kh.hook))
            kh.hook = 0
        }
        core.PostQuitMessage.Call(0)
    }
}

func (kh *setterKeyboardHook) hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
    if nCode == 0 {
        kbd := (*core.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
        vk := int(kbd.VKCode)

        switch wParam {
        case core.WM_KEYDOWN, core.WM_SYSKEYDOWN:
            if core.IsModifier(vk) {
                break
            }
            if kh.pressed[vk] {
                break
            }
            kh.pressed[vk] = true

            modifiers := getCurrentlyPressedModifiers()
            shortcut := append(modifiers, vk)
            kh.callback(shortcut)
        case core.WM_KEYUP, core.WM_SYSKEYUP:
            delete(kh.pressed, vk)
        }
    }
    ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
    return ret
}

// Helper to check which modifiers are currently pressed
func getCurrentlyPressedModifiers() []int {
    getKeyState := core.User32.NewProc("GetKeyState")
    modifiers := []int{}
    for _, mod := range []int{core.VK_SHIFT, core.VK_CONTROL, core.VK_ALT, 0x5B} {
        state, _, _ := getKeyState.Call(uintptr(mod))
        if (state & 0x8000) != 0 {
            modifiers = append(modifiers, mod)
        }
    }
    return modifiers
}