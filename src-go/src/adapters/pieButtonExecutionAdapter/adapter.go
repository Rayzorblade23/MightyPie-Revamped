package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

type PieButtonExecutionAdapter struct {
    natsAdapter *natsAdapter.NatsAdapter
}

func New(natsAdapter *natsAdapter.NatsAdapter) *PieButtonExecutionAdapter {
    a := &PieButtonExecutionAdapter{
        natsAdapter: natsAdapter,
    }

    // Subscribe to pie button execution events
    natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_PIEBUTTON_EXECUTE"), func(msg *nats.Msg) {
        var message pieButtonExecute_Message
        if err := json.Unmarshal(msg.Data, &message); err != nil {
            fmt.Printf("Failed to decode command: %v\n", err)
            return
        }

        if err := a.executeCommand(&message); err != nil {
            fmt.Printf("Failed to execute command: %v\n", err)
        }
    })

    natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_SHORTCUT_PRESSED"), func(msg *nats.Msg) {
        var message shortcutPressed_Message
        if err := json.Unmarshal(msg.Data, &message); err != nil {
            fmt.Printf("Failed to decode command: %v\n", err)
            return
        }

		fmt.Printf("PieButtonExecutionAdapter knows a Shortcut is pressed: %d\n", message.ShortcutPressed)
		fmt.Printf("PieButtonExecutionAdapter also knows where: X: %d, Y: %d\n", message.MouseX, message.MouseY)

    })

    return a
}

func (a *PieButtonExecutionAdapter) executeCommand(executionInfo *pieButtonExecute_Message) error {
    switch executionInfo.TaskType {
    case TaskTypeShowProgramWindow:
        return a.handleShowProgramWindow(executionInfo)
    case TaskTypeShowAnyWindow:
        return a.handleShowAnyWindow(executionInfo)
    case TaskTypeCallFunction:
        return a.handleCallFunction(executionInfo)
    case TaskTypeLaunchProgram:
        return a.handleLaunchProgram(executionInfo)
    case TaskTypeDisabled:
        return nil // Nothing to do for disabled buttons
    default:
        return fmt.Errorf("unknown task type: %s", executionInfo.TaskType)
    }
}

func (a *PieButtonExecutionAdapter) handleShowProgramWindow(executionInfo *pieButtonExecute_Message) error {
    props, ok := executionInfo.Properties.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid properties type for show_program_window")
    }

    // Convert the generic properties to ShowWindowProperties
    propsBytes, err := json.Marshal(props)
    if err != nil {
        return fmt.Errorf("failed to marshal properties: %v", err)
    }

    var windowProps ShowWindowProperties
    if err := json.Unmarshal(propsBytes, &windowProps); err != nil {
        return fmt.Errorf("failed to unmarshal ShowWindowProperties: %v", err)
    }

    fmt.Printf("Button %+v - Showing program window: %+v\n", executionInfo.ButtonIndex, windowProps.ButtonTextUpper)
    return nil
}

func (a *PieButtonExecutionAdapter) handleShowAnyWindow(executionInfo *pieButtonExecute_Message) error {
    // Similar to handleShowProgramWindow
    props, ok := executionInfo.Properties.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid properties type for show_any_window")
    }

    var windowProps ShowWindowProperties
    propsBytes, _ := json.Marshal(props)
    if err := json.Unmarshal(propsBytes, &windowProps); err != nil {
        return fmt.Errorf("failed to unmarshal ShowWindowProperties: %v", err)
    }

    fmt.Printf("Button %+v - Showing any window: %+v\n", executionInfo.ButtonIndex, windowProps.ButtonTextUpper)
    return nil
}

func (a *PieButtonExecutionAdapter) handleLaunchProgram(executionInfo *pieButtonExecute_Message) error {
    props, ok := executionInfo.Properties.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid properties type for launch_program")
    }

    var launchProps LaunchProgramProperties
    propsBytes, _ := json.Marshal(props)
    if err := json.Unmarshal(propsBytes, &launchProps); err != nil {
        return fmt.Errorf("failed to unmarshal LaunchProgramProperties: %v", err)
    }

    fmt.Printf("Button %+v - Launching program: %+v\n", executionInfo.ButtonIndex, launchProps.ButtonTextUpper)
    return nil
}

func (a *PieButtonExecutionAdapter) handleCallFunction(executionInfo *pieButtonExecute_Message) error {
    props, ok := executionInfo.Properties.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid properties type for call_function")
    }

    var functionProps CallFunctionProperties
    propsBytes, _ := json.Marshal(props)
    if err := json.Unmarshal(propsBytes, &functionProps); err != nil {
        return fmt.Errorf("failed to unmarshal CallFunctionProperties: %v", err)
    }

    // Look up the function handler
    handler, exists := functionHandlers[functionProps.ButtonTextUpper]
    if !exists {
        return fmt.Errorf("unknown function: %s", functionProps.ButtonTextUpper)
    }

    // Execute the function
    return handler()
}

func (a *PieButtonExecutionAdapter) Run() error {
    fmt.Println("PieButtonExecutor started")
    select {} 
}

// Add these types near the top of the file
type FunctionHandler func() error

var functionHandlers = map[string]FunctionHandler{
    "Maximize Window": MaximizeWindow,
    "Minimize Window": MinimizeWindow,
    // Add more handlers as needed
}

// Add these functions after the existing code
func MaximizeWindow() error {
    fmt.Println("Maximizing window")
    return nil
}

func MinimizeWindow() error {
    fmt.Println("Minimizing window")
    return nil
}

func CloseWindow() error {
    fmt.Println("Closing window")
    return nil
}

func ToggleTopmost() error {
    fmt.Println("Toggling window topmost state")
    return nil
}