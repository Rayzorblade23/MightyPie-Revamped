// Package pieButtonExecutionAdapter handles the logic triggered by pie menu button actions.
// It listens to NATS messages for button executions, shortcut presses (to get context like mouse position),
// and window updates, dispatching actions accordingly.
package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"log" // Use the standard log package
	"maps"
	"sync"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd" // Assuming this provides environment variables
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/nats-io/nats.go"
)

// NATS Subjects - fetched from environment, consider constants if these are static.
var (
	natsSubjectPieButtonExecute    = env.Get("PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE")
	natsSubjectShortcutPressed     = env.Get("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	natsSubjectWindowManagerUpdate = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_UPDATE")
	natsSubjectInstalledAppsInfo   = env.Get("PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO")
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
	natsAdapter       *natsAdapter.NatsAdapter
	lastMouseX        int
	lastMouseY        int
	mu                sync.RWMutex // Protects access to windowsList
	windowsList       core.WindowsUpdate
	installedAppsInfo map[string]core.AppInfo
	functionHandlers  map[string]HandlerWrapper
}

// --- Adapter Implementation ---

// New creates and initializes a new PieButtonExecutionAdapter.
func New(natsAdapter *natsAdapter.NatsAdapter) *PieButtonExecutionAdapter {
	a := &PieButtonExecutionAdapter{
		natsAdapter:       natsAdapter,
		windowsList:       make(core.WindowsUpdate),
		installedAppsInfo: make(map[string]core.AppInfo),
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
	a.subscribe(natsSubjectInstalledAppsInfo, a.handleInstalledAppsInfoMessage)
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
	// Efficiently replace the map content. Avoids re-allocating if capacity allows.
	clear(a.windowsList) // Clear existing entries (Go 1.21+)
	maps.Copy(a.windowsList, currentWindows)
	// For older Go versions:
	// a.windowsList = make(core.WindowsUpdate, len(currentWindows))
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
	var windowProps core.ShowProgramWindowProperties
	if err := unmarshalProperties(executionInfo.Properties, &windowProps); err != nil {
		return fmt.Errorf("failed to process properties for show_program_window: %w", err)
	}

	appNameKey := windowProps.ButtonTextLower

	log.Printf("Button %d - Action: ShowProgramWindow, Target AppName: %s (Window Title: %s), ClickType: %s",
		executionInfo.ButtonIndex, appNameKey, windowProps.ButtonTextUpper, executionInfo.ClickType)

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		if windowProps.WindowHandle > 0 {
			hwnd := uintptr(windowProps.WindowHandle)
			if err := setForegroundOrMinimize(hwnd); err != nil {
				return fmt.Errorf("show_program_window: failed to focus window: %w", err)
			}
			log.Printf("ShowProgramWindow: Focused existing window for '%s' (Title: %s, HWND: %X)",
				appNameKey, windowProps.ButtonTextUpper, hwnd)
			return nil
		}

		log.Printf("ShowProgramWindow: No existing window found for '%s'. Attempting to launch.", appNameKey)
		a.mu.RLock()
		// Assuming appNameKey (from ButtonTextLower) will always be in installedAppsInfo
		appInfoToLaunch := a.installedAppsInfo[appNameKey]
		a.mu.RUnlock()

		if err := LaunchApp(appNameKey, appInfoToLaunch); err != nil {
			return fmt.Errorf("show_program_window: failed to launch program '%s': %w", appNameKey, err)
		}
		return nil

	case ClickTypeRightUp:
		log.Printf("ShowProgramWindow (Right Click STUB) for app '%s'", appNameKey)
		return nil
	case ClickTypeMiddleUp:
		log.Printf("ShowProgramWindow (Middle Click STUB) for app '%s'", appNameKey)
		return nil
	default:
		log.Printf("ShowProgramWindow: Unhandled click type '%s' for app '%s'. No action taken.",
			executionInfo.ClickType, appNameKey)
		return nil
	}
}

func (a *PieButtonExecutionAdapter) handleShowAnyWindow(executionInfo *pieButtonExecute_Message) error {
	var props core.ShowAnyWindowProperties
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
	var launchProps core.LaunchProgramProperties
	if err := unmarshalProperties(executionInfo.Properties, &launchProps); err != nil {
		return fmt.Errorf("failed to process properties for launch_program: %w", err)
	}

	appNameKey := launchProps.ButtonTextUpper

	log.Printf("Button %d - Action: LaunchProgram, Target AppName: %s (Configured ExePath: %s), ClickType: %s",
		executionInfo.ButtonIndex, appNameKey, launchProps.ExePath, executionInfo.ClickType)

	var err error
	a.mu.RLock()

	appInfoToLaunch := a.installedAppsInfo[appNameKey]

	a.mu.RUnlock()

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("LaunchProgram (Left Click): Standard launch for '%s'", appNameKey)
		err = LaunchApp(appNameKey, appInfoToLaunch)
	case ClickTypeRightUp:
		log.Printf("LaunchProgram (Right Click STUB) for '%s'", appNameKey)
	case ClickTypeMiddleUp:
		log.Printf("LaunchProgram (Middle Click STUB) for '%s'", appNameKey)
	default:
		log.Printf("LaunchProgram: Unhandled ClickType '%s' for '%s'. Performing default (left-click like) action.",
			executionInfo.ClickType, appNameKey)
		err = LaunchApp(appNameKey, appInfoToLaunch)
	}

	if err != nil && (executionInfo.ClickType == ClickTypeLeftUp || executionInfo.ClickType == "") {
		return fmt.Errorf("launch_program action for '%s' failed: %w", appNameKey, err)
	}
	return err
}

func (a *PieButtonExecutionAdapter) handleCallFunction(executionInfo *pieButtonExecute_Message) error {
	var functionProps core.CallFunctionProperties
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

// LaunchApp launches an application using its unique application name.
func LaunchApp(appNameKey string, appInfo core.AppInfo) error {

	if appInfo.URI != "" {
		return launchViaURI(appNameKey, appInfo.URI)
	}

	// appInfo.ExePath could still be empty if the app is misconfigured and not URI-based.
	if appInfo.ExePath == "" {
		return fmt.Errorf("no executable path or URI for application '%s'", appNameKey)
	}

	cmd, err := buildExecCmd(appInfo.ExePath, appInfo.WorkingDirectory, appInfo.Args)
	if err != nil {
		// This error from _prepareExecutableCommand would typically be because appInfo.ExePath was empty,
		// but we have a check above for that already. Still, good to propagate.
		return fmt.Errorf("cannot launch '%s', failed to prepare command: %w", appNameKey, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start executable '%s' for app '%s': %w", appInfo.ExePath, appNameKey, err)
	}

	log.Printf("Successfully started application: '%s' (Path: %s, PID: %d)", appNameKey, appInfo.ExePath, cmd.Process.Pid)
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
