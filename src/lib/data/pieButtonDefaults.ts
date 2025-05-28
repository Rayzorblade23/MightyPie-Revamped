// piebuttonDefaults.ts

import {
    type Button,
    ButtonType,
    type CallFunctionProperties,
    type DisabledProperties,
    type LaunchProgramProperties,
    type ShowAnyWindowProperties,
    type ShowProgramWindowProperties
} from "$lib/data/piebuttonTypes.ts";

const BUTTON_PROPERTIES_MAP = {
    [ButtonType.ShowAnyWindow]: {
        button_type: ButtonType.ShowAnyWindow,
        properties: {
            button_text_upper: "",
            button_text_lower: "",
            icon_path: "",
            window_handle: -1,
        } as ShowAnyWindowProperties,
        dropdownFields: []
    },
    [ButtonType.ShowProgramWindow]: {
        button_type: ButtonType.ShowProgramWindow,
        properties: {
            button_text_upper: "",
            button_text_lower: "Windows Explorer",
            icon_path: "",
            window_handle: -1,
        } as ShowProgramWindowProperties,
        dropdownFields: ["button_text_lower"]
    },
    [ButtonType.LaunchProgram]: {
        button_type: ButtonType.LaunchProgram,
        properties: {
            button_text_upper: "Windows Explorer",
            button_text_lower: " - Launch - ",
            icon_path: "",
        } as LaunchProgramProperties,
        dropdownFields: ["button_text_upper"]
    },
    [ButtonType.CallFunction]: {
        button_type: ButtonType.CallFunction,
        properties: {
            button_text_upper: "Maximize",
            button_text_lower: "",
            icon_path: "",
        } as CallFunctionProperties,
        dropdownFields: ["button_text_upper"]
    },
    [ButtonType.Disabled]: {
        button_type: ButtonType.Disabled,
        properties: {
            button_text_upper: "",
            button_text_lower: "",
            icon_path: "",
        } as DisabledProperties,
        dropdownFields: [] // No dropdowns for disabled buttons
    }
} as const;

export function getDefaultButton(buttonType: ButtonType): Button {
    return BUTTON_PROPERTIES_MAP[buttonType] || BUTTON_PROPERTIES_MAP[ButtonType.Disabled];
}

export function getDropdownFields(buttonType: ButtonType): string[] {
    return [...(BUTTON_PROPERTIES_MAP[buttonType]?.dropdownFields || [])];
}