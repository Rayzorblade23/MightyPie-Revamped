package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"github.com/nats-io/nats-server/v2/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/buttonManagerAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/mouseInputAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/pieButtonExecutionAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/settingsManagerAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/shortcutDetectionAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/shortcutSetterAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/windowManagementAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/Rayzorblade23/MightyPie-Revamped/pkg/processmonitor"
)

var (
	natsServer *server.Server
	cmds    []*os.Process

	// Worker flags
	workerFlags = map[string]*bool{
		"buttonManager":     flag.Bool("buttonManager", false, "Run as button manager worker"),
		"mouseInputHandler": flag.Bool("mouseInputHandler", false, "Run as mouse input handler worker"),
		"pieButtonExecutor": flag.Bool("pieButtonExecutor", false, "Run as pie button executor worker"),
		"settingsManager":   flag.Bool("settingsManager", false, "Run as settings manager worker"),
		"shortcutDetector":  flag.Bool("shortcutDetector", false, "Run as shortcut detector worker"),
		"shortcutSetter":    flag.Bool("shortcutSetter", false, "Run as shortcut setter worker"),
		"windowManagement":  flag.Bool("windowManagement", false, "Run as window management worker"),
	}
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Initialize structured logger
	log := logger.New("Main")
	logger.ReplaceStdLog("Main")
	
	// Check if we should run as a specific worker
	for workerName, flagValue := range workerFlags {
		if *flagValue {
			// Workers only log a single line
			runWorker(workerName)
			return
		}
	}

	// Only the main coordinator logs these messages
	log.Info("Starting MightyPie backend...")
	log.Info("Log Level: %s", os.Getenv("RUST_LOG"))
	
	// If no worker flag is set, run as the main coordinator
	log.Info("Running as main coordinator")
	
	// Start parent process monitoring
	processmonitor.MonitorParentProcess()
	
	// Register cleanup function for when parent process exits
	processmonitor.RegisterShutdownCallback(func() {
		log.Info("Parent process terminated, cleaning up...")
		cleanupAllProcesses(log)
		os.Exit(0)
	})

	// Start NATS server and wait for it to be ready
	// Only the main coordinator starts the NATS server
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
	
	// No external natsCmd needed for embedded server.


	// Get the executable path for launching worker processes
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Error getting executable path: %v", err)
	}

	for _, workerName := range workers {
		// Create a process that runs this same executable with the appropriate worker flag
		procAttr := &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Env:   append(os.Environ(), "MIGHTYPIE_WORKER_TYPE=worker"),
		}
		proc, err := os.StartProcess(exePath, []string{exePath, fmt.Sprintf("--%s", workerName)}, procAttr)
		if err != nil {
			log.Error("Failed to start worker %s: %v", workerName, err)
			continue
		}
		cmds = append(cmds, proc)
	}

	// Launch all processes in goroutines.
	for _, proc := range cmds {
		wg.Add(1)
		go func(p *os.Process) {
			defer wg.Done()
			// No process launch needed for embedded NATS
			if p != nil {
				state, err := p.Wait()
				if err != nil {
					log.Error("Process %d exited with error: %v", p.Pid, err)
				} else {
					log.Info("Process %d exited: %v", p.Pid, state)
				}
			}
		}(proc)
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		
		for _, proc := range cmds {
			if proc != nil {
				err := proc.Kill()
				if err != nil {
					log.Error("Failed to kill process %d: %v", proc.Pid, err)
				}
			}
		}
		// Stop embedded NATS server if running
		if natsServer != nil {
			log.Info("[NATS] Shutting down embedded NATS server...")
			natsServer.Shutdown()
		}
		os.Exit(0)
	}()

	wg.Wait()
}

// startNatsServer initializes and starts the embedded NATS server.
// It orchestrates path resolution, directory setup, configuration, and launching the embedded server.
func startNatsServer(log *logger.Logger) error {
	log.Debug("[NATS] Preparing to start embedded NATS server...")
	// Log the actual listen URL from config after parsing
	defaultConfPath, err := getNatsConfigPath(log)
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

	log.Debug("[NATS] Parsing NATS config from: %s", userConfPath)
	// Read config file for embedded NATS
	opts, err := server.ProcessConfigFile(userConfPath)
	if err != nil {
		return fmt.Errorf("failed to parse NATS config: %w", err)
	}
	log.Debug("[NATS] NATS config parsed successfully.")
	log.Info("[NATS] Embedded NATS server will listen on: nats://%s:%d", opts.Host, opts.Port)

	// Set auth token from env if present
	natsToken := os.Getenv("NATS_AUTH_TOKEN")
	if natsToken != "" {
		opts.Authorization = natsToken
	}

	natsPort := os.Getenv("NATS_PORT")
	if natsPort != "" {
		if port, err := strconv.Atoi(natsPort); err == nil {
			opts.Port = port
		}
	}

	log.Debug("[NATS] Starting embedded NATS server on port %d...", opts.Port)
	natsServer = server.New(opts)
	if natsServer == nil {
		return fmt.Errorf("failed to create embedded NATS server")
	}

	go natsServer.Start()

	// Wait for server to be ready
	for range 10 {
		if natsServer.ReadyForConnections(100 * time.Millisecond) {
			log.Info("[NATS] Embedded NATS server is ready.")
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("embedded NATS server did not start in time")
}


// getNatsConfigPath determines the path for the default NATS config based on the environment.
func getNatsConfigPath(log *logger.Logger) (defaultConfPath string, err error) {
	if os.Getenv("APP_ENV") == "development" {
		log.Info("Development environment: using dev paths for NATS.")
		rootDir := os.Getenv("MIGHTYPIE_ROOT_DIR")
		if rootDir == "" {
			return "", fmt.Errorf("MIGHTYPIE_ROOT_DIR environment variable not set")
		}
		defaultConfPath = filepath.Join(rootDir, "src-tauri", "assets", "data", "nats.conf")
	} else {
		log.Info("Production environment: using bundled paths for NATS.")
		defaultConfPath = filepath.Join(os.Getenv("MIGHTYPIE_ROOT_DIR"), "assets", "data", "nats.conf")
	}
	return defaultConfPath, nil
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
		log.Info("[NATS] NATS config not found in AppData, creating from default...")
		defaultConfig, err := os.ReadFile(defaultConfPath)
		if err != nil {
			return fmt.Errorf("could not read default NATS config: %w", err)
		}

		natsDataDirSlash := filepath.ToSlash(natsDataDir)
		configStr := strings.Replace(string(defaultConfig), "store_dir=\"natsdata\"", fmt.Sprintf("store_dir=\"%s\"", natsDataDirSlash), 1)

		if err := os.WriteFile(userConfPath, []byte(configStr), 0644); err != nil {
			return fmt.Errorf("could not write NATS config to AppData: %w", err)
		}
		log.Info("[NATS] NATS config created in AppData: %s", userConfPath)
	} else if err != nil {
		return fmt.Errorf("error checking for NATS config in AppData: %w", err)
	}
	return nil
}

func cleanupAllProcesses(log *logger.Logger) {
	log.Info("Cleaning up all processes...")
	for _, proc := range cmds {
		if proc != nil {
			err := proc.Kill()
			if err != nil {
				log.Error("Failed to kill process %d: %v", proc.Pid, err)
			}
		}
	}
}

// runWorker runs the specified worker type
func runWorker(workerType string) {
	// Use Unicode-correct title casing for worker name
	c := cases.Title(language.Und)
	workerTitle := c.String(workerType)
	log := logger.New(workerTitle)
	logger.ReplaceStdLog(workerTitle)

	// Wait for NATS server to be ready before connecting
	// Use a shorter timeout and less verbose logging
	for i := range 5 { // Try for up to 5 seconds
		conn, err := net.DialTimeout("tcp", "127.0.0.1:4222", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if i == 4 {
			log.Fatal("NATS server did not start in time")
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create a NATS adapter for the worker
	natsAdapter, err := natsAdapter.New(strings.Title(workerType))
	if err != nil {
		log.Fatal("Failed to connect to NATS: %v", err)
	}

	// Initialize and run the appropriate worker based on type
	switch workerType {
	case "buttonManager":
		buttonManager := buttonManagerAdapter.New(natsAdapter)
		buttonManager.Run()
	case "mouseInputHandler":
		mouseInputAdapter := mouseInputAdapter.New(natsAdapter)
		mouseInputAdapter.Run()
	case "pieButtonExecutor":
		pieButtonExecutor := pieButtonExecutionAdapter.New(natsAdapter)
		pieButtonExecutor.Run()
	case "settingsManager":
		settingsManager := settingsManagerAdapter.New(natsAdapter)
		settingsManager.Run()
	case "shortcutDetector":
		shortcutDetectionAdapter := shortcutDetectionAdapter.New(natsAdapter)
		shortcutDetectionAdapter.Run()
	case "shortcutSetter":
		shortcutSetterAdapter := shortcutSetterAdapter.New(natsAdapter)
		shortcutSetterAdapter.Run()
	case "windowManagement":
		windowManagement, err := windowManagementAdapter.New(natsAdapter)
		if err != nil {
			log.Fatal("Failed to create WindowManagementAdapter: %v", err)
		}
		windowManagement.Run()
	default:
		panic("Unknown worker type: " + workerType)
	}
}
