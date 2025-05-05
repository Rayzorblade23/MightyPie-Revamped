import {
    type ConfigData,
    type MenuConfiguration,
    type Task,
    TaskType
} from "$lib/components/piebutton/piebuttonTypes.ts";

import {subscribeToTopic} from "$lib/natsAdapter.ts";
import {getEnvVar} from "$lib/envHandler.ts";


// Internal state
let menuConfiguration = $state<MenuConfiguration>(new Map());

// Getter for external access
export function getMenuConfiguration(): MenuConfiguration {
    return menuConfiguration;
}

// Setter for updating the configuration
export function updateMenuConfiguration(newConfig: Map<number, Map<number, Task>>) {
    menuConfiguration = newConfig;
}


// Update the subscriber
subscribeToTopic(getEnvVar("NATSSUBJECT_BUTTONMANAGER_UPDATE"), message => {
    try {
        const configData: ConfigData = JSON.parse(message);
        const newConfig = parseNestedRawConfig(configData);
        updateMenuConfiguration(newConfig);
    } catch (e) {
        console.error('Failed to parse button manager update:', e);
    }
}).catch(error => {
    console.error('Failed to subscribe to NATS topic:', error);
});


// --- 3. Parsing Function (Handles Nested Input) ---


export function parseNestedRawConfig(data: ConfigData) {
    const newConfig = new Map<number, Map<number, Task>>();

    Object.entries(data).forEach(([menuKey, menuData]) => {
        const buttonMap = new Map<number, Task>();
        const menuIndex = parseInt(menuKey);

        Object.entries(menuData).forEach(([buttonKey, taskData]) => {
            const buttonIndex = parseInt(buttonKey);

            switch (taskData.task_type) {
                case TaskType.LaunchProgram:
                    if (taskData.properties) {
                        buttonMap.set(buttonIndex, {
                            task_type: TaskType.LaunchProgram,
                            properties: {
                                button_text_upper: taskData.properties.button_text_upper ?? '',
                                button_text_lower: taskData.properties.button_text_lower ?? '',
                                icon_path: taskData.properties.icon_path ?? '',
                                exe_path: taskData.properties.exe_path ?? ''
                            }
                        });
                    }
                    break;

                case TaskType.ShowProgramWindow:
                case TaskType.ShowAnyWindow:
                    if (taskData.properties) {
                        buttonMap.set(buttonIndex, {
                            task_type: taskData.task_type as TaskType.ShowProgramWindow | TaskType.ShowAnyWindow,
                            properties: {
                                button_text_upper: taskData.properties.button_text_upper ?? '',
                                button_text_lower: taskData.properties.button_text_lower ?? '',
                                icon_path: taskData.properties.icon_path ?? '',
                                window_handle: taskData.properties.window_handle ?? 0,
                                exe_path: taskData.properties.exe_path ?? ''
                            }
                        });
                    }
                    break;

                case TaskType.CallFunction:
                    if (taskData.properties) {
                        buttonMap.set(buttonIndex, {
                            task_type: TaskType.CallFunction,
                            properties: {
                                button_text_upper: taskData.properties.button_text_upper ?? '',
                                button_text_lower: taskData.properties.button_text_lower ?? '',
                                icon_path: taskData.properties.icon_path ?? ''
                            }
                        });
                    }
                    break;

                default:
                    buttonMap.set(buttonIndex, {
                        task_type: TaskType.Disabled
                    });
            }
        });

        newConfig.set(menuIndex, buttonMap);
    });

    return newConfig;
}

// In piebuttonConfig.svelte.ts
export function getTaskProperties(menuIndex: number, buttonIndex: number) {
    const buttonMap = menuConfiguration.get(menuIndex);
    const task = buttonMap?.get(buttonIndex);
    return task && 'properties' in task ? task.properties : undefined;
}

export function getTaskType(menuIndex: number, buttonIndex: number) {
    const buttonMap = menuConfiguration.get(menuIndex);
    return buttonMap?.get(buttonIndex)?.task_type;
}