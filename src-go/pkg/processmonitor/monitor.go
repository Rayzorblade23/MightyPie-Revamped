package processmonitor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
)

var (
	log            = logger.New("ProcessMonitor")
	shutdownCalled = false
	shutdownMutex  sync.Mutex
	callbacks      []func()
)

// MonitorParentProcess checks if the parent process is still running
// and triggers a shutdown if it's not
func MonitorParentProcess() {
	// Get the app name from environment variable or use default
	appName := os.Getenv("PUBLIC_APPNAME")
	if appName == "" {
		appName = "MightyPieRevamped"
	}
	
	log.Info("Starting process monitor for Tauri parent: %s, PID: %d", appName, os.Getppid())

	// Start monitoring in a separate goroutine
	go func() {

		log.Info("Starting process monitoring...")
		
		// Find the main application process by name
		pid, err := findProcessByName(appName)
		if err != nil {
			log.Error("Failed to find main application process: %v", err)
			log.Warn("Process monitoring disabled")
			return
		}
		
		if pid <= 0 {
			log.Error("Main application process not found")
			log.Warn("Process monitoring disabled")
			return
		}
		
		log.Info("Found main application process with PID: %d", pid)
		
		// Start monitoring the process
		monitorProcessLoop(pid)
	}()
}

// findProcessByName searches for a process by name and returns its PID
func findProcessByName(name string) (int, error) {
	if runtime.GOOS == "windows" {
		// On Windows, use tasklist to find the process
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", name), "/NH", "/FO", "CSV")
		output, err := cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("failed to execute tasklist: %v", err)
		}
		
		// Parse the output to find the PID
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Remove quotes and split by comma
			line = strings.Trim(line, "\r\n")
			if line == "" {
				continue
			}
			
			parts := strings.Split(strings.Trim(line, "\""), "\",\"")
			if len(parts) >= 2 {
				// The second column is the PID
				pidStr := parts[1]
				pid, err := strconv.Atoi(pidStr)
				if err != nil {
					continue
				}
				return pid, nil
			}
		}
		
		return 0, fmt.Errorf("process not found")
	} else {
		// On Unix systems, use ps and grep
		cmd := exec.Command("sh", "-c", fmt.Sprintf("ps -ef | grep %s | grep -v grep | awk '{print $2}'", name))
		output, err := cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("failed to execute ps command: %v", err)
		}
		
		// Parse the output to find the PID
		pidStr := strings.TrimSpace(string(output))
		if pidStr == "" {
			return 0, fmt.Errorf("process not found")
		}
		
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse PID: %v", err)
		}
		
		return pid, nil
	}
}

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
