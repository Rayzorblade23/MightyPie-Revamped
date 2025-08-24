package pieButtonExecutionAdapter

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/go-ole/go-ole"
	"github.com/go-vgo/robotgo"
)

// Windows mouse event constants
const (
	MOUSEEVENTF_XDOWN = 0x0080
	MOUSEEVENTF_XUP   = 0x0100
	XBUTTON1          = 0x0001
	XBUTTON2          = 0x0002
)

// BringLastExplorerWindowToForeground brings the last used Explorer window to the foreground, if available.
func (a *PieButtonExecutionAdapter) BringLastExplorerWindowToForeground() error {
	a.mu.RLock()
	hwnd := a.lastExplorerWindowHWND
	a.mu.RUnlock()
	if hwnd == 0 {
		return fmt.Errorf("no last Explorer window recorded")
	}
	if err := a.setForegroundOrMinimize(uintptr(hwnd)); err != nil {
		return fmt.Errorf("failed to bring Explorer window to foreground: %w", err)
	}
	return nil
}

// BringAllExplorerWindowsToForeground brings all current Explorer windows to the foreground.
func (a *PieButtonExecutionAdapter) BringAllExplorerWindowsToForeground() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	count := 0
	for hwndInt, winInfo := range a.windowsList {
		if winInfo.ExeName != "" && strings.EqualFold(winInfo.ExeName, "explorer.exe") {
			hwnd := uintptr(hwndInt)
			if err := a.setForegroundOrMinimize(hwnd); err == nil {
				count++
			}
		}
	}
	if count == 0 {
		return fmt.Errorf("no Explorer windows found")
	}
	return nil
}

// ForwardsButtonClick simulates a press and release of the XBUTTON1 (Forwards) mouse button.
func (a *PieButtonExecutionAdapter) ForwardsButtonClick() error {
	// Give the UI a brief moment to defocus/hide before sending the XBUTTON click
	time.Sleep(150 * time.Millisecond)
	return sendXButtonClick(XBUTTON2)
}

// BackwardsButtonClick simulates a press and release of the XBUTTON2 (Backwards) mouse button.
func (a *PieButtonExecutionAdapter) BackwardsButtonClick() error {
	// Give the UI a brief moment to defocus/hide before sending the XBUTTON click
	time.Sleep(150 * time.Millisecond)
	return sendXButtonClick(XBUTTON1)
}

// RestartAndRestoreExplorerWindows restarts explorer.exe and restores all previously open Explorer windows to their original positions.
func (a *PieButtonExecutionAdapter) RestartAndRestoreExplorerWindows() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitialize(0); err != nil {
		return fmt.Errorf("CoInitialize failed: %w", err)
	}
	defer ole.CoUninitialize()

	// Show a warning dialog to the user
	ShowWarningMessageBox()

	// 1. Capture all open Explorer windows
	states, err := GetExplorerWindows()
	if err != nil {
		return fmt.Errorf("failed to get Explorer windows: %w", err)
	}
	log.Info("Captured %d Explorer windows before restart", len(states))

	// 2. Restart explorer.exe
	if err := RestartExplorer(); err != nil {
		return fmt.Errorf("failed to restart explorer.exe: %w", err)
	}
	log.Info("Explorer.exe restarted")
	// Give Explorer time to fully restart
	time.Sleep(2 * time.Second)

	// 3. Restore windows
	if err := RestoreExplorerWindows(states); err != nil {
		return fmt.Errorf("failed to restore Explorer windows: %w", err)
	}
	log.Info("Restored %d Explorer windows", len(states))
	// Give windows time to open and fully initialize
	delay := max(time.Duration(len(states))*900*time.Millisecond+2*time.Second, 3*time.Second)
	time.Sleep(delay)

	// 4. Move windows to original positions
	if err := SetExplorerWindowPositions(states); err != nil {
		return fmt.Errorf("failed to reposition Explorer windows: %w", err)
	}
	log.Info("Repositioned Explorer windows")

	// Close the warning dialog automatically
	CloseWarningMessageBox()
	return nil
}

// Copy simulates Ctrl+C to copy selected content to the clipboard.

func (a *PieButtonExecutionAdapter) Copy() error {
	time.Sleep(100 * time.Millisecond)
	robotgo.KeyDown("ctrl")
	robotgo.KeyDown("c")
	time.Sleep(50 * time.Millisecond)
	robotgo.KeyUp("c")
	robotgo.KeyUp("ctrl")
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Paste simulates Ctrl+V to paste clipboard content.
func (a *PieButtonExecutionAdapter) Paste() error {
	time.Sleep(100 * time.Millisecond)
	robotgo.KeyDown("ctrl")
	robotgo.KeyDown("v")
	time.Sleep(50 * time.Millisecond)
	robotgo.KeyUp("v")
	robotgo.KeyUp("ctrl")
	time.Sleep(100 * time.Millisecond)
	return nil
}

// OpenClipboard simulates Win+V to open the Windows clipboard history UI.
func (a *PieButtonExecutionAdapter) OpenClipboard() error {
	time.Sleep(100 * time.Millisecond)
	robotgo.KeyDown("cmd")
	robotgo.KeyDown("v")
	time.Sleep(50 * time.Millisecond)
	robotgo.KeyUp("v")
	robotgo.KeyUp("cmd")
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Fullscreen_F11 simulates pressing F11 to toggle fullscreen mode in most applications.
func (a *PieButtonExecutionAdapter) Fullscreen_F11() error {
	time.Sleep(100 * time.Millisecond)
	err := robotgo.KeyTap("f11")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap f11 failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

// MediaPrev simulates pressing the Previous Track media key twice to skip to the previous track.
func (a *PieButtonExecutionAdapter) MediaPrev() error {
	time.Sleep(100 * time.Millisecond)
	err := robotgo.KeyTap("audio_prev")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap audio_prev (first press) failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	err = robotgo.KeyTap("audio_prev")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap audio_prev (second press) failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

// MediaNext simulates pressing the Next Track media key.
func (a *PieButtonExecutionAdapter) MediaNext() error {
	time.Sleep(100 * time.Millisecond)
	err := robotgo.KeyTap("audio_next")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap audio_next failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

// MediaPlayPause toggles play/pause using the media key (audio_play).
func (a *PieButtonExecutionAdapter) MediaPlayPause() error {
	time.Sleep(100 * time.Millisecond)
	err := robotgo.KeyTap("audio_play")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap audio_play failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

// MediaToggleMute toggles mute using the media key (audio_mute).
func (a *PieButtonExecutionAdapter) MediaToggleMute() error {
	time.Sleep(100 * time.Millisecond)
	err := robotgo.KeyTap("audio_mute")
	if err != nil {
		log.Error("[DEBUG] robotgo.KeyTap audio_mute failed: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

func sendXButtonClick(xbutton uint32) error {
	// Press
	ret1, _, err1 := procMouseEvent.Call(
		uintptr(MOUSEEVENTF_XDOWN),
		0, 0,
		uintptr(xbutton),
		0,
	)
	if ret1 == 0 {
		return fmt.Errorf("failed to send XBUTTON DOWN: %v", err1)
	}
	// Release
	ret2, _, err2 := procMouseEvent.Call(
		uintptr(MOUSEEVENTF_XUP),
		0, 0,
		uintptr(xbutton),
		0,
	)
	if ret2 == 0 {
		return fmt.Errorf("failed to send XBUTTON UP: %v", err2)
	}
	return nil
}

// LaunchApp launches an application using its unique application name.
func LaunchApp(appNameKey string, appInfo core.AppInfo) error {

	if appInfo.URI != "" {
		return launchViaURI(appNameKey, appInfo.URI)
	}

	if appInfo.ExePath == "" {
		return fmt.Errorf("no executable path or URI for application '%s'", appNameKey)
	}

	cmd, err := buildExecCmd(appInfo.ExePath, appInfo.WorkingDirectory, appInfo.Args)
	if err != nil {
		return fmt.Errorf("cannot launch '%s', failed to prepare command: %w", appNameKey, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start executable '%s' for app '%s': %w", appInfo.ExePath, appNameKey, err)
	}

	log.Info("Successfully started application: '%s' (Path: %s, PID: %d)", appNameKey, appInfo.ExePath, cmd.Process.Pid)
	return nil
}

// CenterWindowUnderCursor centers and resizes the window under the given coordinates.
func (a *PieButtonExecutionAdapter) CenterWindowUnderCursor(x, y int) error {
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err
	}
	return CenterWindowOnMonitor(uintptr(hwnd))
}

// MaximizeWindowUnderCursor
func (a *PieButtonExecutionAdapter) MaximizeWindowUnderCursor(x, y int) error {
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err
	}

	isMax, err := hwnd.IsMaximized()
	if err != nil {
		return err
	}
	if isMax {
		return hwnd.Restore()
	}
	return hwnd.Maximize()
}

// MinimizeWindowUnderCursor
func (a *PieButtonExecutionAdapter) MinimizeWindowUnderCursor(x, y int) error {
	// NOTE: Relies on a.GetWindowAtPoint and an assumed Minimize method
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err // Original direct error return
	}
	a.lastMinimizedWindow = hwnd
	return hwnd.Minimize()
}

// RestoreLastMinimized restores the last window that was minimized using MinimizeWindowUnderCursor
func (a *PieButtonExecutionAdapter) RestoreLastMinimized() error {
	if a.lastMinimizedWindow == 0 {
		return fmt.Errorf("no window has been minimized yet")
	}
	return a.lastMinimizedWindow.Restore()
}

// CloseWindowUnderCursor
func (a *PieButtonExecutionAdapter) CloseWindowUnderCursor(x, y int) error {
	// NOTE: Relies on a.GetWindowAtPoint and the new Close method
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err
	}
	return hwnd.Close()
}

// OpenSettings opens the settings window.
func (a *PieButtonExecutionAdapter) OpenSettings() error {
	log.Info("Publishing navigation message for Settings")
	a.natsAdapter.PublishMessage(natsSubjectPieMenuNavigate, "settings")
	return nil
}

// OpenConfig opens the Pie Menu configuration window.
func (a *PieButtonExecutionAdapter) OpenConfig() error {
	log.Info("Publishing navigation message for Config")
	a.natsAdapter.PublishMessage(natsSubjectPieMenuNavigate, "piemenuConfigEditor")
	return nil
}

// FuzzySearch opens the Fuzzy Search window.
func (a *PieButtonExecutionAdapter) FuzzySearch() error {
	log.Info("Publishing navigation message for Fuzzy Search")
	a.natsAdapter.PublishMessage(natsSubjectPieMenuNavigate, "fuzzySearch")
	return nil
}
