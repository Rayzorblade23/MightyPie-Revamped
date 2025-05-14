// Package pieButtonExecutionAdapter handles the logic triggered by pie menu button actions.
// It listens to NATS messages for button executions, shortcut presses (to get context like mouse position),
// and window updates, dispatching actions accordingly.
package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"log" // Use the standard log package
	"maps"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd" // Assuming this provides environment variables
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

// NATS Subjects - fetched from environment, consider constants if these are static.
var (
	natsSubjectPieButtonExecute    = env.Get("PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE")
	natsSubjectShortcutPressed     = env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	natsSubjectWindowManagerUpdate = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	natsSubjectDiscoveredApps      = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_APPSDISCOVERED")
)

// Function names - fetched from environment, consider constants if static.
var (
	fnMaximize = env.Get("PUBLIC_FN_TEXT_MAXIMIZE")
	fnMinimize = env.Get("PUBLIC_FN_TEXT_MINIMIZE")
	fnClose    = env.Get("PUBLIC_FN_TEXT_CLOSE")
	fnTopmost  = env.Get("PUBLIC_FN_TEXT_TOPMOST")
)

// PieButtonExecutionAdapter listens to NATS events and executes actions.
type PieButtonExecutionAdapter struct {
	natsAdapter      *natsAdapter.NatsAdapter
	lastMouseX       int
	lastMouseY       int
	mu               sync.RWMutex // Protects access to windowsList
	windowsList      WindowsUpdate
	discoveredApps   map[string]AppLaunchInfo
	functionHandlers map[string]HandlerWrapper
}

// --- Adapter Implementation ---

// New creates and initializes a new PieButtonExecutionAdapter.
func New(natsAdapter *natsAdapter.NatsAdapter) *PieButtonExecutionAdapter {
	a := &PieButtonExecutionAdapter{
		natsAdapter:    natsAdapter,
		windowsList:    make(WindowsUpdate),
		discoveredApps: make(map[string]AppLaunchInfo),
	}

	a.functionHandlers = a.initFunctionHandlers() // Initialize handlers
	a.subscribeToEvents()                         // Setup NATS subscriptions

	return a
}

// subscribeToEvents sets up all necessary NATS subscriptions.
func (a *PieButtonExecutionAdapter) subscribeToEvents() {
	a.subscribe(natsSubjectPieButtonExecute, a.handlePieButtonExecuteMessage)
	a.subscribe(natsSubjectShortcutPressed, a.handleShortcutPressedMessage)
	a.subscribe(natsSubjectWindowManagerUpdate, a.handleWindowUpdateMessage)
	a.subscribe(natsSubjectDiscoveredApps, a.handleDiscoveredAppsMessage)
}

// subscribe is a helper to subscribe to a NATS subject with unified error logging.
func (a *PieButtonExecutionAdapter) subscribe(subject string, handler nats.MsgHandler) {
	a.natsAdapter.SubscribeToSubject(subject, handler)

	log.Printf("Successfully subscribed to NATS subject: %s", subject)
}

// --- NATS Message Handlers ---

// handlePieButtonExecuteMessage processes incoming pie button execution commands.
func (a *PieButtonExecutionAdapter) handlePieButtonExecuteMessage(msg *nats.Msg) {
	var message pieButtonExecute_Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		log.Printf("Failed to decode pieButtonExecute message: %v. Data: %s", err, string(msg.Data))
		return
	}

	if err := a.executeCommand(&message); err != nil {
		log.Printf("Failed to execute command for button %d (Type: %s): %v", message.ButtonIndex, message.TaskType, err)
		// Optionally, publish an error response back via NATS
	}
}

// handleShortcutPressedMessage stores the mouse coordinates when a shortcut is detected.
func (a *PieButtonExecutionAdapter) handleShortcutPressedMessage(msg *nats.Msg) {
	var message shortcutPressed_Message
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

// handleDiscoveredAppsMessage updates the internal list of discovered applications
func (a *PieButtonExecutionAdapter) handleDiscoveredAppsMessage(msg *nats.Msg) {
	var apps map[string]AppLaunchInfo
	if err := json.Unmarshal(msg.Data, &apps); err != nil {
		log.Printf("Failed to decode discovered apps message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	a.discoveredApps = apps
	a.mu.Unlock()

	log.Printf("Updated discovered apps list, %d apps tracked", len(apps))
}

// handleWindowUpdateMessage updates the internal list of active windows.
func (a *PieButtonExecutionAdapter) handleWindowUpdateMessage(msg *nats.Msg) {
	var currentWindows WindowsUpdate
	if err := json.Unmarshal(msg.Data, &currentWindows); err != nil {
		log.Printf("Failed to decode window update message: %v. Data: %s", err, string(msg.Data))
		return
	}

	a.mu.Lock()
	// Efficiently replace the map content. Avoids re-allocating if capacity allows.
	clear(a.windowsList) // Clear existing entries (Go 1.21+)
	maps.Copy(a.windowsList, currentWindows)
	// For older Go versions:
	// a.windowsList = make(WindowsUpdate, len(currentWindows))
	// maps.Copy(a.windowsList, currentWindows)
	a.mu.Unlock()

	// log.Printf("Updated windows list, %d windows tracked", len(currentWindows)) // Debug logging if needed
}

// --- Command Execution Logic ---

// executeCommand dispatches the command based on the TaskType.
func (a *PieButtonExecutionAdapter) executeCommand(executionInfo *pieButtonExecute_Message) error {
	log.Printf("Executing command for button %d: TaskType=%s", executionInfo.ButtonIndex, executionInfo.TaskType)

	switch executionInfo.TaskType {
	case TaskTypeShowProgramWindow:
		return a.handleShowProgramWindow(executionInfo)
	case TaskTypeShowAnyWindow:
		return a.handleShowAnyWindow(executionInfo)
	case TaskTypeCallFunction:
		return a.handleCallFunction(executionInfo)
	case TaskTypeLaunchProgram:
		log.Printf("Button %d - Launching program: %s", executionInfo.ButtonIndex, executionInfo.TaskType)
		return a.handleLaunchProgram(executionInfo)
	case TaskTypeDisabled:
		log.Printf("Button %d is disabled, doing nothing.", executionInfo.ButtonIndex)
		return nil // Nothing to do for disabled buttons
	default:
		return fmt.Errorf("unknown task type: %s", executionInfo.TaskType)
	}
}

// unmarshalProperties safely converts the generic properties map into a specific struct.
func unmarshalProperties(props any, target any) error {
	// 1. Type assert to the expected map type
	propsMap, _ := props.(map[string]any)

	// 2. Marshal the map back to JSON bytes
	propsBytes, err := json.Marshal(propsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal intermediate properties map: %v", err)
	}

	// 3. Unmarshal the JSON bytes into the target struct
	if err := json.Unmarshal(propsBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal properties into target type %T: %v", target, err)
	}

	return nil
}

// ----------------------------------------------------------------------
// --------- Handler Functions for the different Button Types -----------
// ----------------------------------------------------------------------

func (a *PieButtonExecutionAdapter) handleShowProgramWindow(executionInfo *pieButtonExecute_Message) error {
	var windowProps ShowWindowProperties
	if err := unmarshalProperties(executionInfo.Properties, &windowProps); err != nil {
		return fmt.Errorf("failed to process properties for show_program_window: %w", err)
	}

	log.Printf("Button %d - Action: ShowProgramWindow, Target: %s, ClickType: %s",
		executionInfo.ButtonIndex, windowProps.ButtonTextUpper, executionInfo.ClickType)

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("ShowProgramWindow (Left Click): Standard show for '%s'", windowProps.ButtonTextUpper)
		// TODO: Implement actual window showing logic using windowProps and potentially a.windowsList
		// This is where your original logic for handleShowProgramWindow would go.
		// For now, it's a placeholder as in your original code.
		log.Println("  (Executing placeholder for show_program_window left-click)")
	case ClickTypeRightUp:
		log.Printf("ShowProgramWindow (Right Click STUB) for '%s'", windowProps.ButtonTextUpper)
		// No operation for right-click yet
	case ClickTypeMiddleUp:
		log.Printf("ShowProgramWindow (Middle Click STUB) for '%s'", windowProps.ButtonTextUpper)
		// No operation for middle-click yet
	default:
		log.Printf("ShowProgramWindow: Unhandled ClickType '%s' for '%s'. Performing default (left-click like) action or nothing.",
			executionInfo.ClickType, windowProps.ButtonTextUpper)
		// Decide if unhandled types should default to left-click behavior or do nothing.
		// For now, let's treat as a stub or do the left-click action.
		log.Println("  (Executing placeholder for show_program_window default/unhandled click)")
	}
	return nil // Placeholder for actual operation
}

func (a *PieButtonExecutionAdapter) handleShowAnyWindow(executionInfo *pieButtonExecute_Message) error {
	var props ShowWindowProperties
	if err := unmarshalProperties(executionInfo.Properties, &props); err != nil {
		return fmt.Errorf("show_any_window: unmarshal failed: %w", err)
	}

	hwnd := uintptr(props.WindowHandle)
	if hwnd == 0 {
		return fmt.Errorf("show_any_window: HWND is zero (Button %d, Text: %s)", executionInfo.ButtonIndex, props.ButtonTextUpper)
	}

	log.Printf("Button %d - Action: ShowAnyWindow, Target HWND: %X, Text: %s, ClickType: %s",
		executionInfo.ButtonIndex, hwnd, props.ButtonTextUpper, executionInfo.ClickType)

	var err error
	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("ShowAnyWindow (Left Click): Standard foreground for HWND %X", hwnd)
		// This is your original logic from handleShowAnyWindow
		if e := setForegroundOrMinimize(hwnd); e != nil {
			log.Printf("show_any_window (Left Click): Failed to foreground HWND %X: %v", hwnd, e)
			err = fmt.Errorf("show_any_window (Left Click): %w", e)
		} else {
			log.Printf("show_any_window (Left Click): HWND %X requested to foreground (Button %d)", hwnd, executionInfo.ButtonIndex)
		}
	case ClickTypeRightUp:
		log.Printf("ShowAnyWindow (Right Click STUB) for HWND %X", hwnd)
		// No operation for right-click yet
	case ClickTypeMiddleUp:
		log.Printf("ShowAnyWindow (Middle Click STUB) for HWND %X", hwnd)
		// No operation for middle-click yet
	default:
		log.Printf("ShowAnyWindow: Unhandled ClickType '%s' for HWND %X. Performing default (left-click like) action.",
			executionInfo.ClickType, hwnd)
		// Defaulting to left-click behavior for unhandled types
		if e := setForegroundOrMinimize(hwnd); e != nil {
			err = fmt.Errorf("show_any_window (Default Click): %w", e)
		}
	}

	return err // err will be nil if successful or if it's a stubbed action
}

func (a *PieButtonExecutionAdapter) handleLaunchProgram(executionInfo *pieButtonExecute_Message) error {
	var launchProps LaunchProgramProperties
	if err := unmarshalProperties(executionInfo.Properties, &launchProps); err != nil {
		return fmt.Errorf("failed to process properties for launch_program: %w", err)
	}

	log.Printf("Button %d - Action: LaunchProgram, Target: %s (%s), ClickType: %s",
		executionInfo.ButtonIndex, launchProps.ButtonTextUpper, launchProps.ExePath, executionInfo.ClickType)

	var err error
	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("LaunchProgram (Left Click): Standard launch for '%s'", launchProps.ExePath)
		a.mu.RLock()
		err = LaunchApp(launchProps.ExePath, a.discoveredApps) // Original logic
		a.mu.RUnlock()
	case ClickTypeRightUp:
		log.Printf("LaunchProgram (Right Click STUB) for '%s'", launchProps.ExePath)
		// No operation for right-click yet
	case ClickTypeMiddleUp:
		log.Printf("LaunchProgram (Middle Click STUB) for '%s'", launchProps.ExePath)
		// No operation for middle-click yet
	default:
		log.Printf("LaunchProgram: Unhandled ClickType '%s' for '%s'. Performing default (left-click like) action.",
			executionInfo.ClickType, launchProps.ExePath)
		// Defaulting to left-click behavior
		a.mu.RLock()
		err = LaunchApp(launchProps.ExePath, a.discoveredApps)
		a.mu.RUnlock()
	}

	if err != nil && executionInfo.ClickType == ClickTypeLeftUp { // Only log launch failure for actual attempts
		return fmt.Errorf("launch_program (Left Click) for '%s' failed: %w", launchProps.ExePath, err)
	}
	return err // err will be nil for successful left-click or for stubbed actions
}

func (a *PieButtonExecutionAdapter) handleCallFunction(executionInfo *pieButtonExecute_Message) error {
	var functionProps CallFunctionProperties
	if err := unmarshalProperties(executionInfo.Properties, &functionProps); err != nil {
		return fmt.Errorf("failed to process properties for call_function: %w", err)
	}

	functionName := functionProps.ButtonTextUpper

	log.Printf("Button %d - Action: CallFunction, TargetFn: %s, ClickType: %s",
		executionInfo.ButtonIndex, functionName, executionInfo.ClickType)

	// Get mouse coordinates regardless of click type, as they might be logged or used by left-click
	a.mu.RLock()
	mouseX := a.lastMouseX
	mouseY := a.lastMouseY
	a.mu.RUnlock()

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("CallFunction (Left Click): Proceeding to execute function '%s'", functionName)
		handler, exists := a.functionHandlers[functionName]
		if !exists {
			return fmt.Errorf("unknown function requested for left-click: %s", functionName)
		}
		// Calls the original handler.Execute(mouseX, mouseY)
		// NO change to HandlerWrapper, Basic/CoordHandler, or a.MaximizeWindow signatures needed for THIS approach.
		err := handler.Execute(mouseX, mouseY)
		if err != nil {
			return fmt.Errorf("call_function '%s' (Left Click) failed: %w", functionName, err)
		}
		return nil // Successfully executed left-click action

	case ClickTypeRightUp:
		log.Printf("CallFunction (Right Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			functionName, mouseX, mouseY)
		return nil // Stub action for right-click

	case ClickTypeMiddleUp:
		log.Printf("CallFunction (Middle Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			functionName, mouseX, mouseY)
		return nil // Stub action for middle-click

	default:
		log.Printf("CallFunction: Unhandled ClickType '%s' for function '%s'. No action taken.",
			executionInfo.ClickType, functionName)

		return nil // For now, unhandled also does nothing specific.
	}
}

// -------------------------
// --------- Run -----------
// -------------------------

// Run starts the adapter's main loop (currently just blocks).
func (a *PieButtonExecutionAdapter) Run() error {
	log.Println("PieButtonExecutionAdapter started and listening for events.")
	// Blocks indefinitely, keeping the adapter alive to process NATS messages.
	select {}
}

// --- Function Handlers & Wrappers ---

// FunctionHandler defines the signature for basic functions without coordinates.
type FunctionHandler func() error

// FunctionHandlerWithCoords defines the signature for functions needing coordinates.
type FunctionHandlerWithCoords func(x, y int) error

// HandlerWrapper provides a common interface for executing different function types.
type HandlerWrapper interface {
	Execute(x, y int) error
}

// BasicHandler wraps a FunctionHandler to satisfy the HandlerWrapper interface.
type BasicHandler struct {
	fn FunctionHandler
}

// Execute calls the wrapped basic function, ignoring coordinates.
func (h BasicHandler) Execute(x, y int) error {
	return h.fn()
}

// CoordHandler wraps a FunctionHandlerWithCoords to satisfy the HandlerWrapper interface.
type CoordHandler struct {
	fn FunctionHandlerWithCoords
}

// Execute calls the wrapped function, passing coordinates.
func (h CoordHandler) Execute(x, y int) error {
	return h.fn(x, y)
}

// initFunctionHandlers registers known functions that can be called.
func (a *PieButtonExecutionAdapter) initFunctionHandlers() map[string]HandlerWrapper {
	handlers := make(map[string]HandlerWrapper)

	// Register handlers that need coordinates
	handlers[fnMaximize] = CoordHandler{fn: a.MaximizeWindow} // Use method value
	handlers[fnMinimize] = CoordHandler{fn: a.MinimizeWindow} // Use method value

	// Register basic handlers
	handlers[fnClose] = BasicHandler{fn: CloseWindow} // Use method value (or static func if no 'a' needed)
	handlers[fnTopmost] = BasicHandler{fn: ToggleTopmost}

	log.Printf("Initialized %d function handlers", len(handlers))
	return handlers
}

// -----------------------------------------
// --- Concrete Function Implementations ---
// -----------------------------------------

// LaunchApp launches an application using its path.
// Returns error if the app cannot be launched.
func LaunchApp(exePath string, apps map[string]AppLaunchInfo) error {
	app, exists := apps[exePath]
	if !exists {
		return fmt.Errorf("application not found: %s", exePath)
	}

	// If URI is specified, use it instead of exe path
	if app.URI != "" {
		cmd := exec.Command("cmd", "/C", "start", app.URI)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start %s via URI: %w", app.Name, err)
		}
		log.Printf("Started %s via URI handler", app.Name)
		return nil
	}

	// Prepare command
	cmd := exec.Command(exePath)

	// Set working directory if specified
	if app.WorkingDirectory != "" {
		// If working directory is relative, make it relative to exe path
		if !filepath.IsAbs(app.WorkingDirectory) {
			cmd.Dir = filepath.Join(filepath.Dir(exePath), app.WorkingDirectory)
		} else {
			cmd.Dir = app.WorkingDirectory
		}
	} else {
		// Default to exe's directory
		cmd.Dir = filepath.Dir(exePath)
	}

	// Add arguments if specified
	if app.Args != "" {
		// Split args respecting quoted strings
		args := strings.Fields(app.Args)
		cmd.Args = append([]string{exePath}, args...)
	}

	// Start the application
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", app.Name, err)
	}

	fmt.Printf("Started application: %s (Path: %s)", app.Name, exePath)
	return nil
}

// MaximizeWindow - Original method implementation
func (a *PieButtonExecutionAdapter) MaximizeWindow(x, y int) error {
	// NOTE: Relies on a.GetWindowAtPoint and an assumed Maximize method
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err // Original direct error return
	}
	// This line assumes 'hwnd' has a Maximize() error method.
	// The actual type and method depend on your OS library.
	// It will fail at runtime if GetWindowAtPoint returns something
	// without that method.
	return hwnd.Maximize() // Original direct call
}

// MinimizeWindow - Original method implementation
func (a *PieButtonExecutionAdapter) MinimizeWindow(x, y int) error {
	// NOTE: Relies on a.GetWindowAtPoint and an assumed Minimize method
	hwnd, err := a.GetWindowAtPoint(x, y)
	if err != nil {
		return err // Original direct error return
	}
	// This line assumes 'hwnd' has a Minimize() error method.
	// The actual type and method depend on your OS library.
	// It will fail at runtime if GetWindowAtPoint returns something
	// without that method.
	return hwnd.Minimize() // Original direct call
}

// CloseWindow - Original standalone function implementation
func CloseWindow() error {
	fmt.Println("Closing window") // Original Println
	return nil                    // Original return nil
}

// ToggleTopmost - Original standalone function implementation
func ToggleTopmost() error {
	fmt.Println("Toggling window topmost state") // Original Println
	return nil                                   // Original return nil
}

func (a *PieButtonExecutionAdapter) ToggleTopmost() error {
	log.Println("Attempting to toggle topmost state for the foreground window")
	// TODO: Replace with actual implementation
	return fmt.Errorf("ToggleTopmost not implemented")
}
