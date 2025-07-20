interface MouseEvent {
    left_down: string;
    left_up: string;
    right_down: string;
    right_up: string;
    middle_down: string;
    middle_up: string;
}

export const mouseEvents: MouseEvent = {
    left_down: "left_down",
    left_up: "left_up",
    right_down: "right_down",
    right_up: "right_up",
    middle_down: "middle_down",
    middle_up: "middle_up",
};

export interface IPiemenuOpenedMessage {
    piemenuOpened: boolean;
}

export interface IPiemenuClickMessage {
    click: string;
}

export interface IShortcutPressedMessage {
    shortcutPressed: number;
    mouseX: number; // not used for now
    mouseY: number; // not used for now
    openSpecificPage: boolean;
    pageID: number;
}