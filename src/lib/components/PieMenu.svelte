<script lang="ts">
    import PieButton from './PieButton.svelte';
    import {onDestroy, onMount} from 'svelte';
    import {getCurrentWindow, PhysicalPosition} from "@tauri-apps/api/window";
    import {getMousePosition} from "$lib/mouseFunctions.ts";



    // Configuration
    const numButtons = 8;
    const radius = 150;
    const buttonWidth = 8.75;
    const buttonHeight = 2.125;
    const width = 600;
    const height = 500;

    const deadzoneRadius = 18

    let activeSlice = $state(-1);

    let buttonPositions: { x: number; y: number }[] = $state([]);

    function convertRemToPixels(rem: number) {
        return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
    }

    function calculateOffsets(i: number): { offsetX: number; offsetY: number } {
        const buttonWidthPx = convertRemToPixels(buttonWidth)
        const buttonHeightPx = convertRemToPixels(buttonHeight)

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

    function calculateButtonPosition(
        index: number,
        numButtons: number,
        offsetX: number,
        offsetY: number,
        radius: number):
        {
            x: number;
            y: number
        } {
        const centerX = width / 2;
        const centerY = height / 2;

        const angleInRad = (index / numButtons) * 2 * Math.PI;

        const x = centerX + offsetX + radius * Math.sin(angleInRad);
        const y = centerY - offsetY - radius * Math.cos(angleInRad);
        return {x, y};
    }

    function getActivePieSlice(x: number, y: number, winSize: {
        width: number;
        height: number
    }, deadzoneRadius: number) {
        const centerX = winSize.width / 2;
        const centerY = winSize.height / 2;

        const dx = x - centerX;
        const dy = y - centerY;

        // Compute angle in degrees (0 to 360)
        let theta = Math.atan2(dy, dx) * (180 / Math.PI); // Convert to degrees

        // Normalize angle to be between 0 and 360 degrees
        if (theta < 0) {
            theta += 360;
        }

        // Compute distance from the center
        const r = Math.sqrt(dx * dx + dy * dy);

        // If the distance is smaller than the inner radius, return -1 (dead zone)
        if (r < deadzoneRadius) {
            return -1; // Inside inner radius
        }

        type PieSlice = { start: number; end: number };

        const numberSlices = 8

        const startAngle = 247.5

        let thetaNormalized = (theta - startAngle) % 360

        if (thetaNormalized < 0) {
            thetaNormalized += 360;
        }

        const pieSlices: PieSlice[] = Array.from({length: numberSlices}, (_, i) => {
            const angleSize = 360 / numberSlices

            const start = (i * angleSize) % 360;
            const end = (start + angleSize) % 360;
            return {start, end};
        });

        for (let i = 0; i < pieSlices.length; i++) {
            if (pieSlices[i].start <= thetaNormalized && thetaNormalized < pieSlices[i].end) {
                return i;
            }
        }

        return 7; // Default case is the last slice (North West)
    }


    // The function to get the mouse position via Tauri invoke
    async function handleActivePieSlice() {
        const mousePosition = await getMousePosition();
        try {
            const window = getCurrentWindow();

            const mousePosition = await getMousePosition();
            // Log the position clearly indicating it's from the interval
            // console.log(`Interval Update (150ms) - Mouse X: ${x}, Y: ${y}`);

            const winPos: PhysicalPosition = await window.outerPosition();
            const winSize = await window.outerSize();

            // console.log(`Window pos: X: ${winPos.x} - Y: ${winPos.y}`);

            // Relative position within the window
            const relX = mousePosition.x - winPos.x;
            const relY = mousePosition.y - winPos.y;

            // Get the slice using the helper function
            activeSlice = getActivePieSlice(relX, relY, winSize, deadzoneRadius);

            if (activeSlice === -1) {
                console.log("Mouse is inside the inner radius (dead zone).");
            } else {
                console.log(`Mouse is in slice: ${activeSlice}`);
            }

        } catch (error) {
            console.log("Error fetching mouse position:", error);
            // Optionally stop the interval if there's an error
            stopInterval();
        }
    }

    let intervalId: ReturnType<typeof setInterval> | null = null;

    // Function to stop the interval timer
    function stopInterval() {
        if (intervalId !== null) {
            clearInterval(intervalId);
            intervalId = null; // Clear the ID
            console.log("Stopped mouse position interval.");
        }
    }

    // --- Start the Interval ---
    // When the component mounts, start calling getMousePos every 150ms
    console.log("Starting mouse position interval (150ms)...");
    intervalId = setInterval(handleActivePieSlice, 10); // 150 milliseconds interval

    // --- Setup Cleanup ---
    // Use onDestroy to ensure the interval is cleared when the component is destroyed
    // This prevents memory leaks and errors.
    onDestroy(() => {
        stopInterval(); // Call the cleanup function
    });

    onMount(() => {
        console.log("PieMenu.svelte: onMount hook running");  // Check if onMount is executed

        let newButtonPositions: { x: number; y: number }[] = []; // Create a new array

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculateOffsets(i);
            const {x, y} = calculateButtonPosition(i, numButtons, offsetX, offsetY, radius);
            newButtonPositions = [...newButtonPositions, {x: x, y: y}]; // Add to the new array
        }
        buttonPositions = newButtonPositions; // Assign the new array to buttonPositions

    });
</script>

<div class="relative" style="width: {width}px; height: {height}px;">
    {#each buttonPositions as position, i}
        <PieButton index={i} x={position.x} y={position.y} hovered={activeSlice === i}/>
    {/each}
</div>

<style>
    /* Consider adding a backdrop style here to dim the background */
    .relative {
        /* Center the pie menu */
        display: flex;
        justify-content: center;
        align-items: center;
        backdrop-filter: brightness(50%);
    }
</style>