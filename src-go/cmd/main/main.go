package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

var natsCmd *exec.Cmd

func main() {
	// Start NATS server and wait for it to be ready
	err := startNatsServer()
	if err != nil {
		log.Fatalf("Failed to start NATS server: %v", err)
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
	var cmds []*exec.Cmd

	// Add the NATS command to the list of commands to be managed
	if natsCmd != nil {
		cmds = append(cmds, natsCmd)
	}

	for _, worker := range workers {
		wg.Add(1)
		go func(workerName string) {
			defer wg.Done()
			// Determine the executable path based on the OS
			var exePath string
			binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				fmt.Printf("Error getting bin directory: %v\n", err)
				return
			}

			if runtime.GOOS == "windows" {
				exePath = filepath.Join(binDir, workerName+".exe")
			} else {
				exePath = filepath.Join(binDir, workerName)
			}

			cmd := exec.Command(exePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmds = append(cmds, cmd)

			fmt.Printf("Starting worker: %s\n", workerName)
			if err := cmd.Run(); err != nil {
				fmt.Printf("Worker %s failed: %v\n", workerName, err)
			}
		}(worker)
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down all workers...")
		for _, cmd := range cmds {
			if cmd.Process != nil {
				err := cmd.Process.Kill()
				if err != nil {
					fmt.Printf("Failed to kill process %d: %v\n", cmd.Process.Pid, err)
				}
			}
		}
		os.Exit(0)
	}()

	wg.Wait()
}

func startNatsServer() error {
	var natsPath, confPath string

	var natsExe string
	if runtime.GOOS == "windows" {
		natsExe = "nats-server-x86_64-pc-windows-msvc.exe"
	} else {
		natsExe = "nats-server"
	}

	// Use APP_ENV to determine which paths to use.
	if os.Getenv("APP_ENV") == "development" {
		// Development: Use paths relative to the project root.
		fmt.Println("Development environment detected. Using dev paths for NATS.")
		rootDir, err := core.GetRootDir()
		if err != nil {
			return fmt.Errorf("error getting project root for dev: %w", err)
		}
		natsPath = filepath.Join(rootDir, "scripts", "nats-server", natsExe)
		confPath = filepath.Join(rootDir, "scripts", "nats.conf")
	} else {
		// Production: Paths are relative to the running executable.
		fmt.Println("Production environment detected. Using bundled paths for NATS.")
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not get executable path for prod: %w", err)
		}
		binDir := filepath.Dir(exePath)
		natsPath = filepath.Join(binDir, natsExe)
		confPath = filepath.Join(binDir, "nats.conf")
	}

	// For production, the token should be handled securely.
	// For now, we'll try to get it from an environment variable.
	natsToken := os.Getenv("NATS_AUTH_TOKEN")
	if natsToken == "" {
		return fmt.Errorf("NATS_AUTH_TOKEN environment variable not set")
	}

	natsCmd = exec.Command(natsPath, "-c", confPath, "--auth", natsToken)
	natsCmd.Stdout = os.Stdout
	natsCmd.Stderr = os.Stderr

	fmt.Println("Starting NATS server...")
	if err := natsCmd.Start(); err != nil {
		return fmt.Errorf("could not start NATS server: %w", err)
	}

	// Wait for NATS to be ready
	fmt.Println("Waiting for NATS server to be ready...")
	maxWaitTime := 10 * time.Second
	startTime := time.Now()
	for time.Since(startTime) < maxWaitTime {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:4222", 1*time.Second)
		if err == nil {
			conn.Close()
			fmt.Println("NATS server is ready.")
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("NATS server did not become ready in time")
}
