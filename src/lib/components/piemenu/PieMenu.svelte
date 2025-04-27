<script lang="ts">
    import {onDestroy, onMount} from 'svelte';
    import {fly} from 'svelte/transition';
    import {publishMessage, subscribeToTopic} from "$lib/natsAdapter.ts";
    import {goto} from "$app/navigation";
    import {getEnvVar} from "$lib/envHandler.ts";
    import PieButton from "$lib/components/piebutton/PieButton.svelte";
    import {
        calculateButtonPosition,
        calculateOffsets,
        getActivePieSliceAndAngleFromMousePosition
    } from "$lib/components/piemenu/piemenuUtils.ts";
    import {
        type IPiemenuClickMessage,
        type IPiemenuOpenedMessage,
        mouseEvents
    } from "$lib/components/piemenu/piemenuTypes.ts";
    import {loadAndProcessSVG} from "$lib/components/piemenu/indicatorSVGLoader.ts";

    export const menu_index = 0;

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
    let currentMouseEvent = $state<string>('');
    let indicator = $state("");
    let indicatorRotation = $state(0);


    // TODO: Send the clicked slice info to a Trigger Adapter
    subscribeToTopic(getEnvVar("NATSSUBJECT_PIEMENU_CLICK"), message => {
        try {
            const clickMsg: IPiemenuClickMessage = JSON.parse(message);
            currentMouseEvent = clickMsg.click;

            if (clickMsg.click == mouseEvents.left_up) {
                console.log(`Left click in Slice: ${activeSlice}!`);
            } else if (clickMsg.click == mouseEvents.right_up) {
                console.log(`Right click in Slice: ${activeSlice}!`);
                publishMessage<IPiemenuOpenedMessage>(getEnvVar("NATSSUBJECT_PIEMENU_OPENED"), {piemenuOpened: false})
                goto('/');
            } else if (clickMsg.click == mouseEvents.middle_up) {
                console.log(`Middle click in Slice: ${activeSlice}!`);
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    })

    function startAnimationLoop() {
        const update = async () => {
            let {slice, mouseAngle} = await getActivePieSliceAndAngleFromMousePosition(deadzoneRadius);
            activeSlice = slice;
            indicatorRotation = mouseAngle;

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

    onMount(async () => {
        console.log("PieMenu.svelte: onMount hook running");
        publishMessage<IPiemenuOpenedMessage>(getEnvVar("NATSSUBJECT_PIEMENU_OPENED"), {piemenuOpened: true})

        indicator = await loadAndProcessSVG();

        let newButtonPositions: { x: number; y: number }[] = [];

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculateOffsets(i, buttonWidth, buttonHeight);
            const {x, y} = calculateButtonPosition(i, numButtons, offsetX, offsetY, radius, width, height);
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
    <div class="absolute left-1/2 top-1/2 z-10"
         style="transform: translate(-50%, -50%) rotate({indicatorRotation}deg);">
        <img alt="indicator" height="300" src={indicator} width="300"/>
    </div>

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
            <PieButton
                    menu_index={menu_index}
                    button_index={i}
                    x={0}
                    y={0}
                    mouseState={{
                        hovered: activeSlice === i,
                        leftDown: activeSlice === i && currentMouseEvent === mouseEvents.left_down,
                        leftUp: activeSlice === i && currentMouseEvent === mouseEvents.left_up,
                        rightDown: activeSlice === i && currentMouseEvent === mouseEvents.right_down,
                        rightUp: activeSlice === i && currentMouseEvent === mouseEvents.right_up,
                        middleDown: activeSlice === i && currentMouseEvent === mouseEvents.middle_down,
                        middleUp: activeSlice === i && currentMouseEvent === mouseEvents.middle_up
                    }}
            />
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