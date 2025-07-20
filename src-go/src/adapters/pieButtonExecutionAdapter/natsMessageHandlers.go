package pieButtonExecutionAdapter

import (
	"encoding/json"
	"log"
	"maps"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

// --- NATS Message Handlers ---

// handlePieButtonExecuteMessage processes incoming pie button execution commands.
func (a *PieButtonExecutionAdapter) handlePieButtonExecuteMessage(msg *nats.Msg) {
	var message pieButtonExecute_Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		log.Printf("Failed to decode pieButtonExecute message: %v. Data: %s", err, string(msg.Data))
		return
	}

	if err := a.executeCommand(&message); err != nil {
		log.Printf("Failed to execute command for button %d (Type: %s): %v", message.ButtonIndex, message.ButtonType, err)
		// Optionally, publish an error response back via NATS
	}
}

// handleShortcutPressedMessage stores the mouse coordinates when a shortcut is detected.
func (a *PieButtonExecutionAdapter) handleShortcutPressedMessage(msg *nats.Msg) {
	var message core.ShortcutPressed_Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		log.Printf("Failed to decode shortcutPressed message: %v. Data: %s", err, string(msg.Data))
		return
	}

	// Acquire Lock for writing
	a.mu.Lock()
	a.lastMouseX = message.MouseX
	a.lastMouseY = message.MouseY
	a.mu.Unlock() // Release Lock

	// log.Printf("Shortcut %d pressed at X: %d, Y: %d", message.ShortcutPressed, message.MouseX, message.MouseY) // Debug logging if needed
}

// handleInstalledAppsInfoMessage updates the internal list of discovered applications
func (a *PieButtonExecutionAdapter) handleInstalledAppsInfoMessage(msg *nats.Msg) {
	var apps map[string]core.AppInfo
	if err := json.Unmarshal(msg.Data, &apps); err != nil {
		log.Printf("Failed to decode discovered apps message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	a.installedAppsInfo = apps
	a.mu.Unlock()

	log.Printf("Updated discovered apps list, %d apps tracked", len(apps))
}

// handleWindowUpdateMessage updates the internal list of active windows.
func (a *PieButtonExecutionAdapter) handleWindowUpdateMessage(msg *nats.Msg) {
	var currentWindows core.WindowsUpdate
	if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
		log.Printf("Failed to decode window update message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	clear(a.windowsList)
	maps.Copy(a.windowsList, currentWindows)
	a.mu.Unlock()

	// log.Printf("Updated windows list, %d windows tracked", len(currentWindows)) // Debug logging if needed
}
