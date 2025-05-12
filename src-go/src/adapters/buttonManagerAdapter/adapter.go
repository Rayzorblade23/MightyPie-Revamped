package buttonManagerAdapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"

	"maps"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
)

var (
	buttonConfig ConfigData
	windowsList  WindowsUpdate
	mu           sync.RWMutex
)

type ButtonManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ButtonManagerAdapter {
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	// ReadButtonConfig and PrintConfig should be adapted to handle the new ConfigData structure
	config, err := ReadButtonConfig() // Assumed function from config.go
	if err != nil {
		// Consider more robust error handling than log.Fatal in a library/adapter
		log.Fatalf("FATAL: Failed to read initial button configuration: %v", err)
	}

	mu.Lock()
	buttonConfig = config
	mu.Unlock()
	PrintConfig(config) // Assumed function from config.go

	buttonUpdateSubject := env.Get("PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE")

	a.natsAdapter.SubscribeToSubject(env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE"), func(msg *nats.Msg) {
		var currentWindows WindowsUpdate

		if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
			log.Printf("ERROR: Failed to decode window update message: %v", err)
			return
		}

		mu.Lock()
		windowsList = make(WindowsUpdate, len(currentWindows))
		maps.Copy(windowsList, currentWindows)
		mu.Unlock()

		currentConfigSnapshot := GetButtonConfig() // Assumed function from config.go, returns a deep copy or is read-only
		updatedConfig, err := a.processWindowUpdate(currentConfigSnapshot, currentWindows)
		if err != nil {
			log.Printf("ERROR: Failed to process window update for button config: %v", err)
			return
		}

		PrintConfig(updatedConfig) // Potentially verbose, useful for debugging

		if updatedConfig != nil {
			// Before publishing, update the global state if this adapter is the source of truth
			mu.Lock()
			buttonConfig = updatedConfig // Update global state
			mu.Unlock()
			log.Println("INFO: Button configuration updated and will be published.")
			a.natsAdapter.PublishMessage(buttonUpdateSubject, updatedConfig)
		} else {
			log.Println("DEBUG: No changes to button configuration after window update. No publish needed.")
		}
	})

	return a
}

// processWindowUpdate - Refactored structure
func (a *ButtonManagerAdapter) processWindowUpdate(currentConfig ConfigData, windows WindowsUpdate) (ConfigData, error) {
	if len(currentConfig) == 0 {
		log.Println("DEBUG: Skipping button processing - currentConfig is empty.")
		return nil, nil
	}
	log.Printf("DEBUG: processWindowUpdate - Starting. Initial currentConfig length: %d", len(currentConfig))

	// 1. Deep Copy Config
	updatedConfig, err := deepCopyConfig(currentConfig)
	// ... (add error handling and checks for deepCopyConfig failure as before) ...
	if err != nil || updatedConfig == nil || (len(updatedConfig) == 0 && len(currentConfig) > 0) {
		log.Printf("ERROR: processWindowUpdate - Deep copy failed or resulted in invalid state. Error: %v", err)
		return nil, fmt.Errorf("config deep copy failed: %w", err)
	}
	log.Printf("DEBUG: processWindowUpdate - Deep copy successful. Copied length: %d", len(updatedConfig))


	// 2. Handle Empty Window List Case (using JSON comparison)
	if len(windows) == 0 {
		return a.handleEmptyWindowListAndCompare(currentConfig, updatedConfig) // Use helper
	}

	// 3. Setup for Processing - Shared State
	availableWindows := make(WindowsUpdate, len(windows))
	maps.Copy(availableWindows, windows)
	processedButtons := make(map[string]bool) // Tracks buttons processed by *any* handler

	// 4. === Phase 1: Process Existing Handles and Non-Window Tasks ===
	log.Println("DEBUG: processWindowUpdate - Starting Phase 1: Process existing state...")
	for profileID, menuConfig := range updatedConfig {
		if menuConfig == nil { continue }
		for menuID, buttonMap := range menuConfig {
			if buttonMap == nil { continue }

			var originalButtonMap ButtonMap // For function call restoration
			if currentConfig[profileID] != nil && currentConfig[profileID][menuID] != nil {
				originalButtonMap = currentConfig[profileID][menuID]
			}

			// Separate tasks (could optimize later if needed)
			showProgramButtons, showAnyButtons, launchProgramButtons, functionCallButtons :=
				a.separateTasksByType(buttonMap)

			// Process tasks that DON'T consume windows first or just update state
			a.processLaunchProgramTasks(profileID, menuID, launchProgramButtons, buttonMap) // Modifies buttonMap
			a.processFunctionCallTasks(profileID, menuID, functionCallButtons, buttonMap, originalButtonMap) // Modifies buttonMap

			// Process existing handles for window-related tasks
			a.processExistingShowProgramHandles(profileID, menuID, showProgramButtons, availableWindows, processedButtons, buttonMap) // Modifies buttonMap, availableWindows, processedButtons
			a.processExistingShowAnyHandles(profileID, menuID, showAnyButtons, availableWindows, processedButtons, buttonMap)       // Modifies buttonMap, availableWindows, processedButtons
		}
	}
	log.Printf("DEBUG: processWindowUpdate - Finished Phase 1. Remaining windows: %d", len(availableWindows))


    // 5. === Phase 1.5 (NEW): Assign Matching Program Windows ===
    log.Println("DEBUG: processWindowUpdate - Starting Phase 1.5: Assign matching program windows...")
    a.assignMatchingProgramWindows(availableWindows, processedButtons, updatedConfig) // Modifies maps
    log.Printf("DEBUG: processWindowUpdate - Finished Phase 1.5. Remaining windows: %d", len(availableWindows))


	// 6. === Phase 2: Assign Remaining Windows to Available ShowAny Slots ===
	log.Println("DEBUG: processWindowUpdate - Starting Phase 2: Assign remaining windows to ShowAny slots...")
	a.assignRemainingWindows(availableWindows, processedButtons, updatedConfig) // Modifies maps
	log.Println("DEBUG: processWindowUpdate - Finished Phase 2.")


	// 7. Final Comparison (Use JSON)
	log.Println("DEBUG: processWindowUpdate - Performing final JSON comparison...")
	jsonSnapshotFinal, errSnapFinal := json.Marshal(currentConfig)
	jsonUpdatedFinal, errUpdateFinal := json.Marshal(updatedConfig)

	if errSnapFinal != nil || errUpdateFinal != nil {
		log.Printf("ERROR: Failed to marshal for final comparison (SnapErr: %v, UpdateErr: %v)", errSnapFinal, errUpdateFinal)
		return nil, fmt.Errorf("final marshal error") // Indicate error
	}

	if bytes.Equal(jsonSnapshotFinal, jsonUpdatedFinal) {
		log.Println("DEBUG: Final JSON comparison shows configurations ARE equal. Returning nil.")
		return nil, nil
	}

	log.Println("INFO: Final JSON comparison shows configurations ARE different. Returning updated config.")
	return updatedConfig, nil
}


// Helper for empty window list case
func (a *ButtonManagerAdapter) handleEmptyWindowListAndCompare(currentConfig, updatedConfig ConfigData) (ConfigData, error) {
	log.Println("DEBUG: Handling empty window list.")
	_, err := a.handleEmptyWindowList(updatedConfig) // Modifies updatedConfig directly
	if err != nil {
		log.Printf("ERROR: handleEmptyWindowList returned error: %v", err)
		return nil, fmt.Errorf("error handling empty window list: %w", err)
	}

	// Compare original with potentially modified config
	jsonSnapshot, errSnap := json.Marshal(currentConfig)
	jsonAfterClear, errClear := json.Marshal(updatedConfig)
	if errSnap != nil || errClear != nil {
		 log.Printf("ERROR: Failed to marshal for empty list comparison (SnapErr: %v, ClearErr: %v)", errSnap, errClear)
		 return nil, fmt.Errorf("marshal error during empty list check")
	}
	if bytes.Equal(jsonSnapshot, jsonAfterClear) {
		log.Println("DEBUG: JSON comparison shows config unchanged after clearing. Returning nil.")
		return nil, nil
	}
	log.Println("DEBUG: JSON comparison shows config *changed* after clearing. Returning modified config.")
	return updatedConfig, nil
}

// handleEmptyWindowList clears window-related properties from tasks.
// It modifies the provided config directly.
// Returns the modified config if changes were made, otherwise nil.
func (a *ButtonManagerAdapter) handleEmptyWindowList(configToModify ConfigData) (ConfigData, error) {
	log.Println("DEBUG: Window list is empty. Clearing existing window handles/info in button config.")
	anyChangeMade := false
	for profileID, menuConfig := range configToModify {
		if menuConfig == nil {
			continue
		}
		for menuID, buttonMap := range menuConfig {
			if buttonMap == nil {
				continue
			}
			for btnID, task := range buttonMap {
				// Create a copy to modify, then potentially assign back
				taskCopy := task // This is a struct, so it's a copy. Properties is RawMessage (slice)
				// To be perfectly safe with Properties (json.RawMessage), deep copy it if necessary,
				// but clearButtonWindowProperties typically re-marshals known structs.

				originalTaskBeforeClear := task // Keep a copy for comparison

				err := clearButtonWindowProperties(&taskCopy)
				if err != nil {
					log.Printf("ERROR: Failed to clear properties for task (Profile:%s Menu:%s Button:%s) on empty window list: %v", profileID, menuID, btnID, err)
					// Continue to attempt clearing other buttons
				} else {
					if !reflect.DeepEqual(originalTaskBeforeClear, taskCopy) {
						buttonMap[btnID] = taskCopy // Update the map with the modified task
						anyChangeMade = true
					}
				}
			}
		}
	}

	if anyChangeMade {
		log.Printf("DEBUG: Cleared window handles/info in config due to empty window list.")
		return configToModify, nil // Return the modified config
	}

	log.Printf("DEBUG: No window handles/info needed clearing on empty window list (no effective changes).")
	return nil, nil // No changes were made
}

// separateTasksByType classifies tasks in a button map by their type.
// No changes needed here as it operates on a single menu's ButtonMap.
func (a *ButtonManagerAdapter) separateTasksByType(buttonMap ButtonMap) (
	showProgram map[string]*Task,
	showAny map[string]*Task,
	launchProgram map[string]*Task,
	functionCall map[string]*Task) {

	showProgram = make(map[string]*Task)
	showAny = make(map[string]*Task)
	launchProgram = make(map[string]*Task)
	functionCall = make(map[string]*Task)

	for btnID, task := range buttonMap {
		taskCopy := task // Make a copy to take its address for the map
		taskPtr := &taskCopy

		switch TaskType(taskPtr.TaskType) {
		case TaskTypeShowProgramWindow:
			showProgram[btnID] = taskPtr
		case TaskTypeShowAnyWindow:
			showAny[btnID] = taskPtr
		case TaskTypeLaunchProgram:
			launchProgram[btnID] = taskPtr
		case TaskTypeCallFunction:
			functionCall[btnID] = taskPtr
		}
	}
	return
}

func (a *ButtonManagerAdapter) Run() error {
	fmt.Println("ButtonManagerAdapter started")
	select {}
}
