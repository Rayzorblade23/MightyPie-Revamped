export interface IPieButtonExecuteMessage {
    page_index: number;
    button_index: number;
    button_type: ButtonType;
    properties: any;
    click_type: string;
}

export enum ButtonType {
    ShowProgramWindow = 'show_program_window',
    ShowAnyWindow = 'show_any_window',
    CallFunction = 'call_function',
    LaunchProgram = 'launch_program',
    Disabled = 'disabled',
}

// Button Interfaces
export interface ShowAnyWindowProperties {
    button_text_upper: string; // window title
    button_text_lower: string; // app name
    icon_path: string;
    window_handle: number;
    exe_path: string;
}

export interface ShowProgramWindowProperties {
    button_text_upper: string; // window title
    button_text_lower: string; // app name
    icon_path: string;
    window_handle: number;
    exe_path: string;
}


export interface LaunchProgramProperties {
    button_text_upper: string; // app name
    button_text_lower: string; // " - Launch - "
    icon_path: string;
    exe_path: string;
}

export interface CallFunctionProperties {
    button_text_upper: string; // function name
    button_text_lower: string; // "" Empty string
    icon_path?: string;
}

export type Button =
    | { button_type: ButtonType.ShowProgramWindow; properties: ShowProgramWindowProperties }
    | { button_type: ButtonType.ShowAnyWindow; properties: ShowAnyWindowProperties }
    | { button_type: ButtonType.CallFunction; properties: CallFunctionProperties }
    | { button_type: ButtonType.LaunchProgram; properties: LaunchProgramProperties }
    | { button_type: ButtonType.Disabled };


// Represents the raw JSON structure: { "profileId": { "menuId": { "buttonId": ButtonData, ... }, ... }, ... }
export type ConfigData = Record<string, Record<string, Record<string, ButtonData>>>;

export type ButtonData = {
    button_type: string;
    properties?: Record<string, any>; // Properties are optional only for 'disabled' type technically
};

// Button Index -> Typed Button object
export type PageConfiguration = Map<number, Button>;

// Menu Index -> Page / PageConfiguration
export type MenuConfiguration = Map<number, PageConfiguration>;

// Profile Index -> MenuConfiguration
export type ProfileConfiguration = Map<number, MenuConfiguration>;