import {LogicalPosition, LogicalSize, monitorFromPoint, Window} from "@tauri-apps/api/window";
import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('WindowUtils');

/**
 * Centers and sizes a Tauri window on the monitor it currently resides on, DPI-aware.
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
    // Convert monitor size/position to logical pixels
    const logicalMonitorWidth = monitor.size.width / monitorScaleFactor;
    const logicalMonitorHeight = monitor.size.height / monitorScaleFactor;
    const logicalMonitorX = monitor.position.x / monitorScaleFactor;
    const logicalMonitorY = monitor.position.y / monitorScaleFactor;
    // Clamp settings to logical monitor size
    const settingsWidth = Math.min(desiredWidth, logicalMonitorWidth);
    const settingsHeight = Math.min(desiredHeight, logicalMonitorHeight);
    // Set window size (logical)
    await currentWindow.setSize(new LogicalSize(settingsWidth, settingsHeight));
    // Center window in logical coordinates
    const posX = logicalMonitorX + Math.floor((logicalMonitorWidth - settingsWidth) / 2);
    const posY = logicalMonitorY + Math.floor((logicalMonitorHeight - settingsHeight) / 2);
    await currentWindow.setPosition(new LogicalPosition(posX, posY));
}