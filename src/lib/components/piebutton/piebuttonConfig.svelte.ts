import {
    type ButtonMap,
    type MenuConfiguration,
    type RawNestedConfigData,
    type Task,
    TaskType
} from "$lib/components/piebutton/piebuttonTypes.ts";

// TODO: Replace with read-in config json
const rawNestedMenuConfigData: RawNestedConfigData = {
    "0": { // Menu Index 0
        "0": {
            "task_type": "show_program_window",
            "properties": {
                "button_text_upper": "Youtube",
                "button_text_lower": "Vivaldi",
                "app_icon_path": "",
                "window_handle": "",
                "exe_path": "",
            }
        },
        "1": {
            "task_type": "show_program_window",
            "properties": {
                "button_text_upper": "Something else",
                "button_text_lower": "YO",
                "app_icon_path": "",
                "window_handle": "",
                "exe_path": "",
            }
        },
        "3": {
            "task_type": "show_program_window",
            "properties": {
                "button_text_upper": "Function",
                "button_text_lower": "",
                "app_icon_path": "",
                "window_handle": "",
                "exe_path": "",
            }
        },
    }
};


// --- 3. Parsing Function (Handles Nested Input) ---

function parseNestedRawConfig(nestedRawData: RawNestedConfigData): MenuConfiguration {
    const menuConfig: MenuConfiguration = new Map();

    // Iterate through Menu Indices (string keys "0", "1", ...)
    for (const menuIndexStr in nestedRawData) {
        if (!Object.prototype.hasOwnProperty.call(nestedRawData, menuIndexStr)) {
            continue;
        }

        const menuIndex = parseInt(menuIndexStr, 10);
        if (isNaN(menuIndex)) {
            console.warn(`Skipping invalid menu index: ${menuIndexStr}`);
            continue;
        }

        const rawButtonMap = nestedRawData[menuIndexStr];
        const buttonMap: ButtonMap = new Map(); // Create the inner Map for this menu

        // Iterate through Button Indices (string keys "0", "1", ..., "8", ...)
        for (const buttonIndexStr in rawButtonMap) {
            if (!Object.prototype.hasOwnProperty.call(rawButtonMap, buttonIndexStr)) {
                continue;
            }

            const buttonIndex = parseInt(buttonIndexStr, 10);
            if (isNaN(buttonIndex)) {
                console.warn(`Skipping invalid button index: ${buttonIndexStr} in menu ${menuIndex}`);
                continue;
            }

            const rawTaskData = rawButtonMap[buttonIndexStr];
            const taskType = rawTaskData.task_type as TaskType; // Cast
            const properties = rawTaskData.properties; // Might be undefined

            // Validate task_type
            if (!Object.values(TaskType).includes(taskType)) {
                console.warn(`Skipping button index ${buttonIndex} in menu ${menuIndex}: Unknown task_type "${taskType}"`);
                continue;
            }

            // Create the typed Task object
            let task: Task;
            if (taskType === TaskType.Disabled) {
                task = {task_type: TaskType.Disabled};
            } else if (properties) {
                // Add more robust validation/casting here if needed (Zod recommended)
                task = {
                    task_type: taskType, // TS knows this isn't Disabled here
                    properties: properties as any // Cast needed without validation lib
                };
            } else {
                console.warn(`Skipping button index ${buttonIndex} in menu ${menuIndex}: Task type "${taskType}" requires properties, but none found.`);
                continue; // Skip if properties are missing for types that need them
            }

            // Add the typed Task to the inner ButtonMap
            buttonMap.set(buttonIndex, task);
        }

        // Add the populated ButtonMap to the main MenuConfiguration Map
        if (buttonMap.size > 0) { // Only add menus that actually have buttons
            menuConfig.set(menuIndex, buttonMap);
        }
    }

    return menuConfig;
}


const initialConfig = parseNestedRawConfig(rawNestedMenuConfigData);

export const menuConfiguration = $state<MenuConfiguration>(initialConfig);

/**
 * Gets the task type for a specific button in a menu
 * @returns TaskType or undefined if the task doesn't exist
 */
export function getTaskType(menuIndex: number, buttonIndex: number): TaskType | undefined {
    const menu = menuConfiguration.get(menuIndex);
    const task = menu?.get(buttonIndex);
    return task?.task_type;
}

/**
 * Gets the properties for a specific button in a menu
 * @returns Task properties or undefined if the task doesn't exist or is disabled
 */
export function getTaskProperties<T extends Exclude<Task, { task_type: TaskType.Disabled }>>(
    menuIndex: number,
    buttonIndex: number
): T["properties"] | undefined {
    const menu = menuConfiguration.get(menuIndex);
    const task = menu?.get(buttonIndex);

    if (!task || task.task_type === TaskType.Disabled) {
        return undefined;
    }

    return (task as T).properties;
}