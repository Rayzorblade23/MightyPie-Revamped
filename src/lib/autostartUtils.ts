import {createLogger} from './logger';
import {invoke} from "@tauri-apps/api/core";

const logger = createLogger('AutostartUtils');

// Local storage key for autostart preference
const AUTOSTART_STORAGE_KEY = 'mightypie_autostart_enabled';
// Local storage key for admin rights preference
const ADMIN_RIGHTS_STORAGE_KEY = 'mightypie_admin_rights_enabled';

/**
 * Enable autostart for the application
 * @param withAdminRights Whether to run with admin rights
 * @returns Promise that resolves when autostart is enabled or rejects with an error
 */
export async function enableAutoStart(withAdminRights: boolean = false): Promise<void> {
    try {
        logger.info(`Enabling autostart${withAdminRights ? ' with admin rights' : ''}`);
        
        // Check if we're already running as admin
        const isAdmin = await isRunningAsAdmin();
        logger.info(`Current admin status: ${isAdmin}`);
        
        // Try to create the startup task
        try {
            await invoke('create_startup_task', { runAsAdmin: withAdminRights });
            
            logger.info('Autostart enabled successfully');
            return;
        } catch (error: any) {
            // If we get "Access is denied", we need admin privileges
            const errorStr = String(error);
            logger.error('Error creating startup task:', errorStr);
            
            if (errorStr.includes('Access is denied') || errorStr.includes('access denied')) {
                logger.info('Need admin privileges to enable autostart, will prompt user');
                
                // Don't save pending operation here, it will be saved when user confirms elevation
                
                // Throw a special error to indicate we need elevation
                throw new Error('NEEDS_ELEVATION');
            }
            
            // For other errors, just pass them through
            throw error;
        }
    } catch (error) {
        if (error instanceof Error && error.message === 'NEEDS_ELEVATION') {
            throw error; // Let the caller handle this special error
        }
        
        logger.error('Failed to enable autostart:', error);
        throw error;
    }
}

/**
 * Disable autostart for the application
 * @returns Promise that resolves when autostart is disabled or rejects with an error
 */
export async function disableAutoStart(): Promise<void> {
    try {
        logger.info('Disabling autostart');
        
        // Check if we're already running as admin
        const isAdmin = await isRunningAsAdmin();
        logger.info(`Current admin status: ${isAdmin}`);
        
        // First check if the task exists
        const taskExists = await isAutoStartEnabled();
        logger.info(`Autostart task exists: ${taskExists}`);
        
        if (!taskExists) {
            logger.info('No autostart task found, nothing to disable');
            return;
        }
        
        // Try to remove the startup task
        try {
            await invoke('remove_startup_task');
            
            logger.info('Autostart disabled successfully');
            return;
        } catch (error: any) {
            // If we get "Access is denied", we need admin privileges
            const errorStr = String(error);
            logger.error('Error removing startup task:', errorStr);
            
            if (errorStr.includes('Access is denied') || errorStr.includes('access denied')) {
                logger.info('Need admin privileges to disable autostart, will prompt user');
                
                // Throw a special error to indicate we need elevation
                throw new Error('NEEDS_ELEVATION');
            }
            
            // For other errors, just pass them through
            throw error;
        }
    } catch (error) {
        if (error instanceof Error && error.message === 'NEEDS_ELEVATION') {
            logger.debug('NEEDS_ELEVATION error caught in disableAutoStart');
            throw error; // Let the caller handle this special error
        }
        
        logger.error('Failed to disable autostart:', error);
        throw error;
    }
}

/**
 * Check if autostart is enabled
 * @returns Promise that resolves to true if autostart is enabled, false otherwise
 */
export async function isAutoStartEnabled(): Promise<boolean> {
    try {
        return await invoke('is_startup_task_enabled');
    } catch (error) {
        logger.error('Failed to check if autostart is enabled:', error);
        return false;
    }
}

/**
 * Check if autostart is configured to run with admin rights
 * @returns Promise that resolves to true if autostart is configured with admin rights, false otherwise
 */
export async function isAutoStartWithAdminRights(): Promise<boolean> {
    try {
        return await invoke('is_startup_task_admin');
    } catch (error) {
        logger.error('Failed to check if autostart is configured with admin rights:', error);
        return false;
    }
}

/**
 * Get the saved autostart preference from local storage
 * @returns The saved preference, or null if not set
 */
export function getSavedAutoStartPreference(): boolean | null {
    const value = localStorage.getItem(AUTOSTART_STORAGE_KEY);
    if (value === null) return null;
    return value === 'true';
}

/**
 * Get the saved admin rights preference from local storage
 * @returns The saved preference, or null if not set
 */
export function getSavedAdminRightsPreference(): boolean | null {
    const value = localStorage.getItem(ADMIN_RIGHTS_STORAGE_KEY);
    if (value === null) return null;
    return value === 'true';
}

/**
 * Synchronize the local storage preference with the actual system state
 * This should be called on app startup to ensure the UI reflects the actual state
 * @returns Promise that resolves to true if autostart is enabled, false otherwise
 */
export async function syncAutoStartPreference(): Promise<boolean> {
    const isEnabled = await isAutoStartEnabled();
    localStorage.setItem(AUTOSTART_STORAGE_KEY, isEnabled ? 'true' : 'false');
    return isEnabled;
}

/**
 * Synchronize the local storage admin rights preference with the actual system state
 * @returns Promise that resolves to true if autostart with admin rights is enabled, false otherwise
 */
export async function syncAdminRightsPreference(): Promise<boolean> {
    const isAdmin = await isAutoStartWithAdminRights();
    localStorage.setItem(ADMIN_RIGHTS_STORAGE_KEY, isAdmin ? 'true' : 'false');
    return isAdmin;
}

/**
 * Restart the application with admin privileges
 * @returns Promise that resolves when the restart is initiated
 */
export async function restartWithAdminRights(): Promise<void> {
    try {
        logger.info('Restarting with admin privileges');
        const result = await invoke('restart_as_admin');
        if (!result) {
            throw new Error('Failed to restart with admin privileges');
        }
    } catch (error) {
        logger.error('Failed to restart with admin privileges:', error);
        throw error;
    }
}

/**
 * Check if the application is running with admin privileges
 * @returns Promise that resolves to true if running as admin, false otherwise
 */
export async function isRunningAsAdmin(): Promise<boolean> {
    try {
        const result = await invoke<boolean>('is_running_as_admin');
        logger.info(`Admin check result: ${result}`);
        return result;
    } catch (error) {
        logger.error('Failed to check if running as admin:', error);
        return false;
    }
}
