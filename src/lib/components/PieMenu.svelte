<script lang="ts">
    import PieButton from './PieButton.svelte';
    import {onDestroy, onMount} from 'svelte';
    import {getCurrentWindow, PhysicalPosition} from "@tauri-apps/api/window";
    import {getMousePosition} from "$lib/mouseFunctions.ts";
    import {fly} from 'svelte/transition';
    import {publishMessage, subscribeToTopic} from "$lib/natsAdapter.ts";
    import {NatsSubjects} from "$lib/natsSubjects.ts";
    import {goto} from "$app/navigation";

    const numButtons = 8;
    const radius = 150;
    const buttonWidth = 8.75;
    const buttonHeight = 2.125;
    const width = 600;
    const height = 500;
    const deadzoneRadius = 18;

    let activeSlice = $state(-1);
    let buttonPositions: { x: number; y: number }[] = $state([]);
    let animationFrameId: number | null = null;

    interface IPiemenuOpenedMessage {
        piemenuOpened: boolean;
    }

    interface IPiemenuClickMessage {
        click: string;
    }

    subscribeToTopic(NatsSubjects.PIEMENU.CLICK, message => {
        try {
            const clickMsg: IPiemenuClickMessage = JSON.parse(message);

            if (clickMsg.click == "left") {
                console.log("Left click from Go!");
            } else if (clickMsg.click == "right") {
                console.log("Right click from Go!");
                publishMessage<IPiemenuOpenedMessage>(NatsSubjects.PIEMENU.OPENED, {piemenuOpened: false})
                goto('/');
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    })

    function convertRemToPixels(rem: number) {
        return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
    }

    function calculateOffsets(i: number): { offsetX: number; offsetY: number } {
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

    function calculateButtonPosition(
        index: number,
        numButtons: number,
        offsetX: number,
        offsetY: number,
        radius: number
    ): { x: number; y: number } {
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

    async function handleActivePieSlice() {
        try {
            const mousePosition = await getMousePosition();
            const window = getCurrentWindow();
            const winPos: PhysicalPosition = await window.outerPosition();
            const winSize = await window.outerSize();

            const relX = mousePosition.x - winPos.x;
            const relY = mousePosition.y - winPos.y;

            activeSlice = getActivePieSlice(relX, relY, winSize, deadzoneRadius);

            if (activeSlice === -1) {
                console.log("Mouse is inside the inner radius (dead zone).");
            } else {
                console.log(`Mouse is in slice: ${activeSlice}`);
            }
        } catch (error) {
            console.log("Error fetching mouse position:", error);
        }
    }

    function startAnimationLoop() {
        const update = async () => {
            await handleActivePieSlice();
            animationFrameId = requestAnimationFrame(update);
        };
        animationFrameId = requestAnimationFrame(update);
    }

    function stopAnimationLoop() {
        if (animationFrameId !== null) {
            cancelAnimationFrame(animationFrameId);
            animationFrameId = null;
            console.log("Stopped animation frame loop.");
        }
    }

    onMount(() => {
        console.log("PieMenu.svelte: onMount hook running");
        publishMessage<IPiemenuOpenedMessage>(NatsSubjects.PIEMENU.OPENED, {piemenuOpened: true})

        let newButtonPositions: { x: number; y: number }[] = [];

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculateOffsets(i);
            const {x, y} = calculateButtonPosition(i, numButtons, offsetX, offsetY, radius);
            newButtonPositions = [...newButtonPositions, {x: x, y: y}];
        }
        buttonPositions = newButtonPositions;

        startAnimationLoop();
    });

    onDestroy(() => {
        stopAnimationLoop();
    });
</script>

<div class="relative" style="width: {width}px; height: {height}px;">
    {#each buttonPositions as position, i}
        <div
                style="position: absolute; left: {position.x}px; top: {position.y}px;"
                transition:fly={{
                x: -(position.x - width/2),
                y: -(position.y - height/2),
                duration: 150,
                opacity: 0

            }}
        >
            <PieButton index={i} x={0} y={0} hovered={activeSlice === i}/>
        </div>
    {/each}
</div>

<style>
    .relative {
        display: flex;
        justify-content: center;
        align-items: center;
        backdrop-filter: brightness(50%);
        position: relative;
    }
</style>