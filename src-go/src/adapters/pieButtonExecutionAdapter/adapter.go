// Package pieButtonExecutionAdapter handles the logic triggered by pie menu button actions.
// It listens to NATS messages for button executions, shortcut presses (to get context like mouse position),
// and window updates, dispatching actions accordingly.
package pieButtonExecutionAdapter

import (
	"fmt"
	"log"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// NATS Subjects - fetched from environment, consider constants if these are static.
var (
	natsSubjectPieButtonExecute    = env.Get("PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE")
	natsSubjectShortcutPressed     = env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	natsSubjectWindowManagerUpdate = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	natsSubjectInstalledAppsInfo   = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO")
	natsSubjectPieMenuNavigate     = env.Get("PUBLIC_NATSSUBJECT_PIEMENU_NAVIGATE")
)

// PieButtonExecutionAdapter listens to NATS events and executes actions.
type PieButtonExecutionAdapter struct {
	natsAdapter         *natsAdapter.NatsAdapter
	lastMouseX          int
	lastMouseY          int
	mu                  sync.RWMutex // Protects access to windowsList
	windowsList         core.WindowsUpdate
	installedAppsInfo   map[string]core.AppInfo
	functionHandlers    map[string]ButtonFunctionExecutor
	lastMinimizedWindow WindowHandle

	lastExplorerWindowHWND WindowHandle // Stores the HWND of the last Explorer window brought to foreground
}

// --- Adapter Implementation ---

// New creates and initializes a new PieButtonExecutionAdapter.
func New(natsAdapter *natsAdapter.NatsAdapter) *PieButtonExecutionAdapter {
	a := &PieButtonExecutionAdapter{
		natsAdapter:       natsAdapter,
		windowsList:       make(core.WindowsUpdate),
		installedAppsInfo: make(map[string]core.AppInfo),
	}

	a.functionHandlers = map[string]ButtonFunctionExecutor{
		"Maximize":               CoordinatesButtonFunctionExecutor{fn: a.MaximizeWindowUnderCursor},
		"Minimize":               CoordinatesButtonFunctionExecutor{fn: a.MinimizeWindowUnderCursor},
		"Close Window":           CoordinatesButtonFunctionExecutor{fn: a.CloseWindowUnderCursor},
		"Center Window":          CoordinatesButtonFunctionExecutor{fn: a.CenterWindowUnderCursor},
		"Restore Last Minimized": NoArgButtonFunctionExecutor{fn: a.RestoreLastMinimized},
		"Forwards":               NoArgButtonFunctionExecutor{fn: a.ForwardsButtonClick},
		"Backwards":              NoArgButtonFunctionExecutor{fn: a.BackwardsButtonClick},
		"Copy":                   NoArgButtonFunctionExecutor{fn: a.Copy},
		"Paste":                  NoArgButtonFunctionExecutor{fn: a.Paste},
		"Clipboard":              NoArgButtonFunctionExecutor{fn: a.OpenClipboard},
		"Fullscreen (F11)":       NoArgButtonFunctionExecutor{fn: a.Fullscreen_F11},
		// Media
		"Previous Track":               NoArgButtonFunctionExecutor{fn: a.MediaPrev},
		"Next Track":                   NoArgButtonFunctionExecutor{fn: a.MediaNext},
		"Play/Pause":                   NoArgButtonFunctionExecutor{fn: a.MediaPlayPause},
		"Mute":                         NoArgButtonFunctionExecutor{fn: a.MediaToggleMute},
		"Most Recent Explorer Window":  NoArgButtonFunctionExecutor{fn: a.BringLastExplorerWindowToForeground},
		"Show All Explorer Windows":    NoArgButtonFunctionExecutor{fn: a.BringAllExplorerWindowsToForeground},
		"Restart Explorer": NoArgButtonFunctionExecutor{fn: a.RestartAndRestoreExplorerWindows},
		// New functions
		"Open Settings":  NoArgButtonFunctionExecutor{fn: a.OpenSettings},
		"Open Config":    NoArgButtonFunctionExecutor{fn: a.OpenConfig},
		"Fuzzy Search":   NoArgButtonFunctionExecutor{fn: a.FuzzySearch},
		// Add more function handlers here as needed
	}

	ValidateFunctionHandlers(a.functionHandlers)

	a.subscribeToEvents() // Setup NATS subscriptions

	return a
}

// subscribeToEvents sets up all necessary NATS subscriptions.
func (a *PieButtonExecutionAdapter) subscribeToEvents() {
	a.natsAdapter.SubscribeToSubject(natsSubjectPieButtonExecute, core.GetTypeName(a), a.handlePieButtonExecuteMessage)
	a.natsAdapter.SubscribeToSubject(natsSubjectShortcutPressed, core.GetTypeName(a), a.handleShortcutPressedMessage)
	a.natsAdapter.SubscribeToSubject(natsSubjectWindowManagerUpdate, core.GetTypeName(a), a.handleWindowUpdateMessage)
	a.natsAdapter.SubscribeToSubject(natsSubjectInstalledAppsInfo, core.GetTypeName(a), a.handleInstalledAppsInfoMessage)
}

// executeCommand dispatches the command based on the ButtonType.
func (a *PieButtonExecutionAdapter) executeCommand(executionInfo *pieButtonExecute_Message) error {
	log.Printf("Executing command for button %d: ButtonType=%s", executionInfo.ButtonIndex, executionInfo.ButtonType)

	switch executionInfo.ButtonType {
	case ButtonTypeShowProgramWindow:
		return a.handleShowProgramWindow(executionInfo)
	case ButtonTypeShowAnyWindow:
		return a.handleShowAnyWindow(executionInfo)
	case ButtonTypeCallFunction:
		return a.handleCallFunction(executionInfo)
	case ButtonTypeLaunchProgram:
		log.Printf("Button %d - Launching program: %s", executionInfo.ButtonIndex, executionInfo.ButtonType)
		return a.handleLaunchProgram(executionInfo)
	case ButtonTypeDisabled:
		log.Printf("Button %d is disabled, doing nothing.", executionInfo.ButtonIndex)
		return nil // Nothing to do for disabled buttons
	default:
		return fmt.Errorf("unknown button type: %s", executionInfo.ButtonType)
	}
}

// Run starts the adapter's main loop (currently just blocks).
func (a *PieButtonExecutionAdapter) Run() error {
	log.Println("PieButtonExecutionAdapter started and listening for events.")
	// Blocks indefinitely, keeping the adapter alive to process NATS messages.
	select {}
}
