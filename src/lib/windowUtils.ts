import {LogicalPosition, LogicalSize, monitorFromPoint, Window} from "@tauri-apps/api/window";
import {invoke} from "@tauri-apps/api/core";
import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('WindowUtils');

/**
 * Gets the Windows Text Size accessibility setting scale factor.
 * @returns Promise<number> The text scale factor (1.0 = 100%, 1.25 = 125%, etc.)
 */
export async function getTextScaleFactor(): Promise<number> {
    try {
        const scaleFactor = await invoke<number>('get_text_scale_factor');
        return scaleFactor;
    } catch (error) {
        logger.warn('Failed to get text scale factor:', error);
        return 1.0;
    }
}

/**
 * Centers and sizes a Tauri window on the monitor it currently resides on, DPI-aware.
 * Accounts for Windows Text Size accessibility setting by multiplying window dimensions.
 * @param currentWindow The Tauri window instance
 * @param desiredWidth Desired window width (logical units)
 * @param desiredHeight Desired window height (logical units)
 * @returns Promise<void>
 */
export async function centerAndSizeWindowOnMonitor(currentWindow: Window, desiredWidth: number, desiredHeight: number): Promise<void> {
    const windowPos = await currentWindow.outerPosition();
    const monitor = await monitorFromPoint(windowPos.x, windowPos.y);
    if (!monitor) {
        logger.error("Could not get monitor info");
        return;
    }
    const monitorScaleFactor = monitor.scaleFactor;
    
    // Get Windows Text Size setting
    const textScaleFactor = await getTextScaleFactor();
    
    // Multiply window dimensions by text scale factor to compensate for content scaling
    const adjustedWidth = desiredWidth * textScaleFactor;
    const adjustedHeight = desiredHeight * textScaleFactor;
    
    // Convert monitor size/position to logical pixels
    const logicalMonitorWidth = monitor.size.width / monitorScaleFactor;
    const logicalMonitorHeight = monitor.size.height / monitorScaleFactor;
    const logicalMonitorX = monitor.position.x / monitorScaleFactor;
    const logicalMonitorY = monitor.position.y / monitorScaleFactor;
    
    // Clamp settings to logical monitor size
    const settingsWidth = Math.min(adjustedWidth, logicalMonitorWidth);
    const settingsHeight = Math.min(adjustedHeight, logicalMonitorHeight);
    
    // Set window size (logical)
    await currentWindow.setSize(new LogicalSize(settingsWidth, settingsHeight));
    
    // Center window in logical coordinates
    const posX = logicalMonitorX + Math.floor((logicalMonitorWidth - settingsWidth) / 2);
    const posY = logicalMonitorY + Math.floor((logicalMonitorHeight - settingsHeight) / 2);
    await currentWindow.setPosition(new LogicalPosition(posX, posY));
}