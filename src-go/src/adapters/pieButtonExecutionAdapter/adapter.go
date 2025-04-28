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

type pieButtonExecute_Message struct {
    MenuIndex   int         `json:"menu_index"`
    ButtonIndex int         `json:"button_index"`
    TaskType    TaskType    `json:"task_type"`
    Properties  interface{} `json:"properties"`
    ClickType   string      `json:"click_type"`
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

    return a
}

// TaskType represents the available task types
type TaskType string

const (
    TaskTypeShowProgramWindow TaskType = "show_program_window"
    TaskTypeShowAnyWindow    TaskType = "show_any_window"
    TaskTypeCallFunction     TaskType = "call_function"
    TaskTypeLaunchProgram    TaskType = "launch_program"
    TaskTypeDisabled         TaskType = "disabled"
)

// ShowWindowProperties contains common properties for window-related tasks
type ShowWindowProperties struct {
    ButtonTextUpper string `json:"button_text_upper"` // window title
    ButtonTextLower string `json:"button_text_lower"` // app name
    IconPath        string `json:"icon_path"`
    WindowHandle    int64  `json:"window_handle"`
    ExePath         string `json:"exe_path"`
}

// LaunchProgramProperties contains properties for launching programs
type LaunchProgramProperties struct {
    ButtonTextUpper string `json:"button_text_upper"` // app name
    ButtonTextLower string `json:"button_text_lower"` // " - Launch - "
    IconPath        string `json:"icon_path"`
    ExePath         string `json:"exe_path"`
}

// CallFunctionProperties contains properties for function calls
type CallFunctionProperties struct {
    ButtonTextUpper string `json:"button_text_upper"` // function name
    ButtonTextLower string `json:"button_text_lower"` // empty string
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

    fmt.Printf("Showing program window: %+v\n", windowProps)
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

    fmt.Printf("Showing any window: %+v\n", windowProps)
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

    fmt.Printf("Calling function: %+v\n", functionProps.ButtonTextUpper)
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

    fmt.Printf("Launching program: %+v\n", launchProps)
    return nil
}

func (a *PieButtonExecutionAdapter) Run() error {
    fmt.Println("PieButtonExecutor started")
    select {} 
}