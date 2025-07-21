import {getSettings} from "$lib/data/settingsManager.svelte.ts";

/**
 * Loads and processes the indicator SVG using current settings.
 * Always call this from a Svelte file/rune for reactivity.
 */
export async function getIndicatorSVG() {
    const settings = getSettings();
    const response = await fetch("/indicator_arrow_1.svg");
    let svg = await response.text();
    const colors = {
        indicator: settings.colorIndicator?.value ?? "#5f3c8e",
        ringFill: settings.colorRingFill?.value ?? "#202020",
        ringStroke: settings.colorRingStroke?.value ?? "#303030"
    };
    svg = svg
        .replace("{indicator}", colors.indicator)
        .replace("{ring_fill}", colors.ringFill)
        .replace("{ring_stroke}", colors.ringStroke);
    return `data:image/svg+xml;base64,${btoa(svg)}`;
}
