import {createLogger} from './logger';
import {invoke} from "@tauri-apps/api/core";

const logger = createLogger('AutostartUtils');

// Local storage key for autostart preference
const AUTOSTART_STORAGE_KEY = 'mightypie_autostart_enabled';

/**
 * Enable autostart for the application
 * @returns Promise that resolves when autostart is enabled
 */
export async function enableAutoStart(): Promise<void> {
    try {
        logger.info('Enabling autostart');
        await invoke('enable_autostart');
        // Save preference to local storage
        localStorage.setItem(AUTOSTART_STORAGE_KEY, 'true');
        logger.info('Autostart enabled successfully');
    } catch (error) {
        logger.error('Failed to enable autostart:', error);
        throw error;
    }
}

/**
 * Disable autostart for the application
 * @returns Promise that resolves when autostart is disabled
 */
export async function disableAutoStart(): Promise<void> {
    try {
        logger.info('Disabling autostart');
        await invoke('disable_autostart');
        // Save preference to local storage
        localStorage.setItem(AUTOSTART_STORAGE_KEY, 'false');
        logger.info('Autostart disabled successfully');
    } catch (error) {
        logger.error('Failed to disable autostart:', error);
        throw error;
    }
}

/**
 * Check if autostart is enabled for the application
 * @returns Promise that resolves to true if autostart is enabled, false otherwise
 */
export async function isAutoStartEnabled(): Promise<boolean> {
    try {
        logger.info('Checking if autostart is enabled');
        const isEnabled = await invoke<boolean>('is_autostart_enabled');
        // Update local storage to match system state
        localStorage.setItem(AUTOSTART_STORAGE_KEY, isEnabled ? 'true' : 'false');
        logger.info(`Autostart is ${isEnabled ? 'enabled' : 'disabled'}`);
        return isEnabled;
    } catch (error) {
        logger.error('Failed to check if autostart is enabled:', error);
        throw error;
    }
}

/**
 * Get the saved autostart preference from local storage
 * @returns The saved preference, or null if not set
 */
export function getSavedAutoStartPreference(): boolean | null {
    const saved = localStorage.getItem(AUTOSTART_STORAGE_KEY);
    if (saved === 'true') return true;
    if (saved === 'false') return false;
    return null;
}

/**
 * Synchronize the local storage preference with the actual system state
 * This should be called on app startup to ensure the UI reflects the actual state
 */
export async function syncAutoStartPreference(): Promise<boolean> {
    try {
        const systemEnabled = await isAutoStartEnabled();
        return systemEnabled;
    } catch (error) {
        logger.error('Failed to sync autostart preference:', error);
        // If we can't check, return the saved preference or default to false
        const saved = getSavedAutoStartPreference();
        return saved !== null ? saved : false;
    }
}
