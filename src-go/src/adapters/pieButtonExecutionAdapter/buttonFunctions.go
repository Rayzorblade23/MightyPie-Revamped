package pieButtonExecutionAdapter

import (
	"fmt"
	"log"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

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

	log.Printf("Successfully started application: '%s' (Path: %s, PID: %d)", appNameKey, appInfo.ExePath, cmd.Process.Pid)
	return nil
}

// MaximizeWindow - Original method implementation
func (a *PieButtonExecutionAdapter) MaximizeWindow(x, y int) error {
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err
	}

	return hwnd.Maximize()
}

// MinimizeWindow - Original method implementation
func (a *PieButtonExecutionAdapter) MinimizeWindow(x, y int) error {
	// NOTE: Relies on a.GetWindowAtPoint and an assumed Minimize method
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err // Original direct error return
	}

	return hwnd.Minimize()
}

// CloseWindow - Original standalone function implementation
func CloseWindow() error {
	fmt.Println("Closing window")
	return nil
}
