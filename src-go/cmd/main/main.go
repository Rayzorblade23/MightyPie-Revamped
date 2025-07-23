package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

func main() {
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
				cmd.Process.Kill()
			}
		}
		os.Exit(0)
	}()

	wg.Wait()
}
