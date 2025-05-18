import {
    ButtonType,
    type ConfigData,
    type MenuConfiguration,
    type PageConfiguration,
    type ProfileConfiguration
} from "$lib/components/piebutton/piebuttonTypes.ts";


// Internal state
let profileConfiguration = $state<ProfileConfiguration>(new Map());

// Getter for external access
export function getProfileConfiguration(): ProfileConfiguration {
    return profileConfiguration;
}

// Setter for updating the configuration
export function updateProfileConfiguration(newConfig: ProfileConfiguration) {
    profileConfiguration = newConfig;
}

// --- Parsing Function (Handles Nested Input with Profiles) ---
export function parseNestedRawConfig(data: ConfigData): ProfileConfiguration {
    const newProfileConfig: ProfileConfiguration = new Map();

    Object.entries(data).forEach(([profileKey, profileData]) => {
        const profileIndex = parseInt(profileKey, 10);
        if (isNaN(profileIndex)) {
            console.warn(`Invalid profile key: ${profileKey}, skipping.`);
            return;
        }

        const menuConfiguration: MenuConfiguration = new Map();

        Object.entries(profileData).forEach(([menuKey, menuData]) => {
            const menuIndex = parseInt(menuKey, 10);
            if (isNaN(menuIndex)) {
                console.warn(`Invalid menu key: ${menuKey} for profile ${profileIndex}, skipping.`);
                return;
            }

            const buttonMapForMenu: PageConfiguration = new Map();

            Object.entries(menuData).forEach(([buttonKey, buttonData]) => {
                const buttonIndex = parseInt(buttonKey, 10);
                if (isNaN(buttonIndex)) {
                    console.warn(`Invalid button key: ${buttonKey} for menu ${menuIndex}, profile ${profileIndex}, skipping.`);
                    return;
                }

                // The internal logic for parsing individual buttonData remains the same
                switch (buttonData.button_type) {
                    case ButtonType.LaunchProgram:
                        if (buttonData.properties) {
                            buttonMapForMenu.set(buttonIndex, {
                                button_type: ButtonType.LaunchProgram,
                                properties: {
                                    button_text_upper: buttonData.properties.button_text_upper ?? '',
                                    button_text_lower: buttonData.properties.button_text_lower ?? '',
                                    icon_path: buttonData.properties.icon_path ?? '',
                                    exe_path: buttonData.properties.exe_path ?? ''
                                }
                            });
                        }
                        break;

                    case ButtonType.ShowProgramWindow:
                    case ButtonType.ShowAnyWindow:
                        if (buttonData.properties) {
                            buttonMapForMenu.set(buttonIndex, {
                                button_type: buttonData.button_type as ButtonType.ShowProgramWindow | ButtonType.ShowAnyWindow,
                                properties: {
                                    button_text_upper: buttonData.properties.button_text_upper ?? '',
                                    button_text_lower: buttonData.properties.button_text_lower ?? '',
                                    icon_path: buttonData.properties.icon_path ?? '',
                                    window_handle: buttonData.properties.window_handle ?? 0,
                                    exe_path: buttonData.properties.exe_path ?? ''
                                }
                            });
                        }
                        break;

                    case ButtonType.CallFunction:
                        if (buttonData.properties) {
                            buttonMapForMenu.set(buttonIndex, {
                                button_type: ButtonType.CallFunction,
                                properties: {
                                    button_text_upper: buttonData.properties.button_text_upper ?? '',
                                    button_text_lower: buttonData.properties.button_text_lower ?? '',
                                    icon_path: buttonData.properties.icon_path ?? ''
                                }
                            });
                        }
                        break;

                    default:
                        buttonMapForMenu.set(buttonIndex, {
                            button_type: ButtonType.Disabled
                        });
                }
            });
            menuConfiguration.set(menuIndex, buttonMapForMenu);
        });
        newProfileConfig.set(profileIndex, menuConfiguration);
    });

    return newProfileConfig;
}

// Accessor functions
export function getButtonProperties(profileIndex: number, menuIndex: number, buttonIndex: number) {
    const menuConfig = profileConfiguration.get(profileIndex);
    const buttonMap = menuConfig?.get(menuIndex);
    const button = buttonMap?.get(buttonIndex);
    return button && 'properties' in button ? button.properties : undefined;
}

export function getButtonType(profileIndex: number, menuIndex: number, buttonIndex: number) {
    const menuConfig = profileConfiguration.get(profileIndex);
    const buttonMap = menuConfig?.get(menuIndex);
    return buttonMap?.get(buttonIndex)?.button_type;
}

/**
 * Checks if a specific Menu Index exists within the configuration
 * for a given Profile Index.
 * (Uses the numeric index types you provided)
 *
 * @param profileIndex - The index of the profile to check.
 * @param menuIndex - The index of the menu to look for within that profile.
 * @returns True if the menu exists for the profile, false otherwise.
 */
export function hasPageForMenu(profileIndex: number, menuIndex: number): boolean {
    // Get the current configuration map
    const config = getProfileConfiguration(); // Or access internal `profilesConfiguration`

    // Check if the profile index exists and then if the menu index exists within that profile's map
    return config.get(profileIndex)?.has(menuIndex) ?? false;
}