package pieButtonExecutionAdapter

import (
	"syscall"
	"time"

	"github.com/lxn/win"
)

// ShowWarningMessageBox displays a blocking MessageBox with a warning message.
// Call CloseWarningMessageBox to close it programmatically.
var warningBoxTitle = "Restoring Explorer Windows..."

func ShowWarningMessageBox() {
	msg, _ := syscall.UTF16PtrFromString("Please wait while all Explorer windows are being restored.\r\nDo not interact with Explorer until this dialog disappears.")
	title, _ := syscall.UTF16PtrFromString(warningBoxTitle)
	go func() {
		win.MessageBox(0, msg, title, win.MB_OK|win.MB_ICONWARNING|win.MB_SYSTEMMODAL)
	}()
	// Give the MessageBox time to appear
	time.Sleep(500 * time.Millisecond)
}

func CloseWarningMessageBox() {
	title, _ := syscall.UTF16PtrFromString(warningBoxTitle)
	hwnd := win.FindWindow(nil, title)
	if hwnd != 0 {
		win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
	}
}
