// src/lib/utils/svgLoader.ts
export async function loadAndProcessSVG() {
    try {
        const response = await fetch("/indicator.svg");
        let svg = await response.text();

        const colors = {
            indicator: '#5a14b7', // Your accent color
            ringFill: '#202020',  // Your ring fill color
            ringStroke: '#303030' // Your ring stroke color
        };

        svg = svg
            .replace("{indicator}", colors.indicator)
            .replace("{ring_fill}", colors.ringFill)
            .replace("{ring_stroke}", colors.ringStroke);

        return `data:image/svg+xml;base64,${btoa(svg)}`;
    } catch (error) {
        console.error('Failed to load or process SVG:', error);
        throw error;
    }
}