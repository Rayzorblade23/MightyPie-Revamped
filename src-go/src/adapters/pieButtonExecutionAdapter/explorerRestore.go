package pieButtonExecutionAdapter

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/lxn/win"
)

// SetExplorerWindowPositions matches open Explorer windows by path and moves them to their saved RECT.
func SetExplorerWindowPositions(states []ExplorerWindowState) error {
	// Suggestion: If windows are not positioned correctly, increase delay in RestartAndRestoreExplorerWindows to 5-8 seconds.

	// 1. Gather all open Explorer windows (path + hwnd)
	shellUnknown, err := oleutil.CreateObject("Shell.Application")
	if err != nil {
		return fmt.Errorf("CreateObject Shell.Application failed: %w", err)
	}
	shellDisp, err := shellUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("QueryInterface failed: %w", err)
	}
	defer shellDisp.Release()

	windowsObj, err := oleutil.CallMethod(shellDisp, "Windows")
	if err != nil {
		return fmt.Errorf("Shell.Windows failed: %w", err)
	}
	windows := windowsObj.ToIDispatch()
	defer windows.Release()

	countVar, err := oleutil.GetProperty(windows, "Count")
	if err != nil {
		return fmt.Errorf("Windows.Count failed: %w", err)
	}
	count := int(countVar.Val)

	type openWin struct {
		Path string
		HWND win.HWND
	}
	var openWindows []openWin
	for i := 0; i < count; i++ {
		itemVar, err := oleutil.CallMethod(windows, "Item", i)
		if err != nil {
			continue
		}
		item := itemVar.ToIDispatch()
		defer item.Release()

		nameVar, _ := oleutil.GetProperty(item, "Name")
		name := strings.ToLower(nameVar.ToString())
		if !strings.Contains(name, "explorer") {
			continue
		}
		locVar, _ := oleutil.GetProperty(item, "LocationURL")
		path := locVar.ToString()
		if !strings.HasPrefix(path, "file:///") {
			continue
		}
		// Convert URL to Windows path and decode URL-encoded characters
		decodedPath, err := url.QueryUnescape(path[8:])
		if err != nil {
			// If decoding fails, use the original path
			decodedPath = path[8:]
		}
		winPath := strings.ReplaceAll(decodedPath, "/", "\\")
		hwndVar, _ := oleutil.GetProperty(item, "HWND")
		hwnd := win.HWND(uintptr(hwndVar.Val))
		openWindows = append(openWindows, openWin{Path: winPath, HWND: hwnd})
	}

	// Debug: print all saved states
	log.Debug("Saved Explorer window states:")
	for i, s := range states {
		log.Debug("  [%d] Path: %s Rect: %+v", i, s.Path, s.Rect)
	}
	// Debug: print all currently open Explorer windows
	log.Debug("Currently open Explorer windows:")
	for i, w := range openWindows {
		log.Debug("  [%d] Path: %s HWND: %v", i, w.Path, w.HWND)
	}

	// 2. Match by path and order
	used := make([]bool, len(openWindows))
	for _, s := range states {
		for j, w := range openWindows {
			if w.Path == s.Path && !used[j] {
				// Found the nth window for this path
				var rect win.RECT
				if win.GetWindowRect(w.HWND, &rect) {
					log.Debug("Moving HWND %v for path %s from %+v to %+v", w.HWND, w.Path, rect, s.Rect)
				}
				// Get current DPI for this window
				user32 := syscall.NewLazyDLL("user32.dll")
				getDpiForWindow := user32.NewProc("GetDpiForWindow")
				curDPI := uint32(96)
				if getDpiForWindow.Find() == nil {
					dpiRet, _, _ := getDpiForWindow.Call(uintptr(w.HWND))
					curDPI = uint32(dpiRet)
				}
				// Scale only width and height for DPI, not position
				savedDPI := s.DPI
				if savedDPI == 0 {
					savedDPI = 96
				}
				scale := float64(curDPI) / float64(savedDPI)
				left := s.Rect[0]
				top := s.Rect[1]
				width := int(float64(s.Rect[2]-s.Rect[0]) * scale)
				height := int(float64(s.Rect[3]-s.Rect[1]) * scale)
				win.MoveWindow(w.HWND, int32(left), int32(top), int32(width), int32(height), true)
				used[j] = true
				break
			}
		}
	}
	return nil
}



type ExplorerWindowState struct {
	Path string
	Rect [4]int32 // left, top, right, bottom
	DPI  uint32   // DPI at the time of capture
}

// GetExplorerWindows enumerates open Explorer windows and returns their folder paths and positions.
func GetExplorerWindows() ([]ExplorerWindowState, error) {
	var result []ExplorerWindowState

	shellUnknown, err := oleutil.CreateObject("Shell.Application")
	if err != nil {
		return nil, fmt.Errorf("CreateObject Shell.Application failed: %w", err)
	}
	shellDisp, err := shellUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("QueryInterface failed: %w", err)
	}
	defer shellDisp.Release()

	windowsObj, err := oleutil.CallMethod(shellDisp, "Windows")
	if err != nil {
		return nil, fmt.Errorf("Shell.Windows failed: %w", err)
	}
	windows := windowsObj.ToIDispatch()
	defer windows.Release()

	countVar, err := oleutil.GetProperty(windows, "Count")
	if err != nil {
		return nil, fmt.Errorf("Windows.Count failed: %w", err)
	}
	count := int(countVar.Val)

	for i := range count {
		itemVar, err := oleutil.CallMethod(windows, "Item", i)
		if err != nil {
			continue
		}
		item := itemVar.ToIDispatch()
		defer item.Release()

		nameVar, _ := oleutil.GetProperty(item, "Name")
		name := strings.ToLower(nameVar.ToString())
		if !strings.Contains(name, "explorer") {
			continue
		}
		locVar, _ := oleutil.GetProperty(item, "LocationURL")
		path := locVar.ToString()
		if !strings.HasPrefix(path, "file:///") {
			continue
		}
		// Convert URL to Windows path and decode URL-encoded characters
		decodedPath, err := url.QueryUnescape(path[8:])
		if err != nil {
			// If decoding fails, use the original path
			decodedPath = path[8:]
		}
		winPath := strings.ReplaceAll(decodedPath, "/", "\\")
		// Get HWND
		hwndVar, _ := oleutil.GetProperty(item, "HWND")
		hwnd := win.HWND(uintptr(hwndVar.Val))
		// Get window rect
		var rect win.RECT
		if win.GetWindowRect(hwnd, &rect) {
			// Get DPI for this window (Windows 10+)
			user32 := syscall.NewLazyDLL("user32.dll")
			getDpiForWindow := user32.NewProc("GetDpiForWindow")
			var dpi uint32 = 96 // Fallback
			if getDpiForWindow.Find() == nil {
				dpiRet, _, _ := getDpiForWindow.Call(uintptr(hwnd))
				dpi = uint32(dpiRet)
			}
			result = append(result, ExplorerWindowState{
				Path: winPath,
				Rect: [4]int32{rect.Left, rect.Top, rect.Right, rect.Bottom},
				DPI:  dpi,
			})
		} else {
			// fallback: store with dummy rect
			result = append(result, ExplorerWindowState{Path: winPath, Rect: [4]int32{0, 0, 0, 0}})
		}
	}
	return result, nil
}

// RestartExplorer kills and restarts explorer.exe
func RestartExplorer() error {
	exec.Command("taskkill", "/F", "/IM", "explorer.exe").Run()
	time.Sleep(1 * time.Second)
	return exec.Command("explorer.exe").Start()
}

// RestoreExplorerWindows opens a new Explorer window for each saved path
func RestoreExplorerWindows(states []ExplorerWindowState) error {
	for _, state := range states {
		exec.Command("explorer.exe", state.Path).Start()
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
