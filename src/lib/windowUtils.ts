import {LogicalPosition, LogicalSize, monitorFromPoint, Window} from "@tauri-apps/api/window";
import {invoke} from "@tauri-apps/api/core";
import {createLogger} from "$lib/logger";
import {getSettings} from "$lib/data/settingsManager.svelte";

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
 * Gets the combined scale factor for window sizing (uiScale * textScaleFactor).
 * Windows must scale to match the CSS zoom applied to content.
 * @returns Promise<number> The combined scale factor
 */
export async function getCombinedScaleFactor(): Promise<number> {
    const settings = getSettings();
    const uiScale = (settings.uiScale?.value as number) ?? 1.0;
    const textScaleFactor = await getTextScaleFactor();
    
    // Clamp uiScale to reasonable bounds (0.5 to 2.0)
    const clampedUiScale = Math.max(0.5, Math.min(2.0, uiScale));
    
    // Return combined factor to match CSS zoom
    return clampedUiScale * textScaleFactor;
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
    
    // Get combined scale factor (user UI scale * Windows Text Size)
    const combinedScale = await getCombinedScaleFactor();
    
    // Multiply window dimensions by combined scale factor
    const adjustedWidth = desiredWidth * combinedScale;
    const adjustedHeight = desiredHeight * combinedScale;
    
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