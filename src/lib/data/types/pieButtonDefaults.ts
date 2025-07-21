// piebuttonDefaults.ts

import {
    type Button,
    ButtonType,
    type CallFunctionProperties,
    type DisabledProperties,
    type LaunchProgramProperties,
    type OpenSpecificPieMenuPageProperties,
    type ShowAnyWindowProperties,
    type ShowProgramWindowProperties
} from "$lib/data/types/pieButtonTypes.ts";

const BUTTON_PROPERTIES_MAP = {
    [ButtonType.ShowAnyWindow]: {
        button_type: ButtonType.ShowAnyWindow,
        properties: {
            button_text_upper: "",
            button_text_lower: "",
            icon_path: "",
            window_handle: -1,
            instance: 0 as number,
        } as ShowAnyWindowProperties,
    },
    [ButtonType.ShowProgramWindow]: {
        button_type: ButtonType.ShowProgramWindow,
        properties: {
            button_text_upper: "",
            button_text_lower: "Windows Explorer",
            icon_path: "",
            window_handle: -1,
            instance: 0 as number,
        } as ShowProgramWindowProperties,
    },
    [ButtonType.LaunchProgram]: {
        button_type: ButtonType.LaunchProgram,
        properties: {
            button_text_upper: "Windows Explorer",
            button_text_lower: " - Launch - ",
            icon_path: "",
        } as LaunchProgramProperties,
    },
    [ButtonType.CallFunction]: {
        button_type: ButtonType.CallFunction,
        properties: {
            button_text_upper: "Maximize",
            button_text_lower: "",
            icon_path: "",
        } as CallFunctionProperties,
    },
    [ButtonType.OpenSpecificPieMenuPage]: {
        button_type: ButtonType.OpenSpecificPieMenuPage,
        properties: {
            button_text_upper: "Give your button a name ...",
            button_text_lower: "",
            icon_path: "",
            menu_id: 0,
            page_id: 0,
        } as OpenSpecificPieMenuPageProperties,
    },
    [ButtonType.Disabled]: {
        button_type: ButtonType.Disabled,
        properties: {
            button_text_upper: "",
            button_text_lower: "",
            icon_path: "",
        } as DisabledProperties,
    }
} as const;

export function getDefaultButton(buttonType: ButtonType): Button {
    return BUTTON_PROPERTIES_MAP[buttonType] || BUTTON_PROPERTIES_MAP[ButtonType.Disabled];
}