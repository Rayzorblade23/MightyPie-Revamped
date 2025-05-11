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

// WindowsUpdate stores information about currently open windows, keyed by HWND or other ID.
type WindowsUpdate map[int]WindowInfo // Assuming int is the window handle/ID type

// WindowInfo holds details about a specific window.
type WindowInfo struct {
	Title    string
	ExeName  string
	ExePath  string
	AppName  string
	Instance int
	IconPath string
}

// AppLaunchInfo defines the structure of the VALUE in discoveredApps.
type AppLaunchInfo struct {
	Name             string `json:"name"`                       // The original display name
	WorkingDirectory string `json:"workingDirectory,omitempty"` // Working directory from LNK
	Args             string `json:"args,omitempty"`             // Command line args from LNK
	URI              string `json:"uri,omitempty"`              // Add this field for store apps
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
func unmarshalProperties(props interface{}, target interface{}) error {
	// 1. Type assert to the expected map type
	propsMap, ok := props.(map[string]interface{})
	if !ok {
		// If it's already the target type due to direct unmarshalling, skip remarshal
		// This requires careful message structure design on the publisher side.
		// For robustness with interface{}, remarshalling is safer.
		// return fmt.Errorf("invalid properties format: expected map[string]interface{}, got %T", props)

		// Alternative: Attempt direct type assertion if the JSON unmarshaller might have already produced the correct type.
		// if target != nil {
		//    v := reflect.ValueOf(target)
		//    if v.Kind() == reflect.Ptr {
		// 		if reflect.TypeOf(props) == v.Elem().Type() {
		// 			v.Elem().Set(reflect.ValueOf(props))
		// 			return nil // Already correct type
		//		}
		//    }
		// }
		// Falling back to remarshalling for broad compatibility.
	}

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

func (a *PieButtonExecutionAdapter) handleShowProgramWindow(executionInfo *pieButtonExecute_Message) error {
	var windowProps ShowWindowProperties
	if err := unmarshalProperties(executionInfo.Properties, &windowProps); err != nil {
		return fmt.Errorf("failed to process properties for show_program_window: %w", err)
	}

	log.Printf("Button %d - Showing program window: %s", executionInfo.ButtonIndex, windowProps.ButtonTextUpper)
	// TODO: Implement actual window showing logic using windowProps and potentially a.windowsList
	return nil // Placeholder
}

func (a *PieButtonExecutionAdapter) handleShowAnyWindow(msg *pieButtonExecute_Message) error {
	var props ShowWindowProperties
	if err := unmarshalProperties(msg.Properties, &props); err != nil {
		return fmt.Errorf("show_any_window: unmarshal failed: %w", err)
	}

	hwnd := uintptr(props.WindowHandle)
	if hwnd == 0 {
		return fmt.Errorf("show_any_window: HWND is zero (Button %d, Text: %s)", msg.ButtonIndex, props.ButtonTextUpper)
	}

	logWindowContext(msg.ButtonIndex, props.ButtonTextUpper, hwnd)

	if err := setForegroundOrMinimize(hwnd); err != nil {
		log.Printf("show_any_window: Failed to foreground HWND %X: %v", hwnd, err)
		return fmt.Errorf("show_any_window: %w", err)
	}
	log.Printf("show_any_window: HWND %X requested to foreground (Button %d)", hwnd, msg.ButtonIndex)
	return nil
}

func (a *PieButtonExecutionAdapter) handleLaunchProgram(executionInfo *pieButtonExecute_Message) error {
	var launchProps LaunchProgramProperties
	if err := unmarshalProperties(executionInfo.Properties, &launchProps); err != nil {
		return fmt.Errorf("failed to process properties for launch_program: %w", err)
	}

	log.Printf("Button %d - Launching program: %s", executionInfo.ButtonIndex, launchProps.ButtonTextUpper)

	a.mu.RLock()
	err := LaunchApp(launchProps.ExePath, a.discoveredApps)
	a.mu.RUnlock()

	return err
}

func (a *PieButtonExecutionAdapter) handleCallFunction(executionInfo *pieButtonExecute_Message) error {
	var functionProps CallFunctionProperties
	if err := unmarshalProperties(executionInfo.Properties, &functionProps); err != nil {
		return fmt.Errorf("failed to process properties for call_function: %w", err)
	}

	functionName := functionProps.ButtonTextUpper // Assuming this holds the function name
	handler, exists := a.functionHandlers[functionName]
	if !exists {
		return fmt.Errorf("unknown function requested: %s", functionName)
	}

	log.Printf("Button %d - Calling function: %s", executionInfo.ButtonIndex, functionName)

	// Acquire Read Lock before accessing coordinates
	a.mu.RLock()
	mouseX := a.lastMouseX
	mouseY := a.lastMouseY
	a.mu.RUnlock() // Release Read Lock

	return handler.Execute(mouseX, mouseY)
}

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

// --- Concrete Function Implementations ---

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
