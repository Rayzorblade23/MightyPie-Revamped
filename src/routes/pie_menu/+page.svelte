<script lang="ts">
    import '../../app.css';
    import PieMenu from '$lib/components/PieMenu.svelte';
    import {getCurrentWindow, LogicalPosition, LogicalSize, monitorFromPoint,} from '@tauri-apps/api/window';
    import {getMousePosition} from "$lib/mouseFunctions.ts";
    import {onMount} from "svelte";
    import {SHORTCUT_DETECTED_EVENT, subscribeToTopic} from "$lib/natsAdapter.ts";
    import {PhysicalPosition, PhysicalSize} from "@tauri-apps/api/dpi";


    let mousePosition: { x: number, y: number };

    let _monitorScaleFactor: number = 1;

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


    function clampToBounds(
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

    async function centerWindowAtMouse() {
        const window = getCurrentWindow();
        const outerSize = await window.outerSize();
        const innerSize = await window.innerSize();
        await window.setSize(new PhysicalSize(0, 0));

        mousePosition = await getMousePosition();
        const monitor = await monitorFromPoint(mousePosition.x, mousePosition.y);
        if (!monitor) return console.log("Monitor not found");

        const newScaleFactor = monitor.scaleFactor;

        const windowScaleFactor = await window.scaleFactor();

        let windowSizeAdj = new LogicalSize(0, 0);

        if (newScaleFactor !== _monitorScaleFactor) {
            console.log("Monitor Status: First time on this monitor!");
            windowSizeAdj.width = outerSize.width * (newScaleFactor / windowScaleFactor);
            windowSizeAdj.height = outerSize.height * (newScaleFactor / windowScaleFactor);
        } else {
            console.log("Monitor Status: Been on this monitor before!");
            windowSizeAdj.width = outerSize.width;
            windowSizeAdj.height = outerSize.height;
        }
        _monitorScaleFactor = newScaleFactor;

        let windowPosCentered = new LogicalPosition(mousePosition.x - windowSizeAdj.width / 2, mousePosition.y - windowSizeAdj.height / 2);

        const clamped = clampToBounds(
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
    }


</script>

<main>
    <div class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2">
        <PieMenu/>
    </div>
</main>