import {
    type ButtonMap,
    type ConfigData,
    type MenuConfiguration,
    type ProfilesConfiguration,
    TaskType
} from "$lib/components/piebutton/piebuttonTypes.ts";

import {subscribeToTopic} from "$lib/natsAdapter.ts";
import {PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE} from "$env/static/public";


// Internal state
let profilesConfiguration = $state<ProfilesConfiguration>(new Map());

// Getter for external access
export function getProfilesConfiguration(): ProfilesConfiguration {
    return profilesConfiguration;
}

// Setter for updating the configuration
export function updateProfilesConfiguration(newConfig: ProfilesConfiguration) {
    profilesConfiguration = newConfig;
}


subscribeToTopic(PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE, message => {
    try {
        const configData: ConfigData = JSON.parse(message); // Uses updated ConfigData type
        const newParsedConfig = parseNestedRawConfig(configData);
        updateProfilesConfiguration(newParsedConfig);
        console.log('Button configuration updated:', newParsedConfig); // Optional: for debugging
    } catch (e) {
        console.error('Failed to parse button manager update:', e);
    }
}).catch(error => {
    console.error('Failed to subscribe to NATS topic:', error);
});


// --- Parsing Function (Handles Nested Input with Profiles) ---
export function parseNestedRawConfig(data: ConfigData): ProfilesConfiguration {
    const newProfilesConfig: ProfilesConfiguration = new Map();

    Object.entries(data).forEach(([profileKey, profileData]) => {
        const profileIndex = parseInt(profileKey, 10);
        if (isNaN(profileIndex)) {
            console.warn(`Invalid profile key: ${profileKey}, skipping.`);
            return;
        }

        const menuConfigForProfile: MenuConfiguration = new Map();

        Object.entries(profileData).forEach(([menuKey, menuData]) => {
            const menuIndex = parseInt(menuKey, 10);
            if (isNaN(menuIndex)) {
                console.warn(`Invalid menu key: ${menuKey} for profile ${profileIndex}, skipping.`);
                return;
            }

            const buttonMapForMenu: ButtonMap = new Map();

            Object.entries(menuData).forEach(([buttonKey, taskData]) => {
                const buttonIndex = parseInt(buttonKey, 10);
                if (isNaN(buttonIndex)) {
                    console.warn(`Invalid button key: ${buttonKey} for menu ${menuIndex}, profile ${profileIndex}, skipping.`);
                    return;
                }

                // The internal logic for parsing individual taskData remains the same
                switch (taskData.task_type) {
                    case TaskType.LaunchProgram:
                        if (taskData.properties) {
                            buttonMapForMenu.set(buttonIndex, {
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
                            buttonMapForMenu.set(buttonIndex, {
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
                            buttonMapForMenu.set(buttonIndex, {
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
                        buttonMapForMenu.set(buttonIndex, {
                            task_type: TaskType.Disabled
                        });
                }
            });
            menuConfigForProfile.set(menuIndex, buttonMapForMenu);
        });
        newProfilesConfig.set(profileIndex, menuConfigForProfile);
    });

    return newProfilesConfig;
}

// Accessor functions
export function getTaskProperties(profileIndex: number, menuIndex: number, buttonIndex: number) {
    const menuConfig = profilesConfiguration.get(profileIndex);
    const buttonMap = menuConfig?.get(menuIndex);
    const task = buttonMap?.get(buttonIndex);
    return task && 'properties' in task ? task.properties : undefined;
}

export function getTaskType(profileIndex: number, menuIndex: number, buttonIndex: number) {
    const menuConfig = profilesConfiguration.get(profileIndex);
    const buttonMap = menuConfig?.get(menuIndex);
    return buttonMap?.get(buttonIndex)?.task_type;
}