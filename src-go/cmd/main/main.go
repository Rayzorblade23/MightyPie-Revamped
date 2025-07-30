package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/Rayzorblade23/MightyPie-Revamped/pkg/processmonitor"
)

var (
	natsCmd *exec.Cmd
	cmds    []*exec.Cmd
)

func main() {
	// Initialize structured logger
	log := logger.New("Main")
	logger.ReplaceStdLog("Main")
	log.Info("Starting MightyPie...")
	log.Info("Log Level: %s", os.Getenv("RUST_LOG"))
	
	// Start parent process monitoring
	processmonitor.MonitorParentProcess()
	
	// Register cleanup function for when parent process exits
	processmonitor.RegisterShutdownCallback(func() {
		log.Info("Parent process terminated, cleaning up...")
		cleanupAllProcesses(log)
		os.Exit(0)
	})

	// Start NATS server and wait for it to be ready
	err := startNatsServer(log)
	if err != nil {
		log.Fatal("Failed to start NATS server: %v", err)
	}

	workers := []string{
		"buttonManager",
		"mouseInputHandler",
		"pieButtonExecutor",
		"settingsManager",
		"shortcutDetector",
		"shortcutSetter",
		"windowManagement",
	}

	var wg sync.WaitGroup
		// Prepare all commands before starting them to avoid race conditions.
	
	if natsCmd != nil {
		cmds = append(cmds, natsCmd)
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal("Error getting bin directory: %v", err)
	}

	for _, workerName := range workers {
		var exePath string
		if runtime.GOOS == "windows" {
			exePath = filepath.Join(binDir, workerName+".exe")
		} else {
			exePath = filepath.Join(binDir, workerName)
		}
		cmd := exec.Command(exePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmds = append(cmds, cmd)
	}

	// Launch all processes in goroutines.
	for _, cmd := range cmds {
		wg.Add(1)
		go func(c *exec.Cmd) {
			defer wg.Done()
			log.Info("Starting process: %s", c.Path)
			if err := c.Start(); err != nil {
				log.Error("Process %s failed to start: %v", c.Path, err)
				return
			}

			if err := c.Wait(); err != nil {
				// A "signal: killed" error is expected on graceful shutdown, so we check for it.
				if !strings.Contains(err.Error(), "killed") {
					log.Error("Process %s exited with error: %v", c.Path, err)
				}
			} else {
				log.Info("Process %s exited successfully", c.Path)
			}
		}(cmd)
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		
		for _, cmd := range cmds {
			if cmd.Process != nil {
				err := cmd.Process.Kill()
				if err != nil {
					log.Error("Failed to kill process %d: %v", cmd.Process.Pid, err)
				}
			}
		}
		os.Exit(0)
	}()

	wg.Wait()
}

// startNatsServer initializes and starts the NATS server.
// It orchestrates path resolution, directory setup, configuration, and launching the server process.
func startNatsServer(log *logger.Logger) error {
	natsExePath, defaultConfPath, err := getNatsPaths(log)
	if err != nil {
		return err
	}

	_, natsDataDir, userConfPath, err := setupNatsDirectories()
	if err != nil {
		return err
	}

	if err := ensureUserNatsConfig(log, userConfPath, defaultConfPath, natsDataDir); err != nil {
		return err
	}

	if err := launchNatsProcess(log, natsExePath, userConfPath); err != nil {
		return err
	}

	return waitForNatsReady(log)
}

// getNatsPaths determines the paths for the NATS executable and default config based on the environment.
func getNatsPaths(log *logger.Logger) (natsExePath, defaultConfPath string, err error) {
	natsExe := "nats-server"
	if runtime.GOOS == "windows" {
		natsExe = "nats-server.exe"
	}

	if os.Getenv("APP_ENV") == "development" {
		log.Info("Development environment: using dev paths for NATS.")
		rootDir := os.Getenv("MIGHTYPIE_ROOT_DIR")
		if rootDir == "" {
			return "", "", fmt.Errorf("MIGHTYPIE_ROOT_DIR environment variable not set")
		}
		natsExePath = filepath.Join(rootDir, "src-tauri", "assets", "nats-server", natsExe)
		defaultConfPath = filepath.Join(rootDir, "src-tauri", "assets", "nats-server", "nats.conf")
	} else {
		log.Info("Production environment: using bundled paths for NATS.")
		natsExePath = filepath.Join(os.Getenv("MIGHTYPIE_ROOT_DIR"), "assets", "nats-server", natsExe)
		defaultConfPath = filepath.Join(os.Getenv("MIGHTYPIE_ROOT_DIR"), "assets", "nats-server", "nats.conf")
	}
	return natsExePath, defaultConfPath, nil
}

// setupNatsDirectories creates the necessary NATS directories in the AppData folder.
func setupNatsDirectories() (natsAppDataDir, natsDataDir, userConfPath string, err error) {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return "", "", "", fmt.Errorf("could not get AppData directory: %w", err)
	}

	natsAppDataDir = filepath.Join(appDataDir, "nats")
	if err := os.MkdirAll(natsAppDataDir, 0755); err != nil {
		return "", "", "", fmt.Errorf("could not create NATS AppData directory: %w", err)
	}

	natsDataDir = filepath.Join(natsAppDataDir, "data")
	if err := os.MkdirAll(natsDataDir, 0755); err != nil {
		return "", "", "", fmt.Errorf("could not create NATS data directory: %w", err)
	}

	userConfPath = filepath.Join(natsAppDataDir, "nats.conf")
	return natsAppDataDir, natsDataDir, userConfPath, nil
}

// ensureUserNatsConfig checks for a user-specific NATS config and creates one from the default if not found.
func ensureUserNatsConfig(log *logger.Logger, userConfPath, defaultConfPath, natsDataDir string) error {
	if _, err := os.Stat(userConfPath); os.IsNotExist(err) {
		log.Info("NATS config not found in AppData, creating from default...")
		defaultConfig, err := os.ReadFile(defaultConfPath)
		if err != nil {
			return fmt.Errorf("could not read default NATS config: %w", err)
		}

		natsDataDirSlash := filepath.ToSlash(natsDataDir)
		configStr := strings.Replace(string(defaultConfig), "store_dir=\"natsdata\"", fmt.Sprintf("store_dir=\"%s\"", natsDataDirSlash), 1)

		if err := os.WriteFile(userConfPath, []byte(configStr), 0644); err != nil {
			return fmt.Errorf("could not write NATS config to AppData: %w", err)
		}
		log.Info("NATS config created in AppData: %s", userConfPath)
	} else if err != nil {
		return fmt.Errorf("error checking for NATS config in AppData: %w", err)
	} else {
		log.Info("Using existing NATS config from AppData: %s", userConfPath)
	}
	return nil
}

// launchNatsProcess starts the NATS server process with the specified configuration.
func launchNatsProcess(log *logger.Logger, natsExePath, userConfPath string) error {
	natsToken := os.Getenv("NATS_AUTH_TOKEN")
	if natsToken == "" {
		return fmt.Errorf("NATS_AUTH_TOKEN environment variable not set")
	}

	log.Info("Starting NATS server with config: %s", userConfPath)
	natsCmd = exec.Command(natsExePath, "-c", userConfPath, "--auth", natsToken)
	natsCmd.Stdout = os.Stdout
	natsCmd.Stderr = os.Stderr

	log.Info("Starting NATS server...")
	if err := natsCmd.Start(); err != nil {
		return fmt.Errorf("could not start NATS server: %w", err)
	}
	return nil
}

// waitForNatsReady waits for the NATS server to become responsive.
func waitForNatsReady(log *logger.Logger) error {
	log.Info("Waiting for NATS server to be ready...")
	for range 5 {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:4222", 1*time.Second)
		if err == nil {
			conn.Close()
			log.Info("NATS server is ready.")
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("NATS server did not start in time")
}

func cleanupAllProcesses(log *logger.Logger) {
	log.Info("Cleaning up all processes...")
		for _, cmd := range cmds {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				log.Error("Failed to kill process %d: %v", cmd.Process.Pid, err)
			}
		}
	}
}
