package pieButtonExecutionAdapter

import (
	"encoding/json"
	"maps"
	"os/exec"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

// --- NATS Message Handlers ---

// handlePieButtonExecuteMessage processes incoming pie button execution commands.
func (a *PieButtonExecutionAdapter) handlePieButtonExecuteMessage(msg *nats.Msg) {
	var message pieButtonExecute_Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		log.Error("Failed to decode pieButtonExecute message: %v. Data: %s", err, string(msg.Data))
		return
	}

	if err := a.executeCommand(&message); err != nil {
		log.Error("Failed to execute command for button %d (Type: %s): %v", message.ButtonIndex, message.ButtonType, err)
		// Optionally, publish an error response back via NATS
	}
}

// handleShortcutPressedMessage stores the mouse coordinates when a shortcut is detected.
func (a *PieButtonExecutionAdapter) handleShortcutPressedMessage(msg *nats.Msg) {
	var message core.ShortcutPressed_Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		log.Error("Failed to decode shortcutPressed message: %v. Data: %s", err, string(msg.Data))
		return
	}

	// Acquire Lock for writing
	a.mu.Lock()
	a.lastMouseX = message.MouseX
	a.lastMouseY = message.MouseY
	a.mu.Unlock() // Release Lock
}

// handleInstalledAppsInfoMessage updates the internal list of discovered applications
func (a *PieButtonExecutionAdapter) handleInstalledAppsInfoMessage(msg *nats.Msg) {
	var apps map[string]core.AppInfo
	if err := json.Unmarshal(msg.Data, &apps); err != nil {
		log.Error("Failed to decode discovered apps message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	a.installedAppsInfo = apps
	a.mu.Unlock()

	log.Info("Updated discovered apps list, %d apps tracked", len(apps))
}

func (a *PieButtonExecutionAdapter) handleWindowUpdateMessage(msg *nats.Msg) {
	var currentWindows core.WindowsUpdate
	if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
		log.Error("Failed to decode window update message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	clear(a.windowsList)
	maps.Copy(a.windowsList, currentWindows)
	a.mu.Unlock()
}

func (a *PieButtonExecutionAdapter) handleOpenFolder(msg *nats.Msg) {
	var folderType string
	if err := json.Unmarshal(msg.Data, &folderType); err != nil {
		log.Error("Failed to decode folderType message: %v. Data: %s", err, string(msg.Data))
		return
	}
	var path string
	var err error

	switch folderType {
	case "appdata":
		path, err = core.GetAppDataDir()
		if err != nil {
			log.Error("Failed to get AppData directory: %v", err)
			return
		}
	case "appfolder":
		path, err = core.GetRootDir()
		if err != nil {
			log.Error("Failed to get root dir: %v", err)
			return
		}
	default:
		log.Error("Unknown folder type received: %s", folderType)
		return
	}

	// Try to open the folder with our improved function
	if err := openFolder(path); err != nil {
		log.Error("Failed to open folder %s: %v", path, err)
	} else {
		log.Info("Successfully opened folder %s", path)
	}
}


// openFolder opens folders reliably by cleaning the path and invoking the shell.
// Implementation: `cmd /c start "" <path>` so the shell handles spaces and shell folders.
func openFolder(path string) error {
    // Clean extended-length prefix if present
    original := path
    cleanPath := path
    if len(path) > 4 && path[0:4] == "\\\\?\\" {
        cleanPath = path[4:]
        log.Debug("Stripped \\?\\ prefix: %s -> %s", original, cleanPath)
    } else {
        log.Debug("No extended prefix: %s", cleanPath)
    }

    // Use cmd start without embedding quotes; Go will quote args with spaces correctly
    log.Debug("Opening via shell: cmd /c start \"\" %s", cleanPath)
    return exec.Command("cmd", "/c", "start", "", cleanPath).Run()
}