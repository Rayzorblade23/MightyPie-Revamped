import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('ShortcutLabelsManager');

export interface ShortcutEntry {
    codes: number[];
    label: string;
}

export type ShortcutLabelsMessage = Record<number, ShortcutEntry>;

// --- Svelte State and Public API ---
let _shortcutLabels = $state<Record<number, string>>({});

/**
 * Getter for the global shortcut labels.
 * @returns The current shortcutLabels object.
 */
export function getShortcutLabels(): Record<number, string> {
    return _shortcutLabels;
}

/**
 * Setter for updating the global shortcut labels.
 * @param newLabels - A Record<number, string> of shortcut labels.
 */
export function updateShortcutLabels(newLabels: Record<number, string>) {
    logger.debug('updateShortcutLabels called with:', newLabels);
    _shortcutLabels = newLabels;
}

/**
 * Parses the full shortcut labels message and extracts only the labels.
 * @param msg - The raw NATS message string.
 * @returns A Record<number, string> mapping indices to labels.
 */
export function parseShortcutLabelsMessage(msg: string): Record<number, string> {
    const fullObj = JSON.parse(msg) as ShortcutLabelsMessage;
    const labels: Record<number, string> = {};
    for (const key in fullObj) {
        logger.debug('Parsing key:', key, 'typeof:', typeof key);
        const numKey = Number(key);
        if (!isNaN(numKey)) {
            labels[numKey] = fullObj[key].label;
        } else {
            logger.warn('Invalid key in shortcut labels:', key);
        }
    }
    return labels;
}