<script lang="ts">
    import '../../app.css';
    import PieMenu from '$lib/components/PieMenu.svelte';
    import {currentMonitor, getCurrentWindow, LogicalPosition, monitorFromPoint} from '@tauri-apps/api/window';
    import {getMousePosition} from "$lib/mouseFunctions.ts";
    import {onMount} from "svelte";
    import {SHORTCUT_DETECTED_EVENT, subscribeToTopic} from "$lib/natsAdapter.ts";


    let mousePosition: { x: number, y: number };

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

        // TODO: Check if changing from currentWindowMonitor to targetMonitor
        const currentWindowMonitor = await currentMonitor();

        if (!monitor) {
            console.log("Monitor not found");
            return;
        }

        const monitorPosition = monitor.position;

        // console.log(`Monitor position: ${monitorPosition.x} ${monitorPosition.y}`);

        const window = getCurrentWindow();
        const size = await window.outerSize(); // physical pixels

        let windowScaleFactor = await window.scaleFactor();
        console.log(`Window Scale Factor): ${windowScaleFactor}`);

        let targetMonitorScaleFactor = monitor.scaleFactor;
        // console.log(`Position Physical: ${mousePosition.x}, ${mousePosition.y}`);

        let windowSizeAdjX = size.width;
        let windowSizeAdjY = size.height;

        console.log(`Size of window (physical?): ${windowSizeAdjX}, ${windowSizeAdjY}`);
        console.log(`Current Scale Factor): ${targetMonitorScaleFactor}`);

        const targetMonitorPosX = monitorPosition ? monitorPosition.x : 0;
        const targetMonitorPosY = monitorPosition ? monitorPosition.y : 0;

        const currentMonMousePosY = mousePosition.y - targetMonitorPosY;
        const currentMonMousePosX = mousePosition.x - targetMonitorPosX;

        const X = Math.floor(targetMonitorPosX / windowScaleFactor + currentMonMousePosX / windowScaleFactor);
        const Y = Math.floor(targetMonitorPosY / windowScaleFactor + currentMonMousePosY / windowScaleFactor);


        const centeredX = Math.floor(X - windowSizeAdjX / 2 * (1 + targetMonitorScaleFactor - windowScaleFactor));
        const centeredY = Math.floor(Y - windowSizeAdjY / 2 * (1 + targetMonitorScaleFactor - windowScaleFactor));


        await window.setPosition(new LogicalPosition(centeredX, centeredY));

        // console.log(`Center of window (logical?): ${centeredX}, ${centeredY}`);
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