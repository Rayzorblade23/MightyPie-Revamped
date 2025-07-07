import {publishMessage} from "$lib/natsAdapter.svelte.ts";
import {PUBLIC_NATSSUBJECT_SETTINGS_UPDATE} from "$env/static/public";

// --- Type Definitions ---
export interface SettingsEntry {
    label: string;
    isExposed: boolean;
    value: any;
    defaultValue: any;
    type: string;
    options?: string[]; // For enum types
    index?: number; // For sorting order
}

export type SettingsMap = Record<string, SettingsEntry>;

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
 * @param newSettings - The new SettingsMap to apply.
 */
export function updateSettings(newSettings: SettingsMap) {
    settings = newSettings;
}

/**
 * Publishes the current settings to the NATS subject.
 * @param newSettings - The SettingsMap to publish.
 */
export function publishSettings(newSettings: SettingsMap): void {
    publishMessage(PUBLIC_NATSSUBJECT_SETTINGS_UPDATE, newSettings);
}