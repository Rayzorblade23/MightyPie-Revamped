<!-- PieMenuWithTransitions.svelte -->
<script lang="ts">
    import { fly, scale } from 'svelte/transition';
    import { cubicOut } from 'svelte/easing';
    import {onDestroy, onMount} from 'svelte';
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
        PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE,
        PUBLIC_NATSSUBJECT_PIEMENU_CLICK,
        PUBLIC_NATSSUBJECT_PIEMENU_OPENED,
        PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED,
        PUBLIC_PIEBUTTON_HEIGHT as BUTTON_HEIGHT,
        PUBLIC_PIEBUTTON_WIDTH as BUTTON_WIDTH,
        PUBLIC_PIEMENU_RADIUS as RADIUS
    } from "$env/static/public";
    import {getCurrentWindow} from "@tauri-apps/api/window";
    import {getIndicatorSVG} from "$lib/components/piemenu/indicatorSVGLoader.svelte.ts";
    import {getSettings} from "$lib/data/settingsHandler.svelte";

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
    
    // New state for controlling transitions
    let showButtons = $state(false);
    
    let {menuID, pageID, animationKey = 0, opacity = 1}: { 
        menuID: number; 
        pageID: number; 
        animationKey?: number;
        opacity?: number;
    } = $props();

    const indicatorSVG = $derived.by(async () => await getIndicatorSVG());

    // Update showButtons whenever animationKey changes
    $effect(() => {
        console.log("Animation key changed to:", animationKey);
        // Reset and trigger animation when animationKey changes
        showButtons = false;
        // Small delay to ensure DOM updates
        setTimeout(() => {
            showButtons = true;
        }, 0);
    });

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
                if (activeSlice === -1) {
                    // Use the selected deadzone function from settings
                    const settings = getSettings();
                    const fnName = settings.pieMenuDeadzoneFunction?.value || "Maximize";
                    const deadzoneMessage = {
                        page_index: pageID,
                        button_index: -1,
                        button_type: 'call_function',
                        properties: {
                            button_text_upper: fnName,
                            button_text_lower: '',
                            icon_path: '',
                        },
                        click_type: 'left_up',
                    };
                    publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, deadzoneMessage);
                }
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
            // First trigger left_down
            currentMouseEvent = mouseEvents.left_down;
            
            // Wait for the DOM/reactivity to process the left_down state change
            setTimeout(() => {
                // Then trigger left_up which will execute the button via PieButton's effect
                currentMouseEvent = mouseEvents.left_up;
                
                // Delay hiding to ensure the button click is processed
                setTimeout(() => {
                    // Optionally close the menu here
                    publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
                    // Set opacity to 0 before hiding the window
                    opacity = 0;
                    // Hide buttons before hiding window, but after click is processed
                    showButtons = false;
                    getCurrentWindow().hide();
                }, 100); // Add delay to ensure button action is processed
            }, 50); // Increased delay between down and up events
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
            try {
                let {slice, mouseAngle} = await detectActivePieSlice(deadzoneRadius);
                activeSlice = slice;
                indicatorRotation = mouseAngle;
            } catch (error) {
                console.error("Error in animation loop:", error);
            }

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
        
        // Show buttons after a short delay
        setTimeout(() => {
            showButtons = true;
        }, 10);
    });

    onDestroy(() => {
        stopAnimationLoop();
    });
    
    // Expose method to hide buttons (used by the parent component when needed)
    export function cancelAnimations() {
        showButtons = false;
    }

    // Improved flyAndScale transition - uses buttonIndex to calculate delay
    function flyAndScale(node: HTMLElement, { 
        x = 0, 
        y = 0,
        buttonIndex = 0,
        easing = cubicOut
    }: {
        x?: number;
        y?: number;
        buttonIndex?: number;
        easing?: (t: number) => number;
    }) {
        const start = 0.2;
        const duration = 150;
        const baseDelay = 0;
        const delayIncrement = 15;
        const delay = baseDelay + (buttonIndex * delayIncrement);
        
        return {
            duration,
            delay,
            css: (t: number) => {
                const eased = easing(t);
                const flyX = x * (1 - eased);
                const flyY = y * (1 - eased);
                const scaleValue = start + (1 - start) * eased;
                const opacity = eased;
                
                return `transform: translate(${flyX}px, ${flyY}px) scale(${scaleValue}); opacity: ${opacity};`;
            }
        };
    }
    
    // Helper component factory function to reduce repetition
    function createPieButtonContainer(buttonIndex: number) {
        // Since we always check buttonPositions.length and showButtons before calling this
        // function, we can assume these values are valid and remove the null return
        const x = buttonPositions[buttonIndex].x;
        const y = buttonPositions[buttonIndex].y;
        const flyX = width/2 - x;
        const flyY = height/2 - y;
        
        return {
            x, 
            y, 
            flyX, 
            flyY,
            buttonIndex
        };
    }
</script>

<div class="relative" style="width: {width}px; height: {height}px; opacity: {opacity};">
    <div
        class="deadzone"
        class:active={activeSlice === -1 && (currentMouseEvent === mouseEvents.left_down || currentMouseEvent === mouseEvents.middle_down)}
        class:hovered={activeSlice === -1 && !(currentMouseEvent === mouseEvents.left_down || currentMouseEvent === mouseEvents.middle_down)}
        style="position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%); width: {deadzoneRadius * 2}px; height: {deadzoneRadius * 2}px; border-radius: 50%; z-index: 5;"
    ></div>
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

    {#if showButtons && buttonPositions.length >= 1}
        {@const button = createPieButtonContainer(0)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 0 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={0}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 0,
                    leftDown: activeSlice === 0 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 0 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 0 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 0 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 0 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 0 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 2}
        {@const button = createPieButtonContainer(1)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 1 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={1}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 1,
                    leftDown: activeSlice === 1 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 1 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 1 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 1 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 1 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 1 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 3}
        {@const button = createPieButtonContainer(2)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 2 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={2}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 2,
                    leftDown: activeSlice === 2 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 2 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 2 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 2 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 2 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 2 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 4}
        {@const button = createPieButtonContainer(3)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 3 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={3}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 3,
                    leftDown: activeSlice === 3 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 3 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 3 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 3 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 3 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 3 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 5}
        {@const button = createPieButtonContainer(4)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 4 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={4}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 4,
                    leftDown: activeSlice === 4 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 4 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 4 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 4 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 4 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 4 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 6}
        {@const button = createPieButtonContainer(5)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 5 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={5}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 5,
                    leftDown: activeSlice === 5 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 5 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 5 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 5 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 5 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 5 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 7}
        {@const button = createPieButtonContainer(6)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 6 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={6}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 6,
                    leftDown: activeSlice === 6 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 6 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 6 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 6 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 6 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 6 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
    
    {#if showButtons && buttonPositions.length >= 8}
        {@const button = createPieButtonContainer(7)}
        <div 
            class="button-container"
            style="position: absolute; left: {button.x}px; top: {button.y}px;"
            in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 7 }}
            out:scale={{ start: 1, opacity: 0, duration: 150 }}
        >
            <PieButton
                menuID={menuID}
                pageID={pageID}
                buttonID={7}
                x={0}
                y={0}
                width={buttonWidth}
                height={buttonHeight}
                mouseState={{
                    hovered: activeSlice === 7,
                    leftDown: activeSlice === 7 && currentMouseEvent === mouseEvents.left_down,
                    leftUp: activeSlice === 7 && currentMouseEvent === mouseEvents.left_up,
                    rightDown: activeSlice === 7 && currentMouseEvent === mouseEvents.right_down,
                    rightUp: activeSlice === 7 && currentMouseEvent === mouseEvents.right_up,
                    middleDown: activeSlice === 7 && currentMouseEvent === mouseEvents.middle_down,
                    middleUp: activeSlice === 7 && currentMouseEvent === mouseEvents.middle_up
                }}
            />
        </div>
    {/if}
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

    .deadzone {
        background: none;
        transition: background 0.15s;
        pointer-events: auto;
        opacity: 0;
    }

    .deadzone.hovered {
        background: var(--color-button-pressed-middle-bg);
        opacity: 0.3;
    }

    .deadzone.active {
        background: var(--color-button-pressed-middle-bg);
        opacity: 0.9;
    }
</style>
