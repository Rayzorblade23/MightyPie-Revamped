import {
    type Button,
    type ButtonsConfig,
    type ButtonsOnPageMap,
    ButtonType,
    type PagesInMenuMap
} from './types/pieButtonTypes.ts';
import {
    getPieMenuButtons,
    getPieMenuConfig,
    updateButtonInMenuConfig,
    updatePieMenuButtons,
    updatePieMenuConfig
} from './configManager.svelte.ts';
import {getInstalledAppsInfo} from './installedAppsInfoManager.svelte.ts';
import {getDefaultButton} from './types/pieButtonDefaults.ts';
import {createLogger} from '$lib/logger';

// Create a logger for this component
const logger = createLogger('ConfigValidation');

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
    logger.debug("Validating and syncing config.")
    const apps = getInstalledAppsInfo();
    const menuConfig = getPieMenuButtons() as ButtonsConfig;

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
                        // App exists but icon path needs updating
                        logger.info(`Updating icon path for app ${appKey}`);
                        const newButton = updateButtonIconPath(button, appInfo.iconPath ?? button.properties.icon_path);
                        newConfig = updateButtonInMenuConfig(newConfig, menuId, pageId, buttonId, newButton);
                        updated = true;
                    } else if (!appInfo) {
                        // App doesn't exist anymore, reset the button to default
                        logger.warn(`App ${appKey} no longer exists, resetting button`);
                        const defaultButton = getDefaultButton(button.button_type);

                        // Try to set the icon path for the default button
                        const defaultAppKey = getAppKeyForButton(defaultButton);
                        if (defaultAppKey) {
                            const defaultAppInfo = apps.get(defaultAppKey);
                            if (defaultAppInfo && defaultAppInfo.iconPath) {
                                defaultButton.properties.icon_path = defaultAppInfo.iconPath;
                            }
                        }

                        newConfig = updateButtonInMenuConfig(newConfig, menuId, pageId, buttonId, defaultButton);
                        updated = true;
                    }
                }
            }
        }
    }

    if (updated) {
        logger.info('Updated menu configuration with icon path updates and button resets');
        updatePieMenuButtons(newConfig);
    }

    // --- Validate full PieMenuConfig: shortcuts and starred must reference existing menus/pages ---
    try {
        const pieMenuConfig = getPieMenuConfig();
        if (!pieMenuConfig) return;

        const existingMenuIds = new Set<number>(Array.from((updated ? newConfig : menuConfig).keys()));

        // Validate shortcuts: keep only those whose numeric menuID exists
        const oldShortcuts = pieMenuConfig.shortcuts || {} as Record<string, { codes: number[]; label: string }>;
        const newShortcuts: Record<string, { codes: number[]; label: string }> = {};
        let shortcutsChanged = false;
        for (const [k, v] of Object.entries(oldShortcuts)) {
            const n = Number(k);
            if (!Number.isNaN(n) && existingMenuIds.has(n)) {
                newShortcuts[k] = v;
            } else {
                shortcutsChanged = true; // dropped invalid entry
            }
        }

        // Validate starred: must reference existing menu and page
        let newStarred = pieMenuConfig.starred ?? null as null | { menuID: number; pageID: number };
        let starredChanged = false;
        if (newStarred) {
            const pages = (updated ? newConfig : menuConfig).get(newStarred.menuID);
            if (!pages || !pages.has(newStarred.pageID)) {
                newStarred = null;
                starredChanged = true;
            }
        }

        if (shortcutsChanged || starredChanged) {
            updatePieMenuConfig({
                ...pieMenuConfig,
                shortcuts: newShortcuts,
                starred: newStarred,
            });
            if (shortcutsChanged) logger.warn('Removed invalid shortcut entries referencing non-existent menus');
            if (starredChanged) logger.warn('Cleared invalid starred reference to non-existent menu/page');
        }
    } catch (e) {
        logger.error('Failed validating shortcuts/starred against menu config:', e);
    }
}
