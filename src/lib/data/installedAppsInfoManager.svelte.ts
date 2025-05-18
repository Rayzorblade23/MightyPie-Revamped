// src/lib/data/installedAppsInfoManager.svelte.ts

export interface AppInfo {
    exePath: string; // The resolved executable path
    workingDirectory?: string; // Working directory from LNK
    args?: string; // Command line args from LNK
    uri?: string; // Add this field for store apps
    iconPath?: string; // Path to the icon file
}

export type InstalledAppsMap = Map<string, AppInfo>;

// --- Svelte State and Public API ---

let installedAppsInfo = $state<Map<string, AppInfo>>(new Map());

/**
 * Getter for the global installed apps info.
 * @returns The current map of installed apps, where the keys are AppNames and the values are AppInfo objects.
 */
export function getInstalledAppsInfo(): Map<string, AppInfo> {
    return installedAppsInfo;
}

/**
 * Setter for updating the global installed apps info.
 * @param newAppsInfo - A map of installed apps, where the keys are AppNames and the values are AppInfo objects.
 */
export function updateInstalledAppsInfo(newAppsInfo: Map<string, AppInfo>) {
    installedAppsInfo = newAppsInfo;
}

/**
 * Parses the installed apps info from its raw JSON string format
 * into a structured `Map` where the keys are AppNames and the values are `AppInfo` objects.
 *
 * @param jsonString - The raw JSON string containing the installed apps info.
 * @returns A `Map` where the keys are AppNames and the values are `AppInfo` objects.
 */
export function parseInstalledAppsInfo(jsonString: string): Map<string, AppInfo> {
    try {
        const parsedData = JSON.parse(jsonString);

        // Validate that the parsed data is an object
        if (typeof parsedData !== "object" || parsedData === null) {
            console.error("Parsed data is not a valid object:", parsedData);
            return new Map();
        }

        const appsMap = new Map<string, AppInfo>();

        Object.entries(parsedData).forEach(([appName, appInfo]) => {
            if (typeof appInfo === "object" && appInfo !== null) {
                appsMap.set(appName, appInfo as AppInfo);
            } else {
                console.warn(`Invalid app entry for key "${appName}":`, appInfo);
            }
        });

        return appsMap;
    } catch (error) {
        console.error("Failed to parse installed apps JSON:", error);
        return new Map();
    }
}