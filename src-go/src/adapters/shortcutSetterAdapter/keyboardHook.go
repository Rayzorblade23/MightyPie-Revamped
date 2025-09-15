package shortcutSetterAdapter

import (
    "fmt" // Keep for fmt.Errorf
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
        log.Error("Failed to set keyboard hook: %v", err)
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
        // Reset internal state to avoid stuck modifiers between sessions
        kh.pressed = make(map[int]bool)
    }
}

func (kh *setterKeyboardHook) hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
    if nCode == 0 {
        kbd := (*core.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
        vk := int(kbd.VKCode)

        switch wParam {
        case core.WM_KEYDOWN, core.WM_SYSKEYDOWN:
            // Let ESC pass through so the UI can close the dialog
            if vk == 0x1B { // VK_ESCAPE
                ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
                return ret
            }
            if core.IsModifier(vk) {
                // Let modifier keys pass through; we will read their state via GetKeyState
                ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
                return ret
            }
            if kh.pressed[vk] {
                // Already considered this non-modifier as down; swallow repeats
                return 1
            }
            kh.pressed[vk] = true

            // Build modifiers from OS state (since modifiers are allowed to pass through)
            mods := currentModifiersFromOS()
            shortcut := append(mods, vk)
            kh.callback(shortcut)
            // Swallow non-modifier keydown so OS doesn't act
            return 1
        case core.WM_KEYUP, core.WM_SYSKEYUP:
            // Let ESC pass through so the UI can close the dialog
            if vk == 0x1B { // VK_ESCAPE
                ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
                return ret
            }
            if core.IsModifier(vk) {
                // Allow modifier keyup to pass through
                ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
                return ret
            }
            // Non-modifier keyup: clear and swallow (prevents OS action tied to keyup)
            delete(kh.pressed, vk)
            return 1
        }
    }
    // Default: pass through
    ret, _, _ := core.CallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
    return ret
}

// Helper to check which modifiers are currently pressed via OS state
func currentModifiersFromOS() []int {
    // Query OS state for all variants, then normalize to generic codes present in KeyMap
    getKeyState := core.User32.NewProc("GetKeyState")

    // helper to check pressed state
    pressed := func(vk int) bool {
        state, _, _ := getKeyState.Call(uintptr(vk))
        return (state & 0x8000) != 0
    }

    normalized := []int{}
    // Ctrl: VK_CONTROL(0x11) or LCTRL(0xA2) / RCTRL(0xA3)
    if pressed(core.VK_CONTROL) || pressed(0xA2) || pressed(0xA3) {
        normalized = append(normalized, core.VK_CONTROL)
    }
    // Alt: VK_MENU(0x12) or LALT(0xA4) / RALT(0xA5)
    if pressed(core.VK_MENU) || pressed(0xA4) || pressed(0xA5) {
        normalized = append(normalized, core.VK_MENU)
    }
    // Shift: VK_SHIFT(0x10) or LSHIFT(0xA0) / RSHIFT(0xA1)
    if pressed(core.VK_SHIFT) || pressed(0xA0) || pressed(0xA1) {
        normalized = append(normalized, core.VK_SHIFT)
    }
    // Win: Prefer left win VK_LWIN(0x5B) if either LWIN or RWIN is down
    if pressed(core.VK_LWIN) || pressed(core.VK_RWIN) {
        normalized = append(normalized, core.VK_LWIN)
    }

    return normalized
}