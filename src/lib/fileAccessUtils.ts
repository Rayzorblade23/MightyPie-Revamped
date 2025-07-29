import {invoke} from "@tauri-apps/api/core";
import {createLogger} from "$lib/logger.ts";

// Create a logger for this component
const logger = createLogger('pathsUtil');

// Cache for icon data URLs
const iconDataUrlCache: Map<string, string> = new Map();

// Cache for SVG content
const svgCache: Map<string, string> = new Map();

/**
 * Gets the parsed content of the buttonFunctions.json file
 * @returns Promise resolving to the parsed JSON data
 */
export async function getButtonFunctions<T = Record<string, any>>(): Promise<T> {
    try {
        // Use the Tauri command to read the file from the backend
        // This avoids frontend permission issues
        const fileContent: string = await invoke('read_button_functions');

        // Parse the JSON data
        return JSON.parse(fileContent) as T;
    } catch (error) {
        logger.error("Failed to read buttonFunctions.json:", error);
        throw new Error(`Failed to read buttonFunctions.json: ${error}`);
    }
}

/**
 * Converts an icon path to a data URL that can be used in img src
 * @param iconPath The path to the icon file
 * @returns Promise resolving to a data URL string
 */
export async function getIconDataUrl(iconPath: string): Promise<string> {
    try {
        // Skip SVG files - they should be handled by the frontend directly
        if (iconPath.endsWith('.svg')) {
            logger.debug(`Skipping SVG file for backend processing: ${iconPath}`);
            return iconPath; // Return the original path for SVGs
        }
        
        // Check if we have this icon in the cache
        if (iconDataUrlCache.has(iconPath)) {
            logger.debug(`Using cached data URL for ${iconPath}`);
            return iconDataUrlCache.get(iconPath)!;
        }
        
        // Use the Tauri command to get the icon as a data URL
        const dataUrl = await invoke<string>('get_icon_data_url', { iconPath });
        
        // Cache the result
        iconDataUrlCache.set(iconPath, dataUrl);
        
        return dataUrl;
    } catch (error) {
        logger.error(`Failed to get icon data URL for ${iconPath}:`, error);
        throw new Error(`Failed to get icon data URL: ${error}`);
    }
}

/**
 * Loads SVG content from a path, using an in-memory cache.
 * @param svgPath The path to the SVG file
 * @returns Promise resolving to the SVG text content
 */
export async function getSvgContent(svgPath: string): Promise<string> {
    if (svgCache.has(svgPath)) {
        logger.debug(`Using cached SVG for ${svgPath}`);
        return svgCache.get(svgPath)!;
    }
    const response = await fetch(svgPath);
    if (!response.ok) throw new Error(`SVG Fetch Error: ${response.status} from ${svgPath}`);
    const text = await response.text();
    svgCache.set(svgPath, text);
    return text;
}

// Optional: Add a method to clear the cache if needed
export function clearIconCache(): void {
    iconDataUrlCache.clear();
    logger.debug('Icon data URL cache cleared');
}