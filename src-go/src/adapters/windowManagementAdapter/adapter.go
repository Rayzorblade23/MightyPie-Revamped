package windowManagementAdapter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	// Use lxn/win for HWND type consistency if desired, otherwise use windows.HWND
	"github.com/lxn/win" // Assuming you want to keep this for HWND type in GetFilteredListOfWindows call

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter" // Import needed here
	"github.com/nats-io/nats.go"
)

// New creates a new WindowManagementAdapter instance
func New(natsAdapter *natsAdapter.NatsAdapter) *WindowManagementAdapter {
	FetchExecutableApplicationMap()

	// Create manager and watcher using their respective constructors
	windowManager := NewWindowManager()
	windowWatcher := NewWindowWatcher()

	a := &WindowManagementAdapter{
		natsAdapter:   natsAdapter,
		winManager:    windowManager,
		stopChan:      make(chan struct{}),
		windowWatcher: windowWatcher,
	}

	// NATS Subscription for shortcut pressed events
	subject := env.Get("NATSSUBJECT_SHORTCUT_PRESSED")
	natsAdapter.SubscribeToSubject(subject, func(msg *nats.Msg) {
		var message shortcutPressed_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			logger.Printf("Failed to decode command on subject '%s': %v\n", subject, err)
			return
		}
		logger.Printf("Shortcut pressed: %d (X:%d, Y:%d)\n", message.ShortcutPressed, message.MouseX, message.MouseY)

		// // Get current windows using the refactored function and print them
		// // Pass win.HWND(0) as we don't need to exclude a specific window here.
		// currentWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0))
		// logger.Println("--- Window list at time of shortcut press ---")
		// PrintWindowList(currentWindows) // Use the helper function from manager.go
		// logger.Println("---------------------------------------------")

		// // TODO: Add logic here to handle the shortcut press based on the current window list
		// // e.g., find window under mouse, send command, etc.
	})
	logger.Printf("Subscribed to NATS subject: %s\n", subject)

	return a
}

// Run starts the adapter, including the initial window scan and monitoring loop
func (a *WindowManagementAdapter) Run() error {
	logger.Println("Starting WindowManagementAdapter...")

	// Perform initial window scan and update window info
	initialWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0))
	a.winManager.UpdateOpenWindowsInfo(initialWindows)
	logger.Printf("Initial window list created with %d windows.\n", len(initialWindows))
	PrintWindowList(initialWindows)

	// Start window watcher and monitoring goroutine
	if err := a.windowWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start window watcher: %w", err)
	}
	logger.Println("Window watcher started.")

	go a.monitorWindows()
	logger.Println("Window monitoring goroutine started.")

	// Wait for stop signal
	<-a.stopChan
	logger.Println("Received stop signal.")

	logger.Println("WindowManagementAdapter finished.")
	return nil
}

// Stop gracefully shuts down the WindowManagementAdapter
func (a *WindowManagementAdapter) Stop() {
	logger.Println("[STOP] Stopping adapter...")

	// Signal stop to main loop and monitor goroutine
	select {
	case <-a.stopChan:
		logger.Println("[STOP] stopChan already closed.")
	default:
		close(a.stopChan)
		logger.Println("[STOP] Closed stopChan.")
	}

	// Stop window watcher
	if a.windowWatcher != nil {
		a.windowWatcher.Stop()
		logger.Println("[STOP] WindowWatcher stopped.")
	} else {
		logger.Println("[STOP] WindowWatcher is nil.")
	}
}


// monitorWindows runs in a goroutine, listens for change signals, and updates the window list.
func (a *WindowManagementAdapter) monitorWindows() {
	logger.Println("[Monitor] Starting monitor loop.")
	previousWindows := a.winManager.GetOpenWindowsInfo()

	var lastUpdateTime time.Time
	const minUpdateInterval = 1 * time.Second
	var updateTimer *time.Timer
	updatePending := false
	timerChannel := make(chan time.Time)
	changeChannel := a.windowWatcher.GetChangeDetectedChannel()

	// Closure for scheduling the timer
	scheduleTimer := func(delay time.Duration, timerChannel chan time.Time) error {
		if updateTimer == nil {
			updateTimer = time.NewTimer(delay)
			go func() {
				select {
				case <-updateTimer.C:
					timerChannel <- time.Now()
				case <-a.stopChan:
				}
			}()
		} else {
			if !updateTimer.Stop() {
				select {
				case <-updateTimer.C:
				default:
				}
				updateTimer = time.NewTimer(delay)
				go func() {
					select {
					case <-updateTimer.C:
						timerChannel <- time.Now()
					case <-a.stopChan:
					}
				}()
			} else {
				updateTimer.Reset(delay)
			}
		}
		return nil
	}

	defer func() {
		if updateTimer != nil {
			updateTimer.Stop()
		}
		logger.Println("[Monitor] Exiting monitor loop.")
	}()

	for {
		select {
		case <-a.stopChan:
			return
		case <-changeChannel:
			// Handle change signals with throttling
			if time.Since(lastUpdateTime) < minUpdateInterval && !updatePending {
				delay := minUpdateInterval - time.Since(lastUpdateTime)
				if err := scheduleTimer(delay, timerChannel); err != nil {
					return // Exit if we fail to schedule timer
				}
				updatePending = true
				continue
			}

			// Process the update immediately if no throttle is needed
			updatePending = false

		case <-timerChannel:
			// Process the scheduled update after timer fires
			updatePending = false
		}

		// Update window list if necessary
		currentWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0))
		if !reflect.DeepEqual(currentWindows, previousWindows) {
			a.winManager.UpdateOpenWindowsInfo(currentWindows)
			previousWindows = currentWindows
			lastUpdateTime = time.Now()
			PrintWindowList(currentWindows)
		} else {
			lastUpdateTime = time.Now()
		}
	}
}
