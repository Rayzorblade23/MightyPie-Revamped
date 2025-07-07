import {convertRemToPixels} from "$lib/generalUtil.ts";
import {getMousePosition, setMousePosition} from "$lib/mouseFunctions.ts";
import {
    getCurrentWindow,
    LogicalPosition,
    LogicalSize,
    monitorFromPoint,
    PhysicalPosition
} from "@tauri-apps/api/window";
import {PhysicalSize} from "@tauri-apps/api/dpi";

/**
 * Calculates offset positions for pie menu buttons based on their index.
 * @param i - Button index (0-7)
 * @param buttonWidth - Width of the button in rem units
 * @param buttonHeight - Height of the button in rem units
 * @returns Object containing X and Y offset values in pixels
 */
export function calculatePieButtonOffsets(i: number, buttonWidth: number, buttonHeight: number): {
    offsetX: number;
    offsetY: number
} {
    const buttonWidthPx = convertRemToPixels(buttonWidth);
    const buttonHeightPx = convertRemToPixels(buttonHeight);

    const nudgeX = buttonWidthPx / 2 - buttonHeightPx / 2;
    const nudgeY = buttonHeightPx / 2;

    let offsetX = 0;
    let offsetY = 0;

    if (i === 1) {
        offsetX += nudgeX;
        offsetY -= nudgeY;
    } else if (i === 2) {
        offsetX += nudgeX;
        offsetY += 0;
    } else if (i === 3) {
        offsetX += nudgeX;
        offsetY += nudgeY;
    } else if (i === 5) {
        offsetX -= nudgeX;
        offsetY += nudgeY;
    } else if (i === 6) {
        offsetX -= nudgeX;
        offsetY += 0;
    } else if (i === 7) {
        offsetX -= nudgeX;
        offsetY -= nudgeY;
    }
    return {offsetX, offsetY};
}

/**
 * Calculates the absolute position of a pie menu button based on its index and menu parameters.
 * @param index - Button index (0-7)
 * @param numButtons - Total number of buttons in the pie menu
 * @param offsetX - X offset for the button
 * @param offsetY - Y offset for the button
 * @param radius - Radius of the pie menu circle
 * @param width - Total width of the pie menu
 * @param height - Total height of the pie menu
 * @returns Object containing final X and Y coordinates
 */
export function calculatePieButtonPosition(
    index: number,
    numButtons: number,
    offsetX: number,
    offsetY: number,
    radius: number,
    width: number,
    height: number
): { x: number; y: number } {
    const centerX = width / 2;
    const centerY = height / 2;

    const angleInRad = (index / numButtons) * 2 * Math.PI;

    const x = centerX + offsetX + radius * Math.sin(angleInRad);
    const y = centerY - offsetY - radius * Math.cos(angleInRad);
    return {x, y};
}

/**
 * Determines active pie slice and angle based on cursor coordinates relative to pie menu center.
 * @param x - Cursor X coordinate relative to window
 * @param y - Cursor Y coordinate relative to window
 * @param winSize - Window dimensions {width, height}
 * @param deadzoneRadius - Radius of the inner deadzone circle
 * @returns Object containing active slice index and angle in degrees
 */
export function calculatePieSliceFromCoordinates(x: number, y: number, winSize: {
    width: number;
    height: number
}, deadzoneRadius: number): { slice: number; theta: number } {
    const centerX = winSize.width / 2;
    const centerY = winSize.height / 2;

    const dx = x - centerX;
    const dy = y - centerY;

    const numberSlices = 8;
    const sliceAngle = 360 / numberSlices;

    let theta = Math.atan2(dy, dx) * (180 / Math.PI);

    if (theta < 0) {
        theta += 360;
    }

    const r = Math.sqrt(dx * dx + dy * dy);

    if (r < deadzoneRadius) {
        return {slice: -1, theta: theta + sliceAngle / 2};
    }

    type PieSlice = { start: number; end: number };

    const startAngle = 247.5;

    let thetaNormalized = (theta - startAngle) % 360;

    if (thetaNormalized < 0) {
        thetaNormalized += 360;
    }

    const pieSlices: PieSlice[] = Array.from({length: numberSlices}, (_, i) => {

        const start = (i * sliceAngle) % 360;
        const end = (start + sliceAngle) % 360;
        return {start, end};
    });

    for (let i = 0; i < pieSlices.length; i++) {
        if (pieSlices[i].start <= thetaNormalized && thetaNormalized < pieSlices[i].end) {
            return {slice: i, theta: theta + sliceAngle / 2};
        }
    }

    return {slice: 7, theta: theta + sliceAngle / 2};
}

/**
 * Gets the currently active pie slice and mouse angle based on global mouse position.
 * @param deadzoneRadius - Radius of the inner deadzone circle
 * @returns Promise containing active slice index and mouse angle
 */
export async function detectActivePieSlice(deadzoneRadius: number): Promise<{
    slice: number;
    mouseAngle: number
}> {
    try {
        const mousePosition = await getMousePosition();
        const window = getCurrentWindow();
        const winPos: PhysicalPosition = await window.outerPosition();
        const winSize = await window.outerSize();

        const relX = mousePosition.x - winPos.x;
        const relY = mousePosition.y - winPos.y;

        const result = calculatePieSliceFromCoordinates(relX, relY, winSize, deadzoneRadius);

        // if (result.slice === -1) {
        //     console.log("Mouse is inside the inner radius (dead zone).");
        // } else {
        //     console.log(`Mouse is in slice: ${result.slice}`);
        // }

        return {slice: result.slice, mouseAngle: result.theta};
    } catch (error) {
        console.log("Error fetching mouse position:", error);
        return {slice: -1, mouseAngle: 0};
    }
}

/**
 * Clamps window position within monitor bounds to prevent off-screen positioning.
 * @param pos - Window position (logical or physical)
 * @param windowSize - Window dimensions
 * @param monitorSize - Monitor dimensions
 * @param monitorPos - Monitor position
 * @returns Clamped position within monitor bounds
 */
export function clampWindowToBounds(
    pos: LogicalPosition | PhysicalPosition,
    windowSize: LogicalSize | PhysicalSize,
    monitorSize: LogicalSize | PhysicalSize,
    monitorPos: LogicalPosition | PhysicalPosition,
): LogicalPosition | PhysicalPosition {
    const minX = monitorPos.x;
    const minY = monitorPos.y;
    const maxX = monitorPos.x + monitorSize.width - windowSize.width;
    const maxY = monitorPos.y + monitorSize.height - windowSize.height;

    if (pos instanceof LogicalPosition) {
        return new LogicalPosition(Math.min(Math.max(pos.x, minX), maxX), Math.min(Math.max(pos.y, minY), maxY));
    } else {
        return new PhysicalPosition(Math.min(Math.max(pos.x, minX), maxX), Math.min(Math.max(pos.y, minY), maxY));
    }

}

/**
 * Ensures the current window is fully within the bounds of its monitor.
 * If any part is outside, it clamps the position and moves the window.
 */
export async function ensureWindowWithinMonitorBounds(): Promise<void> {
    const window = getCurrentWindow();
    const winPos = await window.outerPosition();
    const winSize = await window.outerSize();
    // Find the monitor at the window's current position (top-left corner)
    const monitor = await monitorFromPoint(winPos.x, winPos.y);
    if (!monitor) {
        console.log("Monitor not found for window position");
        return;
    }
    const clamped = clampWindowToBounds(
        winPos,
        winSize,
        monitor.size,
        monitor.position
    );
    // Only move if needed
    if (clamped.x !== winPos.x || clamped.y !== winPos.y) {
        await window.setPosition(clamped);
    }
}

/**
 * Moves the cursor to the center of the current window.
 */
export async function moveCursorToWindowCenter() {
    const window = getCurrentWindow();
    const outerPos = await window.outerPosition();
    const outerSize = await window.outerSize();
    const centerX = Math.round(outerPos.x + outerSize.width / 2);
    const centerY = Math.round(outerPos.y + outerSize.height / 2);
    await setMousePosition(centerX, centerY);
}

/**
 * Centers the window at the current mouse position and handles monitor scale factors.
 * @param monitorScaleFactor - The current monitor's scale factor
 * @returns Promise containing the new monitor's scale factor
 */
export async function centerWindowAtCursor(monitorScaleFactor: number): Promise<number> {
    const window = getCurrentWindow();
    const outerSize = await window.outerSize();
    const innerSize = await window.innerSize();
    await window.setSize(new PhysicalSize(0, 0));

    const mousePosition = await getMousePosition();
    const monitor = await monitorFromPoint(mousePosition.x, mousePosition.y);
    if (!monitor) {
        console.log("Monitor not found");
        return monitorScaleFactor;
    }

    const newScaleFactor = monitor.scaleFactor;

    const windowScaleFactor = await window.scaleFactor();

    let windowSizeAdj = new LogicalSize(0, 0);

    if (newScaleFactor !== monitorScaleFactor) {
        console.log("Monitor Status: First time on this monitor!");
        windowSizeAdj.width = outerSize.width * (newScaleFactor / windowScaleFactor);
        windowSizeAdj.height = outerSize.height * (newScaleFactor / windowScaleFactor);
    } else {
        console.log("Monitor Status: Been on this monitor before!");
        windowSizeAdj.width = outerSize.width;
        windowSizeAdj.height = outerSize.height;
    }

    let windowPosCentered = new LogicalPosition(mousePosition.x - windowSizeAdj.width / 2, mousePosition.y - windowSizeAdj.height / 2);

    const clamped = clampWindowToBounds(
        windowPosCentered,
        windowSizeAdj,
        monitor.size,
        monitor.position,
    );

    const logicalX = Math.floor(clamped.x / windowScaleFactor);
    const logicalY = Math.floor(clamped.y / windowScaleFactor);

    await window.setPosition(new LogicalPosition(logicalX, logicalY));

    let newSize = new LogicalSize(innerSize.width / windowScaleFactor, innerSize.height / windowScaleFactor);
    await window.setSize(newSize);

    return newScaleFactor;
}
