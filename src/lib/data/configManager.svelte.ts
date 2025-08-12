// configHandler.svelte.ts

import {
    type Button,
    type ButtonData,
    type ButtonsOnPageMap,
    ButtonType,
    type CallFunctionProperties,
    type ConfigData,
    type LaunchProgramProperties,
    type MenuConfiguration,
    type OpenResourceProperties,
    type OpenSpecificPieMenuPageProperties,
    type PagesInMenuMap,
    type ShowAnyWindowProperties,
    type ShowProgramWindowProperties
} from "$lib/data/types/pieButtonTypes.ts";
import {publishMessage} from "$lib/natsAdapter.svelte.ts";
import {getDefaultButton} from "$lib/data/types/pieButtonDefaults.ts";
import {PUBLIC_NATSSUBJECT_PIEMENUCONFIG_UPDATE} from "$env/static/public";
import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('ConfigManager');

// --- Svelte State and Public API ---

let menuConfiguration = $state<MenuConfiguration>(new Map());
let baseMenuConfiguration = $state<MenuConfiguration>(new Map());

/**
 * Getter for the global menu configuration.
 * @returns The current MenuConfiguration.
 */
export function getMenuConfiguration(): MenuConfiguration {
    return menuConfiguration;
}

/**
 * Setter for updating the global menu configuration.
 * @param newConfig - The new MenuConfiguration to apply.
 */
export function updateMenuConfiguration(newConfig: MenuConfiguration) {
    menuConfiguration = newConfig;
}

/**
 * Getter for the base menu configuration.
 * @returns The current MenuConfiguration.
 */
export function getBaseMenuConfiguration(): MenuConfiguration {
    return baseMenuConfiguration;
}

/**
 * Setter for updating the base menu configuration.
 * @param newConfig - The new MenuConfiguration to apply.
 */
export function updateBaseMenuConfiguration(newConfig: MenuConfiguration) {
    baseMenuConfiguration = newConfig;
    logger.info("Base Menu Config updated.")
}

/**
 * Parses the entire pie menu button configuration from its raw data format
 * (typically from buttonConfig.json) into the structured MenuConfiguration used by the application.
 *
 * @param configInput - The raw ConfigData object (e.g., { "0": { "0": { "0": {...} } } }).
 * @returns A fully parsed MenuConfiguration map.
 */
export function parseButtonConfig(configInput: ConfigData): MenuConfiguration {
    const newMenuConfig: MenuConfiguration = new Map();

    Object.entries(configInput).forEach(([menuKey, menuPageData]) => {
        const menuId = parseInt(menuKey, 10);
        if (isNaN(menuId)) {
            logger.warn(`Invalid menu key: "${menuKey}" in configuration. Skipping menu.`);
            return;
        }
        newMenuConfig.set(menuId, buildPagesMapForMenu(menuPageData, menuId));
    });

    return newMenuConfig;
}

/**
 * Retrieves the properties of a specific button.
 * @param menuID - The ID of the menu.
 * @param pageID - The ID of the page.
 * @param buttonID - The ID of the button.
 * @returns The button's properties if found and the button is not disabled, otherwise undefined.
 */
export function getButtonProperties(menuID: number, pageID: number, buttonID: number) {
    const pagesInMenu = menuConfiguration.get(menuID);
    const buttonsOnPage = pagesInMenu?.get(pageID);
    const button = buttonsOnPage?.get(buttonID);

    // Safely access properties only if the button exists and is not the 'Disabled' type
    if (button && button.button_type !== ButtonType.Disabled) {
        return button.properties;
    }
    return undefined;
}

/**
 * Retrieves the type of a specific button.
 * @param menuID - The ID of the menu.
 * @param pageID - The ID of the page.
 * @param buttonID - The ID of the button.
 * @returns The button's type (ButtonType) if found, otherwise undefined.
 */
export function getButtonType(menuID: number, pageID: number, buttonID: number) {
    const pagesInMenu = menuConfiguration.get(menuID);
    const buttonsOnPage = pagesInMenu?.get(pageID);
    const button = buttonsOnPage?.get(buttonID);
    return button?.button_type;
}

/**
 * Checks if a specific Page Index exists within the configuration
 * for a given Menu Index.
 *
 * @param menuID - The index of the menu to check.
 * @param pageID - The index of the page to look for within that menu.
 * @returns True if the page exists for the menu, false otherwise.
 */
export function hasPageForMenu(menuID: number, pageID: number): boolean {
    return menuConfiguration.get(menuID)?.has(pageID) ?? false;
}

// --- Internal Helper Functions for Parsing ---

/**
 * Constructs a map of page configurations for a single menu from its raw configuration data.
 * Each page configuration is itself a map of its buttons.
 *
 * @param menuPageData - The raw page data for a menu, keyed by page ID string.
 *                       (e.g., { "0": { "0": { "button_type": "..." } }, ... })
 * @param menuId - The ID of the current menu.
 * @returns A Map where keys are numeric page IDs and values are PagesInMenuMap (actually ButtonsOnPageMap).
 */
function buildPagesMapForMenu(
    menuPageData: Record<string, Record<string, ButtonData>>,
    menuId: number
): PagesInMenuMap { // The return type should be PagesInMenuMap which is Map<number, ButtonsOnPageMap>
    const pagesInMenu: PagesInMenuMap = new Map();

    Object.entries(menuPageData).forEach(([pageKey, pageButtonData]) => {
        const pageId = parseInt(pageKey, 10);
        if (isNaN(pageId)) {
            logger.warn(`Invalid page key: "${pageKey}" for menu ${menuId}. Skipping page.`);
            return;
        }
        pagesInMenu.set(pageId, buildButtonsMapForPage(pageButtonData, menuId, pageId));
    });

    return pagesInMenu;
}

/**
 * Constructs a map of typed Buttons for a single page from its raw configuration data.
 *
 * @param pageButtonData - The raw button data for a page, keyed by button ID string.
 *                         (e.g., { "0": { "button_type": "...", ... }, ... })
 * @param menuId - The ID of the parent menu.
 * @param pageId - The ID of the current page.
 * @returns A Map where keys are numeric button IDs and values are typed Button objects (ButtonsOnPageMap).
 */
function buildButtonsMapForPage(
    pageButtonData: Record<string, ButtonData>,
    menuId: number,
    pageId: number
): ButtonsOnPageMap {
    const buttonsOnPage: ButtonsOnPageMap = new Map();

    Object.entries(pageButtonData).forEach(([buttonKey, rawButton]) => {
        const buttonId = parseInt(buttonKey, 10);
        if (isNaN(buttonId)) {
            logger.warn(`Invalid button key: "${buttonKey}" for page ${pageId}, menu ${menuId}. Skipping button.`);
            return;
        }
        buttonsOnPage.set(buttonId, convertToButton(rawButton, menuId, pageId, buttonId));
    });

    return buttonsOnPage;
}

/**
 * Converts a single raw button data object from the configuration
 * into a strongly-typed Button object.
 * It handles property validation and defaults to a Disabled button if data is invalid.
 *
 * @param buttonInput - The raw button data (ButtonData from config).
 * @param menuId - The ID of the menu this button belongs to (for logging context).
 * @param pageId - The ID of the page this button belongs to (for logging context).
 * @param buttonId - The ID of the button (for logging context).
 * @returns A typed Button object.
 */
function convertToButton(
    buttonInput: ButtonData,
    menuId: number,
    pageId: number,
    buttonId: number
): Button {
    const {button_type, properties} = buttonInput;

    const createLogMessage = (issue: string) =>
        `${issue} for button ${buttonId} (type: ${button_type || 'unknown'}) on page ${pageId}, menu ${menuId}. Defaulting to Disabled.`;

    switch (button_type) {
        case ButtonType.ShowAnyWindow:
            if (!properties) {
                logger.warn(createLogMessage("Properties missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as ShowAnyWindowProperties};

        case ButtonType.ShowProgramWindow:
            if (!properties) {
                logger.warn(createLogMessage("Properties missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as ShowProgramWindowProperties};

        case ButtonType.LaunchProgram:
            if (!properties) {
                logger.warn(createLogMessage("Properties missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as LaunchProgramProperties};

        case ButtonType.CallFunction:
            if (!properties) {
                logger.warn(createLogMessage("Properties object missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as CallFunctionProperties};

        case ButtonType.OpenSpecificPieMenuPage:
            if (!properties) {
                logger.warn(createLogMessage("Properties missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as OpenSpecificPieMenuPageProperties};

        case ButtonType.OpenResource:
            if (!properties) {
                logger.warn(createLogMessage("Properties missing"));
                return getDefaultButton(ButtonType.Disabled);
            }
            return {button_type, properties: properties as OpenResourceProperties};

        case ButtonType.Disabled:
            return getDefaultButton(ButtonType.Disabled);

        default:
            logger.warn(createLogMessage("Unknown or missing button type"));
            return getDefaultButton(ButtonType.Disabled);
    }
}

/**
 * Converts the structured MenuConfiguration back into the raw ConfigData format
 * and publishes it to the appropriate NATS subject.
 *
 * @param menuConfig - The MenuConfiguration map to unparse and publish.
 */
export function publishBaseMenuConfiguration(menuConfig: MenuConfiguration): void {
    // Unparse MenuConfiguration into ConfigData
    const configData: ConfigData = {};

    menuConfig.forEach((pagesInMenu, menuId) => {
        const menuKey = menuId.toString();
        configData[menuKey] = {};

        pagesInMenu.forEach((buttonsOnPage, pageId) => {
            const pageKey = pageId.toString();
            configData[menuKey][pageKey] = {};

            buttonsOnPage.forEach((button, buttonId) => {
                const buttonKey = buttonId.toString();
                configData[menuKey][pageKey][buttonKey] = {
                    button_type: button.button_type,
                    properties: button.properties,
                };
            });
        });
    });

    // Publish the ConfigData
    logger.info("Publishing updated Base Menu Configuration");
    publishMessage<ConfigData>(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_UPDATE, configData);
}

/**
 * Adds a new empty page to a specified menu within a given MenuConfiguration.
 * This function operates on a copy of the input configuration and returns the modified copy.
 * Newly added pages will be pre-filled with default 'ShowAnyWindow' buttons for 8 slots.
 *
 * @param currentMenuConfig - The MenuConfiguration to modify.
 * @param menuIdToAddPageTo - The ID of the menu to which the new page will be added.
 * @returns An object containing the new MenuConfiguration with the added page and the ID of the new page,
 *          or null if the menuIdToAddPageTo is invalid or not found.
 */
export function addPageToMenuConfiguration(
    currentMenuConfig: MenuConfiguration,
    menuIdToAddPageTo: number
): { newConfig: MenuConfiguration; newPageID: number } | null {
    if (!currentMenuConfig.has(menuIdToAddPageTo) && menuIdToAddPageTo !== 0 && currentMenuConfig.size > 0) {
        logger.warn(`Menu ID ${menuIdToAddPageTo} not found in the provided configuration.`);
        // Allow creating the first menu (ID 0) if the config is empty
        if (menuIdToAddPageTo !== 0 || currentMenuConfig.size > 0) {
            return null;
        }
    }

    const newConfig = new Map(currentMenuConfig);
    const menuToUpdate = new Map(newConfig.get(menuIdToAddPageTo) ?? new Map<number, ButtonsOnPageMap>());

    const existingPageKeys = Array.from(menuToUpdate.keys());
    const newPageId = existingPageKeys.length > 0 ? Math.max(...existingPageKeys) + 1 : 0;

    const newPageData: ButtonsOnPageMap = new Map();
    const numberOfSlots = 8; // Pre-fill 8 slots

    for (let i = 0; i < numberOfSlots; i++) {
        const defaultButton = getDefaultButton(ButtonType.ShowAnyWindow);
        newPageData.set(i, defaultButton);
    }

    menuToUpdate.set(newPageId, newPageData);
    newConfig.set(menuIdToAddPageTo, menuToUpdate);

    return {newConfig, newPageID: newPageId};
}

/**
 * Adds a new empty menu to the MenuConfiguration.
 * The new menu will be pre-filled with a single page (ID 0),
 * which itself is pre-filled with default 'ShowAnyWindow' buttons for 8 slots.
 *
 * @param currentMenuConfig - The MenuConfiguration to modify.
 * @returns An object containing the new MenuConfiguration with the added menu and the ID of the new menu.
 */
export function addMenuToMenuConfiguration(
    currentMenuConfig: MenuConfiguration
): { newConfig: MenuConfiguration; newMenuID: number } {
    // Find the next available menu ID
    const existingMenuKeys = Array.from(currentMenuConfig.keys());
    const newMenuId = existingMenuKeys.length > 0 ? Math.max(...existingMenuKeys) + 1 : 0;

    // Create default buttons for the new page (8 slots)
    const buttonsOnPage: ButtonsOnPageMap = new Map();
    const numberOfSlots = 8;
    for (let buttonId = 0; buttonId < numberOfSlots; buttonId++) {
        const defaultButton = getDefaultButton(ButtonType.ShowAnyWindow);
        buttonsOnPage.set(buttonId, defaultButton);
    }

    // Create the default page (ID 0)
    const pagesInMenu: PagesInMenuMap = new Map();
    pagesInMenu.set(0, buttonsOnPage);

    // Clone the config and add the new menu
    const newConfig = new Map(currentMenuConfig);
    newConfig.set(newMenuId, pagesInMenu);

    return {newConfig, newMenuID: newMenuId};
}

/**
 * Removes a page from a specified menu within a given MenuConfiguration.
 * If the menu becomes empty after removing the page, the menu itself is removed.
 * Remaining pages in the affected menu are re-indexed.
 * This function operates on a copy of the input configuration and returns the modified copy.
 *
 * @param currentMenuConfig - The MenuConfiguration to modify.
 * @param menuIdToRemoveFrom - The ID of the menu from which the page will be removed.
 * @param pageIdToRemove - The ID of the page to remove.
 * @returns The new MenuConfiguration with the page removed (and potentially the menu removed if it became empty),
 *          or null if the menu or page ID is invalid.
 */
export function removePageFromMenuConfiguration(
    currentMenuConfig: MenuConfiguration,
    menuIdToRemoveFrom: number,
    pageIdToRemove: number
): MenuConfiguration | null {
    if (!currentMenuConfig.has(menuIdToRemoveFrom)) {
        logger.warn(`Menu ID ${menuIdToRemoveFrom} not found in current configuration.`);
        return null;
    }

    const newConfig = new Map(currentMenuConfig);
    const currentPagesInMenu = newConfig.get(menuIdToRemoveFrom);

    if (!currentPagesInMenu || !currentPagesInMenu.has(pageIdToRemove)) {
        logger.warn(`Page ID ${pageIdToRemove} not found in menu ${menuIdToRemoveFrom}.`);
        return null;
    }

    const reIndexedPagesInMenu: PagesInMenuMap = new Map();
    let newPageIndex = 0;

    // Sort page entries by their original ID to ensure correct re-indexing
    const sortedPages = Array.from(currentPagesInMenu.entries()).sort(
        ([pageID_A], [pageID_B]) => pageID_A - pageID_B
    );

    for (const [currentPageId, buttonsOnPage] of sortedPages) {
        // Start of changes for the loop body
        if (currentPageId !== pageIdToRemove) { // Skip the page to be removed
            reIndexedPagesInMenu.set(newPageIndex, buttonsOnPage);
            newPageIndex++;
        }
        // End of changes for the loop body
    }

    newConfig.set(menuIdToRemoveFrom, reIndexedPagesInMenu);

    return newConfig;
}

/**
 * Removes a menu from the MenuConfiguration and re-indexes the remaining menus to close any gaps.
 *
 * @param currentMenuConfig - The MenuConfiguration to modify.
 * @param menuIdToRemove - The ID of the menu to remove.
 * @returns The new MenuConfiguration with the menu removed and indexes closed, or null if menuIdToRemove is invalid.
 */
export function removeMenuFromMenuConfiguration(
    currentMenuConfig: MenuConfiguration,
    menuIdToRemove: number
): MenuConfiguration | null {
    if (!currentMenuConfig.has(menuIdToRemove)) {
        logger.warn(`Menu ID ${menuIdToRemove} not found in the provided configuration.`);
        return null;
    }
    // Remove the menu
    const tempConfig = new Map(currentMenuConfig);
    tempConfig.delete(menuIdToRemove);
    // Re-index menus
    const sortedMenuEntries = Array.from(tempConfig.entries()).sort(
        ([menu_A], [menu_B]) => menu_A - menu_B);
    const newConfig: MenuConfiguration = new Map();
    sortedMenuEntries.forEach(([newMenuID, pagesInMenu]) => {
        // Re-index pages and buttons inside each menu if needed (but keep as is for now)
        newConfig.set(newMenuID, pagesInMenu);
    });
    return newConfig;
}

/**
 * Updates a button in the MenuConfiguration map for a specific menu, page, and button slot.
 * Returns a new MenuConfiguration with the update applied.
 *
 * @param config - The MenuConfiguration to update.
 * @param menuID - The ID of the menu to update.
 * @param pageID - The ID of the page to update.
 * @param buttonID - The ID of the button slot to update.
 * @param newButton - The new Button to set at the specified slot.
 * @returns The updated MenuConfiguration.
 */
export function updateButtonInMenuConfig(
    config: MenuConfiguration,
    menuID: number,
    pageID: number,
    buttonID: number,
    newButton: Button
): MenuConfiguration {
    const newConfig = new Map(config);
    const menuToUpdate = new Map(newConfig.get(menuID) ?? new Map<number, ButtonsOnPageMap>());
    const pageToUpdate = new Map(menuToUpdate.get(pageID) ?? new Map<number, Button>());
    pageToUpdate.set(buttonID, newButton);
    menuToUpdate.set(pageID, pageToUpdate);
    newConfig.set(menuID, menuToUpdate);
    return newConfig;
}