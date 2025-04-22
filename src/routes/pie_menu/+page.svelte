<script lang="ts">
    import '../../app.css';
    import PieMenu from '$lib/components/PieMenu.svelte';
    import {getCurrentWindow, LogicalPosition, monitorFromPoint} from '@tauri-apps/api/window';
    import {getMousePosition} from "$lib/mouseFunctions.ts";
    import {onMount} from "svelte";
    import {SHORTCUT_DETECTED_EVENT, subscribeToTopic} from "$lib/natsAdapter.ts";


    let mousePosition: { x: number, y: number };

    let monitorScaleFactor: number = 1;

    interface IShortcutDetectedMessage {
        shortcutDetected: number;
    }

    subscribeToTopic(SHORTCUT_DETECTED_EVENT, message => {
        try {
            const shortcutDetectedMsg: IShortcutDetectedMessage = JSON.parse(message);

            if (shortcutDetectedMsg.shortcutDetected == 1) {
                centerWindowAtMouse();
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    })

    onMount(() => {
        centerWindowAtMouse();
        console.log("Pie Menu opened!");
    });


    async function centerWindowAtMouse() {
        mousePosition = await getMousePosition();

        const monitor = await monitorFromPoint(mousePosition.x, mousePosition.y);

        if (!monitor) return console.log("Monitor not found");

        const targetMonitorScaleFactor = monitor.scaleFactor;

        // Get window properties
        const window = getCurrentWindow();
        const size = await window.outerSize();
        const windowScaleFactor = await window.scaleFactor();

        let windowSizeAdjX = 0;
        let windowSizeAdjY = 0;

        if (targetMonitorScaleFactor != monitorScaleFactor) {
            console.log("Monitor Status: First time on this monitor!");
            monitorScaleFactor = targetMonitorScaleFactor;
            windowSizeAdjX = size.width * (targetMonitorScaleFactor / windowScaleFactor);
            windowSizeAdjY = size.height * (targetMonitorScaleFactor / windowScaleFactor);
        } else {
            console.log("Monitor Status: Been on this monitor before!");
            windowSizeAdjX = size.width;
            windowSizeAdjY = size.height;
        }

        const windowPosCenteredX = mousePosition.x - windowSizeAdjX / 2
        const windowPosCenteredY = mousePosition.y - windowSizeAdjY / 2

        // Make logical and pixel perfect
        const centeredX = Math.floor(windowPosCenteredX / windowScaleFactor);
        const centeredY = Math.floor(windowPosCenteredY / windowScaleFactor);

        console.log("Target Monitor Scale Factor: ", targetMonitorScaleFactor);
        console.log("Window Scale Factor:", windowScaleFactor)

        await window.setPosition(new LogicalPosition(centeredX, centeredY));

        console.log(`Center of window (logical?): ${centeredX}, ${centeredY}`);
    }


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
        <PieMenu/>
    </div>
</main>