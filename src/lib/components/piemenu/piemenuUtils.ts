import {convertRemToPixels} from "$lib/generalUtil.ts";
import {getMousePosition} from "$lib/mouseFunctions.ts";
import {getCurrentWindow, PhysicalPosition} from "@tauri-apps/api/window";

export function calculateOffsets(i: number, buttonWidth: number, buttonHeight: number): {
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

export function calculateButtonPosition(
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

export function getActivePieSliceFromPosition(x: number, y: number, winSize: {
    width: number;
    height: number
}, deadzoneRadius: number) {
    const centerX = winSize.width / 2;
    const centerY = winSize.height / 2;

    const dx = x - centerX;
    const dy = y - centerY;

    let theta = Math.atan2(dy, dx) * (180 / Math.PI);

    if (theta < 0) {
        theta += 360;
    }

    const r = Math.sqrt(dx * dx + dy * dy);

    if (r < deadzoneRadius) {
        return -1;
    }

    type PieSlice = { start: number; end: number };

    const numberSlices = 8;
    const startAngle = 247.5;

    let thetaNormalized = (theta - startAngle) % 360;

    if (thetaNormalized < 0) {
        thetaNormalized += 360;
    }

    const pieSlices: PieSlice[] = Array.from({length: numberSlices}, (_, i) => {
        const angleSize = 360 / numberSlices;

        const start = (i * angleSize) % 360;
        const end = (start + angleSize) % 360;
        return {start, end};
    });

    for (let i = 0; i < pieSlices.length; i++) {
        if (pieSlices[i].start <= thetaNormalized && thetaNormalized < pieSlices[i].end) {
            return i;
        }
    }

    return 7;
}

export async function getActivePieSliceFromMousePosition(deadzoneRadius: number): Promise<number> {
    try {
        const mousePosition = await getMousePosition();
        const window = getCurrentWindow();
        const winPos: PhysicalPosition = await window.outerPosition();
        const winSize = await window.outerSize();

        const relX = mousePosition.x - winPos.x;
        const relY = mousePosition.y - winPos.y;

        const activeSlice = getActivePieSliceFromPosition(relX, relY, winSize, deadzoneRadius);

        if (activeSlice === -1) {
            console.log("Mouse is inside the inner radius (dead zone).");
        } else {
            console.log(`Mouse is in slice: ${activeSlice}`);
        }

        return activeSlice;
    } catch (error) {
        console.log("Error fetching mouse position:", error);
        return -1;
    }
}