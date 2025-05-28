import {
    type Button,
    type ButtonsOnPageMap,
    ButtonType,
    type MenuConfiguration,
    type PagesInMenuMap
} from './piebuttonTypes';
import {
    getBaseMenuConfiguration,
    publishBaseMenuConfiguration,
    updateBaseMenuConfiguration,
    updateButtonInMenuConfig
} from './configHandler.svelte.ts';
import {getInstalledAppsInfo} from './installedAppsInfoManager.svelte.ts';

// Helper: get the app key for a button
function getAppKeyForButton(button: Button): string | undefined {
    if (button.button_type === ButtonType.ShowProgramWindow) {
        return button.properties.button_text_lower;
    } else if (button.button_type === ButtonType.LaunchProgram) {
        return button.properties.button_text_upper;
    }
    return undefined;
}

// Helper: update a button's icon_path immutably
function updateButtonIconPath(button: Button, iconPath: string): Button {
    if (button.button_type === ButtonType.ShowProgramWindow) {
        return {
            button_type: ButtonType.ShowProgramWindow,
            properties: {
                ...button.properties,
                icon_path: iconPath,
            }
        };
    } else if (button.button_type === ButtonType.LaunchProgram) {
        return {
            button_type: ButtonType.LaunchProgram,
            properties: {
                ...button.properties,
                icon_path: iconPath,
            }
        };
    }
    return button;
}

export function validateAndSyncConfig() {
    const apps = getInstalledAppsInfo();
    const menuConfig = getBaseMenuConfiguration() as MenuConfiguration;

    let updated = false;
    let newConfig = menuConfig;

    for (const [menuId, pagesMap] of menuConfig) {
        for (const [pageId, buttonsMap] of pagesMap as PagesInMenuMap) {
            for (const [buttonId, button] of buttonsMap as ButtonsOnPageMap) {
                if (
                    button.button_type === ButtonType.ShowProgramWindow ||
                    button.button_type === ButtonType.LaunchProgram
                ) {
                    const appKey = getAppKeyForButton(button);
                    if (!appKey) continue;
                    const appInfo = apps.get(appKey);
                    if (appInfo && button.properties.icon_path !== appInfo.iconPath) {
                        const newButton = updateButtonIconPath(button, appInfo.iconPath ?? button.properties.icon_path);
                        newConfig = updateButtonInMenuConfig(newConfig, menuId, pageId, buttonId, newButton);
                        updated = true;
                    }
                }
            }
        }
    }

    if (updated) {
        updateBaseMenuConfiguration(newConfig);
        publishBaseMenuConfiguration(newConfig);
    }
}
