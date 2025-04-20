import {PhysicalPosition} from "@tauri-apps/api/window";
import {invoke} from "@tauri-apps/api/core";

export async function getMousePosition(): Promise<PhysicalPosition> {
    const [x, y] = await invoke<[number, number]>("get_mouse_pos");
    return new PhysicalPosition(x, y);
}