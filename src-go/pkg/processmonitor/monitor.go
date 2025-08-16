package processmonitor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"golang.org/x/sys/windows"
)

var (
	log            = logger.New("ProcessMonitor")
	shutdownCalled = false
	shutdownMutex  sync.Mutex
	callbacks      []func()
)

// monitorProcessLoop continuously monitors a process and triggers shutdown if it terminates
func monitorProcessLoop(pid int) {
	log.Info("Monitoring process with PID: %d", pid)
	for {
		// Check if process is still running
		if !isProcessRunning(pid) {
			log.Info("Monitored process (PID: %d) has terminated, initiating shutdown", pid)
			TriggerShutdown()
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// RegisterShutdownCallback registers a function to be called when shutdown is triggered
func RegisterShutdownCallback(callback func()) {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()
	callbacks = append(callbacks, callback)
}

// TriggerShutdown calls all registered shutdown callbacks
func TriggerShutdown() {
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	// Only trigger once
	if shutdownCalled {
		return
	}
	shutdownCalled = true

	log.Info("Triggering shutdown callbacks")
	for _, callback := range callbacks {
		callback()
	}
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		// On Windows, use tasklist to check if the process is running
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH", "/FO", "CSV")
		output, err := cmd.Output()
		if err != nil {
			log.Error("Failed to execute tasklist: %v", err)
			return false
		}

		// If the output contains the PID, the process is running
		return strings.Contains(string(output), fmt.Sprintf(`"%d"`, pid))
	} else {
		// On Unix systems, we can just try to find the process
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}

		// On Unix, FindProcess always succeeds, so we need to send a signal 0 to check if the process exists
		err = process.Signal(os.Signal(nil))
		return err == nil
	}
}

// PrintProcessInfo prints information about the current process and its parent
func PrintProcessInfo() {
	currentPID := os.Getpid()
	parentPID := os.Getppid()

	fmt.Printf("Current process PID: %d\n", currentPID)
	fmt.Printf("Parent process PID: %d\n", parentPID)
}

// MonitorParentPID monitors a specific parent PID and triggers shutdown when it terminates.
// On Windows it uses a WaitForSingleObject on the process handle (no polling). On other OSes it falls back to polling.
func MonitorParentPID(pid int) {
	log.Info("Starting parent PID monitor for PID: %d", pid)
	if runtime.GOOS == "windows" {
		// Use Windows job-style wait to avoid polling
		go func() {
			// Delay import to Windows-specific path to avoid non-Windows builds failing at runtime
			// Open with SYNCHRONIZE access to wait on the handle
			// Note: requires golang.org/x/sys/windows
			if h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)); err == nil {
				defer windows.CloseHandle(h)
				// Wait indefinitely for the process to signal termination
				_, _ = windows.WaitForSingleObject(h, windows.INFINITE)
				log.Info("Parent process (PID: %d) terminated (WaitForSingleObject). Triggering shutdown.", pid)
				TriggerShutdown()
				return
			} else {
				log.Error("OpenProcess failed for PID %d: %v. Falling back to polling.", pid, err)
				monitorProcessLoop(pid)
			}
		}()
	} else {
		go monitorProcessLoop(pid)
	}
}
