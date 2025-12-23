package windowManagementAdapter

import (
	"encoding/json"
	"os"
	"reflect"
	"time"

	// Use lxn/win for HWND type consistency if desired, otherwise use windows.HWND
	"github.com/lxn/win" // Assuming you want to keep this for HWND type in GetFilteredListOfWindows call

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter" // Import needed here
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

var subjectInstalledAppsInfo = os.Getenv("PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO")

// New creates a new WindowManagementAdapter instance
func New(natsAdapter *natsAdapter.NatsAdapter) (*WindowManagementAdapter, error) {
	// Acquire write lock before populating installedAppsInfo
	installedAppsInfoMutex.Lock()
	installedAppsInfo = FetchExecutableApplicationMap()
	installedAppsInfoMutex.Unlock()

	// b, _ := json.MarshalIndent(installedAppsInfo, "", "  ")
	// logger.Debug(string(b))

	// Create manager and watcher using their respective constructors
	windowManager := NewWindowManager()
	windowWatcher := NewWindowWatcher()

	exclusionConfig, err := loadExclusionConfig()
	if err != nil {
		logger.Error("Failed to load exclusion config: %v", err)
		return nil, err
	}

	a := &WindowManagementAdapter{
		exclusionConfig: exclusionConfig,
		natsAdapter:     natsAdapter,
		winManager:      windowManager,
		stopChan:        make(chan struct{}),
		windowWatcher:   windowWatcher,
	}

	shortcutSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")

	a.publishInstalledAppsInfo(installedAppsInfo)

	// Process icons after adapter is created and trigger republish when complete
	ProcessIcons(a)

	// NATS Subscription for shortcut pressed events
	natsAdapter.SubscribeToSubject(shortcutSubject, func(msg *nats.Msg) {
		var message core.ShortcutPressed_Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			log.Error("Failed to decode command on subject '%s': %v", shortcutSubject, err)
			return
		}

		// // Get current windows using the refactored function and print them
		// // Pass win.HWND(0) as we don't need to exclude a specific window here.
		// currentWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0))
		// log.Info("--- Window list at time of shortcut press ---")
		// PrintWindowList(currentWindows) // Use the helper function from manager.go
		// log.Info("---------------------------------------------")

	})

	return a, nil
}

// publishInstalledAppsInfo sends the current discovered apps list to the NATS subject
func (a *WindowManagementAdapter) publishInstalledAppsInfo(apps map[string]core.AppInfo) {
	// Use read lock when publishing the map
	installedAppsInfoMutex.RLock()
	defer installedAppsInfoMutex.RUnlock()
	a.natsAdapter.PublishMessage(subjectInstalledAppsInfo, apps)
}

// Run starts the adapter, including the initial window scan and monitoring loop
func (a *WindowManagementAdapter) Run() error {
	log.Info("Starting WindowManagementAdapter...")

	// Perform initial window scan and update window info
	initialWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0), a.exclusionConfig)
	a.winManager.UpdateOpenWindowsInfo(initialWindows)
	log.Info("Initial window list created with %d windows", len(initialWindows))

	// Publish initial window list
	go a.publishWindowListUpdate(initialWindows)
	// PrintWindowList(initialWindows)

	// Start window watcher and monitoring goroutine
	if err := a.windowWatcher.Start(); err != nil {
		logger.Error("Failed to start window watcher: %v", err)
		return err
	}
	log.Info("Window watcher started")

	go a.monitorWindows()

	// Start focus monitoring
	go a.startFocusMonitoring()

	// Wait for stop signal
	<-a.stopChan
	log.Info("Received stop signal")

	log.Info("WindowManagementAdapter finished")
	return nil
}

// Stop gracefully shuts down the WindowManagementAdapter
func (a *WindowManagementAdapter) Stop() {
	log.Info("[STOP] Stopping adapter...")

	// Signal stop to main loop and monitor goroutine
	select {
	case <-a.stopChan:
		log.Info("[STOP] stopChan already closed.")
	default:
		close(a.stopChan)
		log.Info("[STOP] Closed stopChan.")
	}

	// Stop window watcher
	if a.windowWatcher != nil {
		a.windowWatcher.Stop()
		log.Info("[STOP] WindowWatcher stopped.")
	} else {
		log.Info("[STOP] WindowWatcher is nil.")
	}
}

// monitorWindows runs in a goroutine, listens for change signals, and updates the window list.
func (a *WindowManagementAdapter) monitorWindows() {
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
		log.Info("[Monitor] Exiting monitor loop.")
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
		currentWindows := GetFilteredListOfWindows(a.winManager, win.HWND(0), a.exclusionConfig)
		if !reflect.DeepEqual(currentWindows, previousWindows) {
			a.winManager.UpdateOpenWindowsInfo(currentWindows)
			previousWindows = currentWindows
			lastUpdateTime = time.Now()
			// PrintWindowList(currentWindows)
			a.publishWindowListUpdate(currentWindows)
		} else {
			lastUpdateTime = time.Now()
		}
	}
}

func (a *WindowManagementAdapter) publishWindowListUpdate(windows WindowMapping) {
	convertedMap := make(map[int]core.WindowInfo)
	for hwnd, info := range windows {
		convertedMap[int(hwnd)] = info
	}

	a.natsAdapter.PublishMessage(os.Getenv("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE"), convertedMap)
}
