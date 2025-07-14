package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

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
			// Save HWND if this is an Explorer window
			if windowProps.ButtonTextLower == "Windows Explorer" {
				a.mu.Lock()
				a.lastExplorerWindowHWND = WindowHandle(hwnd)
				a.mu.Unlock()
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
		log.Printf("ShowProgramWindow (Middle Click): Attempting to close window for app '%s'", appNameKey)
		if windowProps.WindowHandle > 0 {
			hwnd := uintptr(windowProps.WindowHandle)
			if err := WindowHandle(hwnd).Close(); err != nil {
				log.Printf("ShowProgramWindow (Middle Click): Failed to close HWND %X: %v", hwnd, err)
				return fmt.Errorf("show_program_window (Middle Click): %w", err)
			} else {
				log.Printf("ShowProgramWindow (Middle Click): HWND %X requested to close (Button %d)", hwnd, executionInfo.ButtonIndex)
			}
		} else {
			log.Printf("ShowProgramWindow (Middle Click): No window handle available to close for app '%s'", appNameKey)
		}
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
		// This is your original logic from handleShowAnyWindow
		if e := setForegroundOrMinimize(hwnd); e != nil {
			log.Printf("show_any_window (Left Click): Failed to foreground HWND %X: %v", hwnd, e)
			err = fmt.Errorf("show_any_window (Left Click): %w", e)
		} else {
			// Save HWND if this is an Explorer window
			if props.ButtonTextLower == "Windows Explorer" {
				a.mu.Lock()
				a.lastExplorerWindowHWND = WindowHandle(hwnd)
				a.mu.Unlock()
			}
		}
	case ClickTypeRightUp:
		log.Printf("ShowAnyWindow (Right Click STUB) for HWND %X", hwnd)
		// No operation for right-click yet
	case ClickTypeMiddleUp:
		log.Printf("ShowAnyWindow (Middle Click): Closing window HWND %X", hwnd)
		if hwnd != 0 {
			if err := WindowHandle(hwnd).Close(); err != nil {
				log.Printf("ShowAnyWindow (Middle Click): Failed to close HWND %X: %v", hwnd, err)
			} else {
				log.Printf("ShowAnyWindow (Middle Click): HWND %X requested to close (Button %d)", hwnd, executionInfo.ButtonIndex)
			}
		}
		// No further action for middle click
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

	log.Printf("Button %d - Action: LaunchProgram, Target AppName: %s, ClickType: %s",
		executionInfo.ButtonIndex, appNameKey, executionInfo.ClickType)

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
		log.Printf("LaunchProgram (Middle Click): No window handle available to close for '%s'", appNameKey)
		// No further action for middle click
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

	displayName := functionProps.ButtonTextUpper

	// Use displayName directly as the handler key
	log.Printf("Button %d - Action: CallFunction, TargetFn: %s, ClickType: %s",
		executionInfo.ButtonIndex, displayName, executionInfo.ClickType)

	// Get mouse coordinates regardless of click type, as they might be logged or used by left-click
	a.mu.RLock()
	mouseX := a.lastMouseX
	mouseY := a.lastMouseY
	a.mu.RUnlock()

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Printf("CallFunction (Left Click): Proceeding to execute function '%s'", displayName)
		handler, exists := a.functionHandlers[displayName]
		if !exists {
			return fmt.Errorf("unknown function requested for left-click: %s", displayName)
		}
		err := handler.Execute(mouseX, mouseY)
		if err != nil {
			return fmt.Errorf("call_function '%s' (Left Click) failed: %w", displayName, err)
		}
		return nil

	case ClickTypeRightUp:
		log.Printf("CallFunction (Right Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			displayName, mouseX, mouseY)
		return nil

	case ClickTypeMiddleUp:
		log.Printf("CallFunction (Middle Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			displayName, mouseX, mouseY)
		return nil

	default:
		log.Printf("CallFunction: Unhandled ClickType '%s' for function '%s'. No action taken.",
			executionInfo.ClickType, displayName)
		return nil
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
