<!-- PieMenu.svelte -->
<script lang="ts">
    import {onDestroy, onMount} from 'svelte';
    import {fly} from 'svelte/transition';
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {goto} from "$app/navigation";
    import PieButton from "$lib/components/piebutton/PieButton.svelte";
    import {
        calculatePieButtonOffsets,
        calculatePieButtonPosition,
        detectActivePieSlice
    } from "$lib/components/piemenu/piemenuUtils.ts";
    import {
        type IPiemenuClickMessage,
        type IPiemenuOpenedMessage,
        mouseEvents
    } from "$lib/components/piemenu/piemenuTypes.ts";
    import {
        PUBLIC_NATSSUBJECT_PIEMENU_CLICK,
        PUBLIC_NATSSUBJECT_PIEMENU_OPENED,
        PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED,
        PUBLIC_PIEBUTTON_HEIGHT as BUTTON_HEIGHT,
        PUBLIC_PIEBUTTON_WIDTH as BUTTON_WIDTH,
        PUBLIC_PIEMENU_RADIUS as RADIUS
    } from "$env/static/public";
    import {getCurrentWindow} from "@tauri-apps/api/window";
    import {getIndicatorSVG} from "$lib/components/piemenu/indicatorSVGLoader.svelte.ts";

    const numButtons = 8;
    const radius = Number(RADIUS);
    const buttonWidth = Number(BUTTON_WIDTH);
    const buttonHeight = Number(BUTTON_HEIGHT);
    const width = 600;
    const height = 380;
    const deadzoneRadius = 18;

    let activeSlice = $state(-1);
    let buttonPositions: { x: number; y: number }[] = $state([]);
    let animationFrameId: number | null = null;
    let currentMouseEvent = $state<string>('');
    let indicatorRotation = $state(0);
    let {menuID, pageID}: { menuID: number; pageID: number } = $props();

    const indicatorSVG = $derived.by(async () => await getIndicatorSVG());

    const handleButtonClickMessage = async (message: string) => {
        try {
            const clickMsg: IPiemenuClickMessage = JSON.parse(message);
            currentMouseEvent = clickMsg.click;

            if (clickMsg.click == mouseEvents.right_up) {
                console.log(`Right click in Slice: ${activeSlice}!`);
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false})
                await getCurrentWindow().hide();
                return;
            }

            if (clickMsg.click == mouseEvents.left_up) {
                console.log(`Left click in Slice: ${activeSlice}!`);
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false})
                await getCurrentWindow().hide();
                return;
            }

            if (clickMsg.click == mouseEvents.middle_up) {
                console.log(`Middle click in Slice: ${activeSlice}!`);
                if (activeSlice === -1) {
                    publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false})
                    console.log("Deadzone clicked! Open piemenuConfig.");
                    await goto('/quickMenu');
                    return;
                }
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    }

    const subscription_button_click = useNatsSubscription(
        PUBLIC_NATSSUBJECT_PIEMENU_CLICK,
        handleButtonClickMessage
    );

    $effect(() => {
        console.log("subscription_button_click Status:", subscription_button_click.status);
        if (subscription_button_click.error) {
            console.error("subscription_button_click Error:", subscription_button_click.error);
        }
    });

    const handleShortcutReleasedMessage = async (message: string) => {
        console.log('[NATS] Shortcut released message received:', message);
        if (activeSlice !== -1) {
            currentMouseEvent = mouseEvents.left_down;
            // Wait for the DOM/reactivity to process the state change
            setTimeout(() => {
                currentMouseEvent = mouseEvents.left_up;
                // Optionally close the menu here
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
                getCurrentWindow().hide();
            }, 0);
        }
    };

    const subscription_shortcut_released = useNatsSubscription(
        PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED,
        handleShortcutReleasedMessage
    );

    $effect(() => {
        console.log("subscription_shortcut_released Status:", subscription_shortcut_released.status);
        if (subscription_shortcut_released.error) {
            console.error("subscription_shortcut_released Error:", subscription_shortcut_released.error);
        }
    });

    function startAnimationLoop() {
        const update = async () => {
            let {slice, mouseAngle} = await detectActivePieSlice(deadzoneRadius);
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

    onMount(() => {
        console.log("PieMenu.svelte: onMount hook running");

        let newButtonPositions: { x: number; y: number }[] = [];

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculatePieButtonOffsets(i, buttonWidth, buttonHeight);
            const {x, y} = calculatePieButtonPosition(i, numButtons, offsetX, offsetY, radius, width, height);
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
        {#await indicatorSVG}
            <span>Loading...</span>
        {:then svg}
            <img alt="indicator" height="300" src={svg} width="300"/>
        {:catch error}
            <span>Error loading indicator: {error && error.message ? error.message : error}</span>
        {/await}
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
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={i}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
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
        overflow: hidden;
    }
</style>