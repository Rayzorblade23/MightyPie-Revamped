// piebuttonTypes.ts

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
    OpenSpecificPieMenuPage = 'open_page_in_menu',
    OpenResource = 'open_resource',
    Disabled = 'disabled',
}

// Button Interfaces
export interface ShowAnyWindowProperties {
    button_text_upper: string; // window title
    button_text_lower: string; // app name
    icon_path: string;
    window_handle: number;
    instance: number;
}

export interface ShowProgramWindowProperties {
    button_text_upper: string; // window title
    button_text_lower: string; // app name
    icon_path: string;
    window_handle: number;
    instance: number;
}


export interface LaunchProgramProperties {
    button_text_upper: string; // app name
    button_text_lower: string; // " - Launch - "
    icon_path: string;
}

export interface CallFunctionProperties {
    button_text_upper: string; // function name
    button_text_lower: string; // "" Empty string
    icon_path?: string;
}

export interface OpenSpecificPieMenuPageProperties {
    button_text_upper: string; // display name
    button_text_lower: string; // always empty string
    icon_path: string;
    menu_id: number;
    page_id: number;
}

export interface DisabledProperties {
    button_text_upper: string; // always empty string
    button_text_lower: string; // always empty string
    icon_path?: string; // always empty string
}

export interface OpenResourceProperties {
    button_text_upper: string; // display name
    button_text_lower: string; // " - Open Resource - "
    icon_path: string;
    resource_path: string;
}

export type Button =
    | { button_type: ButtonType.ShowProgramWindow; properties: ShowProgramWindowProperties }
    | { button_type: ButtonType.ShowAnyWindow; properties: ShowAnyWindowProperties }
    | { button_type: ButtonType.CallFunction; properties: CallFunctionProperties }
    | { button_type: ButtonType.LaunchProgram; properties: LaunchProgramProperties }
    | { button_type: ButtonType.Disabled; properties: DisabledProperties }
    | { button_type: ButtonType.OpenSpecificPieMenuPage; properties: OpenSpecificPieMenuPageProperties }
    | { button_type: ButtonType.OpenResource; properties: OpenResourceProperties };

export type ButtonPropertiesUnion =
    | ShowProgramWindowProperties
    | ShowAnyWindowProperties
    | CallFunctionProperties
    | LaunchProgramProperties
    | OpenSpecificPieMenuPageProperties
    | DisabledProperties
    | OpenResourceProperties;

// Represents the raw JSON structure: { "menuID": { "pageID": { "buttonID": ButtonData, ... }, ... }, ... }
export type MenuConfigData = Record<string, Record<string, Record<string, ButtonData>>>;

export type ButtonData = {
    button_type: string;
    properties?: Record<string, any>; // Properties are optional only for 'disabled' type technically
};

/**
 * Represents the buttons on a single page.
 * Key: Button Index, Value: Button object.
 */
export type ButtonsOnPageMap = Map<number, Button>;

/**
 * Represents the pages within a single menu.
 * Key: Page Index, Value: Map of buttons on that page.
 */
export type PagesInMenuMap = Map<number, ButtonsOnPageMap>;

/**
 * Represents the overall buttons-only configuration for all menus.
 * Key: Menu Index, Value: Map of pages in that menu.
 */
export type ButtonsConfig = Map<number, PagesInMenuMap>;