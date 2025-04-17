<script lang="ts">
    import '../../app.css';
    import PieMenu from '$lib/components/PieMenu.svelte';
    import {getCurrentWindow, LogicalPosition, PhysicalPosition} from '@tauri-apps/api/window';
    import {pieMenuTargetPos} from '$lib/natsAdapter.svelte.ts';
    import {invoke} from "@tauri-apps/api/core";
    import {onDestroy} from "svelte";


    const deadzoneRadius = 18

    let position: { x: number, y: number };
    let activeSlice = $state(-1);

    $effect(() => {
        // Define a synchronous function that calls the asynchronous logic
        const updatePosition = async () => {
            await centerWindowAtMouse()

            console.log(`This is position ${position.x} ${position.y}`);
        };

        updatePosition();
    });


    async function centerWindowAtMouse() {
        position = pieMenuTargetPos();
        const window = getCurrentWindow();
        const size = await window.outerSize(); // physical pixels


        const centeredX = position.x - size.width / 2;
        const centeredY = position.y - size.height / 2;

        console.log(`Center of window (logical?): ${centeredX}, ${centeredY}`);
        console.log(`Size of window (physical?): ${size.width}, ${size.height}`);

        await window.setPosition(new LogicalPosition(1920 - 222, 1540));
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

        // Check which sector the angle belongs to, based on your ranges
        if (247.5 <= theta && theta < 292.5) {
            return 0; // North
        } else if (292.5 <= theta && theta < 337.5) {
            return 1; // North-East
        } else if ((337.5 <= theta && theta < 360) || (0 <= theta && theta < 22.5)) {
            return 2; // East
        } else if (22.5 <= theta && theta < 67.5) {
            return 3; // South-East
        } else if (67.5 <= theta && theta < 112.5) {
            return 4; // South
        } else if (112.5 <= theta && theta < 157.5) {
            return 5; // South-West
        } else if (157.5 <= theta && theta < 202.5) {
            return 6; // West
        } else if (202.5 <= theta && theta < 247.5) {
            return 7; // North-West
        }

        return -1; // Default case if angle doesn't match
    }


    // The function to get the mouse position via Tauri invoke
    async function handleActivePieSlice() {
        try {
            const window = getCurrentWindow();

            const [x, y] = await invoke<[number, number]>("get_mouse_pos");
            // Log the position clearly indicating it's from the interval
            // console.log(`Interval Update (150ms) - Mouse X: ${x}, Y: ${y}`);

            const winPos: PhysicalPosition = await window.outerPosition();
            const winSize = await window.outerSize();

            // console.log(`Window pos: X: ${winPos.x} - Y: ${winPos.y}`);

            // Relative position within the window
            const relX = x - winPos.x;
            const relY = y - winPos.y;

            // Get the slice using the helper function
            activeSlice = getActivePieSlice(relX, relY, winSize, deadzoneRadius);

            if (activeSlice === -1) {
                console.log("Mouse is inside the inner radius (dead zone).");
            } else {
                console.log(`Mouse is in slice: ${activeSlice}`);
            }

        } catch (error) {
            console.error("Error fetching mouse position:", error);
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
    intervalId = setInterval(handleActivePieSlice, 150); // 150 milliseconds interval

    // --- Setup Cleanup ---
    // Use onDestroy to ensure the interval is cleared when the component is destroyed
    // This prevents memory leaks and errors.
    onDestroy(() => {
        stopInterval(); // Call the cleanup function
    });


</script>

<main>
    <div class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2">
        <div class="absolute z-10 pt-0.5 top-19/20 left-19/20">
            <button
                    class="w-[140px] h-[34px] flex items-center justify-center-safe"
                    data-tauri-drag-region
            >
                <span class="text-white" data-tauri-drag-region>Drag Here</span>
            </button>
        </div>
        <PieMenu slice={activeSlice}/>
    </div>
</main>