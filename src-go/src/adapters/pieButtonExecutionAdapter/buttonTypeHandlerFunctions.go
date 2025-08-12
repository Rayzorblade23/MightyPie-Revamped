package pieButtonExecutionAdapter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

func (a *PieButtonExecutionAdapter) handleShowProgramWindow(executionInfo *pieButtonExecute_Message) error {
	var windowProps core.ShowProgramWindowProperties
	if err := unmarshalProperties(executionInfo.Properties, &windowProps); err != nil {
		return fmt.Errorf("failed to process properties for show_program_window: %w", err)
	}

	appNameKey := windowProps.ButtonTextLower

	log.Info("Button %d - Action: ShowProgramWindow - ClickType: %s", executionInfo.ButtonIndex, executionInfo.ClickType)
	log.Info("↳ Target AppName: %s", appNameKey)
	log.Info("↳ Window Title: %s", windowProps.ButtonTextUpper)

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		if windowProps.WindowHandle > 0 {
			hwnd := uintptr(windowProps.WindowHandle)
			if err := a.setForegroundOrMinimize(hwnd); err != nil {
				return fmt.Errorf("show_program_window: failed to focus window: %w", err)
			}
			// Save HWND if this is an Explorer window
			if windowProps.ButtonTextLower == "Windows Explorer" {
				a.mu.Lock()
				a.lastExplorerWindowHWND = WindowHandle(hwnd)
				a.mu.Unlock()
			}
			log.Info("Focused existing window for '%s' (Title: %s, HWND: %X)",
				appNameKey, windowProps.ButtonTextUpper, hwnd)
			return nil
		}

		log.Info("ShowProgramWindow: No existing window found for '%s'. Attempting to launch.", appNameKey)
		a.mu.RLock()
		// Assuming appNameKey (from ButtonTextLower) will always be in installedAppsInfo
		appInfoToLaunch := a.installedAppsInfo[appNameKey]
		a.mu.RUnlock()

		if err := LaunchApp(appNameKey, appInfoToLaunch); err != nil {
			return fmt.Errorf("show_program_window: failed to launch program '%s': %w", appNameKey, err)
		}
		return nil

	case ClickTypeRightUp:
		log.Info("ShowProgramWindow (Right Click STUB) for app '%s'", appNameKey)
		return nil
	case ClickTypeMiddleUp:
		if windowProps.WindowHandle > 0 {
			hwnd := uintptr(windowProps.WindowHandle)
			if err := WindowHandle(hwnd).Close(); err != nil {
				log.Error("ShowProgramWindow (Middle Click): Failed to close HWND %X: %v", hwnd, err)
				return fmt.Errorf("show_program_window (Middle Click): %w", err)
			} else {
				log.Info("ShowProgramWindow (Middle Click): HWND %X requested to close (Button %d)", hwnd, executionInfo.ButtonIndex)
			}
		} else {
			log.Info("No window handle available to close for app '%s'", appNameKey)
		}
		return nil
	default:
		log.Info("ShowProgramWindow: Unhandled click type '%s' for app '%s'. No action taken.",
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
	// Check for both zero and invalid (-1) window handles
	if hwnd == 0 || hwnd == ^uintptr(0) { // ^uintptr(0) is FFFFFFFFFFFFFFFF
		log.Warn("Button %d not assigned.", executionInfo.ButtonIndex)
		return nil
	}

	log.Info("Button %d - Action: ShowAnyWindow - ClickType: %s", executionInfo.ButtonIndex, executionInfo.ClickType)
	log.Info("↳ Target HWND: %X", hwnd)
	log.Info("↳ Text: %s", props.ButtonTextUpper)

	var err error
	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		// This is your original logic from handleShowAnyWindow
		if e := a.setForegroundOrMinimize(hwnd); e != nil {
			log.Error("show_any_window (Left Click): Failed to foreground HWND %X: %v", hwnd, e)
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
		log.Info("ShowAnyWindow (Right Click STUB) for HWND %X", hwnd)
		// No operation for right-click yet
	case ClickTypeMiddleUp:
		log.Info("ShowAnyWindow (Middle Click): Closing window HWND %X", hwnd)
		if hwnd != 0 {
			if err := WindowHandle(hwnd).Close(); err != nil {
				log.Error("ShowAnyWindow (Middle Click): Failed to close HWND %X: %v", hwnd, err)
			} else {
				log.Info("ShowAnyWindow (Middle Click): HWND %X requested to close (Button %d)", hwnd, executionInfo.ButtonIndex)
			}
		}
		// No further action for middle click
	default:
		log.Info("ShowAnyWindow: Unhandled ClickType '%s' for HWND %X. Performing default (left-click like) action.",
			executionInfo.ClickType, hwnd)
		// Defaulting to left-click behavior for unhandled types
		if e := a.setForegroundOrMinimize(hwnd); e != nil {
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

	log.Info("Button %d - Action: LaunchProgram - ClickType: %s", executionInfo.ButtonIndex, executionInfo.ClickType)
	log.Info("↳ Target AppName: %s", appNameKey)

	var err error
	a.mu.RLock()

	appInfoToLaunch := a.installedAppsInfo[appNameKey]

	a.mu.RUnlock()

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Info("LaunchProgram (Left Click): Standard launch for '%s'", appNameKey)
		err = LaunchApp(appNameKey, appInfoToLaunch)
	case ClickTypeRightUp:
		log.Info("LaunchProgram (Right Click STUB) for '%s'", appNameKey)
	case ClickTypeMiddleUp:
		log.Info("LaunchProgram (Middle Click): No window handle available to close for '%s'", appNameKey)
		// No further action for middle click
	default:
		log.Info("LaunchProgram: Unhandled ClickType '%s' for '%s'. Performing default (left-click like) action.",
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
	log.Info("Button %d - Action: CallFunction - ClickType: %s", executionInfo.ButtonIndex, executionInfo.ClickType)
	log.Info("↳ TargetFn: %s", displayName)

	// Get mouse coordinates regardless of click type, as they might be logged or used by left-click
	a.mu.RLock()
	mouseX := a.lastMouseX
	mouseY := a.lastMouseY
	a.mu.RUnlock()

	switch executionInfo.ClickType {
	case ClickTypeLeftUp:
		log.Info("CallFunction (Left Click): Proceeding to execute function '%s'", displayName)
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
		log.Info("CallFunction (Right Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			displayName, mouseX, mouseY)
		return nil

	case ClickTypeMiddleUp:
		log.Info("CallFunction (Middle Click STUB) for function '%s' at X:%d, Y:%d. No action taken.",
			displayName, mouseX, mouseY)
		return nil

	default:
		log.Info("CallFunction: Unhandled ClickType '%s' for function '%s'. No action taken.",
			executionInfo.ClickType, displayName)
		return nil
	}
}

func (a *PieButtonExecutionAdapter) handleOpenPageInMenu(executionInfo *pieButtonExecute_Message) error {
	var props core.OpenSpecificPieMenuPage
	if err := unmarshalProperties(executionInfo.Properties, &props); err != nil {
		return fmt.Errorf("failed to process properties for open_page_in_menu: %w", err)
	}

	log.Info("Button %d - Action: OpenPageInMenu, Target MenuID: %d, PageID: %d, ClickType: %s",
		executionInfo.ButtonIndex, props.MenuID, props.PageID, executionInfo.ClickType)

	if executionInfo.ClickType != ClickTypeLeftUp {
		log.Info("OpenPageInMenu: Unhandled click type '%s'. No action taken.", executionInfo.ClickType)
		return nil
	}

	menuID := props.MenuID
	pageID := props.PageID

	xPos, yPos, errMouse := core.GetMousePosition()
	if errMouse != nil {
		log.Error("Error: Failed to get mouse position: %v", errMouse)
		xPos, yPos = 0, 0
	}

	outgoingMessage := core.ShortcutPressed_Message{
		ShortcutPressed:  menuID,
		MouseX:           xPos,
		MouseY:           yPos,
		OpenSpecificPage: true,
		PageID:           pageID,
	}

	natsSubject := os.Getenv("PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED")
	log.Info("Publishing OpenPageInMenu for Menu %d, Page %d at (%d, %d)", menuID, pageID, xPos, yPos)
	a.natsAdapter.PublishMessage(natsSubject, "PieButtonExecution", outgoingMessage)

	return nil
}

func (a *PieButtonExecutionAdapter) handleOpenResource(executionInfo *pieButtonExecute_Message) error {
	var resourceProps core.OpenResourceProperties
	if err := unmarshalProperties(executionInfo.Properties, &resourceProps); err != nil {
		return fmt.Errorf("failed to process properties for open_resource: %w", err)
	}

	log.Info("Button %d - Action: OpenResource - ClickType: %s", executionInfo.ButtonIndex, executionInfo.ClickType)
	log.Info("↳ Resource Path: %s", resourceProps.ResourcePath)

	// Only respond to left-click
	if executionInfo.ClickType == ClickTypeLeftUp {
		// Check if the resource path exists
		if _, err := os.Stat(resourceProps.ResourcePath); os.IsNotExist(err) {
			return fmt.Errorf("resource path does not exist: %s", resourceProps.ResourcePath)
		}

		// Open the file or folder using the system's default application
		err := openFolder(resourceProps.ResourcePath)
		
		// The explorer.exe command often returns exit status 1 even when successful
		// We'll log the error but not return it as an error to avoid false negatives
		if err != nil && err.Error() == "exit status 1" {
			log.Info("Resource opened: %s", resourceProps.ResourcePath)
			return nil
		}
		return err
	}

	return nil
}

// unmarshalProperties safely converts the generic properties map into a specific struct.
func unmarshalProperties(props any, target any) error {
	// Convert the properties to JSON and then unmarshal into the target struct
	propsBytes, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %w", err)
	}

	if err := json.Unmarshal(propsBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal properties: %w", err)
	}

	return nil
}
