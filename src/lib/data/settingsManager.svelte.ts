import {publishMessage} from "$lib/natsAdapter.svelte.ts";
import {PUBLIC_NATSSUBJECT_SETTINGS_UPDATE} from "$env/static/public";
import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('SettingsManager');

// --- Type Definitions ---
export interface SettingsEntry {
    label: string;
    isExposed: boolean;
    value: any;
    defaultValue: any;
    type: string;
    options?: string[]; // For enum types
    index?: number; // For sorting order within category
    category?: string; // For grouping settings in UI
}

export type SettingsMap = Record<string, SettingsEntry>;

// Default category mapping for migration of old settings
const DEFAULT_CATEGORIES: Record<string, string> = {
    startInPieMenuConfig: "General",
    keepPieMenuAnchored: "Pie Menu Behavior",
    pieMenuDeadzoneFunction: "Pie Menu Behavior",
    mouseWheelWhileOpen: "Pie Menu Behavior",
    autoScrollOverflow: "Button Appearance",
    pieButtonBorderThickness: "Button Appearance",
    colorAccentAnyWin: "Button Appearance",
    colorAccentFunction: "Button Appearance",
    colorAccentLaunch: "Button Appearance",
    colorAccentProgramWin: "Button Appearance",
    colorAccentOpenPage: "Button Appearance",
    colorAccentResource: "Button Appearance",
    colorAccentShortcut: "Button Appearance",
    colorPieButtonHighlight: "Menu Appearance",
    colorIndicator: "Menu Appearance",
    colorRingFill: "Menu Appearance",
    colorRingStroke: "Menu Appearance",
    exampleFloat: "Debug",
    exampleInt: "Debug",
};

/**
 * Migrates settings to ensure they have the category field.
 * This is called when settings are loaded from the backend.
 * @param settingsData - The settings data to migrate.
 * @returns The migrated settings data with categories.
 */
function migrateSettingsCategories(settingsData: SettingsMap): SettingsMap {
    let migrated = false;
    const migratedSettings: SettingsMap = {};

    for (const [key, entry] of Object.entries(settingsData)) {
        if (!entry.category && DEFAULT_CATEGORIES[key]) {
            migrated = true;
            migratedSettings[key] = {
                ...entry,
                category: DEFAULT_CATEGORIES[key]
            };
        } else {
            migratedSettings[key] = entry;
        }
    }

    if (migrated) {
        logger.info("Migrated settings to include categories");
    }

    return migratedSettings;
}

// --- Svelte State and Public API ---
let settings = $state<SettingsMap>({});

/**
 * Getter for the global settings.
 * @returns The current SettingsMap.
 */
export function getSettings(): SettingsMap {
    return settings;
}

/**
 * Setter for updating the global settings.
 * Automatically migrates settings to include categories if missing.
 * @param newSettings - The new SettingsMap to apply.
 */
export function updateSettings(newSettings: SettingsMap) {
    settings = migrateSettingsCategories(newSettings);
    logger.info("Settings updated.");
}

/**
 * Publishes the current settings to the NATS subject.
 * The published settings will include categories after migration.
 * @param newSettings - The SettingsMap to publish.
 */
export function publishSettings(newSettings: SettingsMap): void {
    const migratedSettings = migrateSettingsCategories(newSettings);
    publishMessage(PUBLIC_NATSSUBJECT_SETTINGS_UPDATE, migratedSettings);
}