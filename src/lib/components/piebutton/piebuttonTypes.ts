export interface IPieButtonExecuteMessage {
    menu_index: number;
    button_index: number;
    task_type: TaskType;
    properties: any;
    click_type: string;
}

export enum TaskType {
    ShowProgramWindow = 'show_program_window',
    ShowAnyWindow = 'show_any_window',
    CallFunction = 'call_function',
    LaunchProgram = 'launch_program',
    Disabled = 'disabled',
}

// Task Interfaces
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

}

export type Task =
    | { task_type: TaskType.ShowProgramWindow; properties: ShowProgramWindowProperties }
    | { task_type: TaskType.ShowAnyWindow; properties: ShowAnyWindowProperties }
    | { task_type: TaskType.CallFunction; properties: CallFunctionProperties }
    | { task_type: TaskType.LaunchProgram; properties: LaunchProgramProperties }
    | { task_type: TaskType.Disabled };


// Represents the structure like: { "0": { "0": TaskData, "1": TaskData }, "1": { ... } }
export type ConfigData = Record<number, Record<number, TaskData>>;

export type TaskData = {
    task_type: string;
    properties?: Record<string, any>; // Properties are optional only for 'disabled' type technically
};

// Button Index -> Typed Task object
export type ButtonMap = Map<number, Task>;
// Menu Index -> ButtonMap
export type MenuConfiguration = Map<number, ButtonMap>;