<!-- PieMenu.svelte -->
<script lang="ts">
    import {cubicOut} from 'svelte/easing';
    import { fade } from 'svelte/transition';

    import {onDestroy, onMount} from 'svelte';
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {goto} from "$app/navigation";
    import type PieButtonComponent from "$lib/components/piebutton/PieButton.svelte";
    import PieButton from "$lib/components/piebutton/PieButton.svelte";
    import {
        calculatePieButtonOffsets,
        calculatePieButtonPosition,
        detectActivePieSlice
    } from "$lib/components/piemenu/piemenuUtils.ts";
    import {type IPiemenuClickMessage, mouseEvents} from "$lib/data/types/piemenuTypes.ts";
    import {
        PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE,
        PUBLIC_NATSSUBJECT_PIEMENU_CLICK,
        PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED,
        PUBLIC_PIEBUTTON_HEIGHT as BUTTON_HEIGHT,
        PUBLIC_PIEBUTTON_WIDTH as BUTTON_WIDTH,
        PUBLIC_PIEMENU_RADIUS as RADIUS,
        PUBLIC_DEADZONE_RADIUS as DEADZONE_RADIUS,
        PUBLIC_PIEMENU_SIZE_X as PIEMENU_SIZE_X,
        PUBLIC_PIEMENU_SIZE_Y as PIEMENU_SIZE_Y
    } from "$env/static/public";
    import {getIndicatorSVG, getIndicatorRingSVG} from "$lib/components/piemenu/indicatorSVGLoader.svelte.ts";
    import {getSettings} from "$lib/data/settingsManager.svelte.js";
    import {createLogger} from "$lib/logger";

    // Create a logger for this component
    const logger = createLogger('PieMenu');

    const numButtons = 8;
    const radius = Number(RADIUS);
    const buttonWidth = Number(BUTTON_WIDTH);
    const buttonHeight = Number(BUTTON_HEIGHT);
    const width = Number(PIEMENU_SIZE_X);
    const height = Number(PIEMENU_SIZE_Y);
    const deadzoneRadius = Number(DEADZONE_RADIUS);

    let activeSlice = $state(-1);
    let buttonPositions: { x: number; y: number }[] = $state([]);
    let animationFrameId: number | null = null;
    let isMouseTrackingActive = false;
    let currentMouseEvent = $state<string>('');
    let indicatorRotation = $state(0);
    let indicatorReady = $state(false);

    // New state for controlling transitions
    let showButtons = $state(false);

    // Hold refs to child buttons so we can call their execute method on drag-select
    // Make this reactive to silence Svelte's binding_property_non_reactive warning
    let buttonRefs = $state<(PieButtonComponent | null)[]>(Array(numButtons).fill(null));

    let {menuID, pageID, animationKey = 0, opacity = 1, onClose, isMenuOpen = false}: {
        menuID: number;
        pageID: number;
        animationKey?: number;
        opacity?: number;
        onClose: () => void;
        isMenuOpen?: boolean;
    } = $props();

    const indicatorSVG = $derived.by(async () => await getIndicatorSVG());
    const indicatorRingSVG = $derived.by(async () => await getIndicatorRingSVG());

    // Update showButtons whenever animationKey changes
    $effect(() => {
        logger.debug("Animation key changed to:", animationKey);
        // Reset and trigger animation when animationKey changes
        showButtons = false;
        // Reset indicator state to avoid flashing previous rotation
        activeSlice = -1;
        indicatorRotation = 0;
        indicatorReady = false;
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
                logger.info(`Right click. Closing Pie Menu!`);
                onClose();
                return;
            }

            if (clickMsg.click == mouseEvents.left_up) {
                logger.debug(`Left click in Slice: ${activeSlice}!`);
                if (activeSlice === -1) {
                    // Use the selected deadzone function from settings
                    const settings = getSettings();
                    const functionName = settings.pieMenuDeadzoneFunction?.value || "Maximize";
                    const deadzoneMessage = {
                        page_index: pageID,
                        button_index: -1,
                        button_type: 'call_function',
                        properties: {
                            button_text_upper: functionName,
                            button_text_lower: '',
                            icon_path: '',
                        },
                        click_type: 'left_up',
                    };
                    publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, deadzoneMessage);
                    // Restore previous behavior: close after executing deadzone function
                    onClose();
                }
                return;
            }

            if (clickMsg.click == mouseEvents.middle_up) {
                logger.debug(`Middle click in Slice: ${activeSlice}!`);
                if (activeSlice === -1) {
                    // Do not close the Pie Menu on deadzone middle click; just navigate
                    logger.debug("Deadzone middle click! Open quickMenu.");
                    await goto('/quickMenu');
                    return;
                }
                logger.debug("No action for middle click on slice button.");
                return;
            }
        } catch (e) {
            logger.error('Failed to parse message:', e);
        }
    }

    const subscription_button_click = useNatsSubscription(
        PUBLIC_NATSSUBJECT_PIEMENU_CLICK,
        handleButtonClickMessage
    );

    $effect(() => {
        logger.debug("subscription_button_click Status:", subscription_button_click.status);
        if (subscription_button_click.error) {
            logger.error("subscription_button_click Error:", subscription_button_click.error);
        }
    });

    let destroyed = false;

    // Centralized handler for child button close requests
    function onChildCloseRequested() {
        if (destroyed) return;
        onClose();
    }

    // Drag-select functionality
    // Triggered when a shortcut is released outside the deadzone
    const handleShortcutReleasedMessage = async (message: string) => {
        logger.debug('[NATS] Shortcut released message received:');
        logger.debug('↳', message);
        if (destroyed) return;
        // Determine the slice at release time to avoid races with RAF
        let sliceToUse = activeSlice;
        if (sliceToUse === -1) {
            try {
                const { slice } = await detectActivePieSlice(deadzoneRadius);
                sliceToUse = slice;
            } catch (err) {
                logger.error('Failed to detect slice on release:', err);
            }
        }
        if (sliceToUse !== -1) {
            logger.debug(`Drag-select release on slice ${sliceToUse}: calling button execute`);
            const btn = buttonRefs[sliceToUse];
            btn?.executeFromDragSelect?.();
        } else {
            // No slice selected on release -> leave menu open (do not auto-close)
            logger.debug('Release with no active slice; leaving menu open.');
        }
    };

    const subscription_shortcut_released = useNatsSubscription(
        PUBLIC_NATSSUBJECT_SHORTCUT_RELEASED,
        handleShortcutReleasedMessage
    );

    $effect(() => {
        logger.debug("subscription_shortcut_released Status:", subscription_shortcut_released.status);
        if (subscription_shortcut_released.error) {
            logger.error("subscription_shortcut_released Error:", subscription_shortcut_released.error);
        }
    });

    // isMenuOpen is now provided by parent as a prop

    onDestroy(() => {
        destroyed = true;
    });

    // Allow parent to schedule when buttons should unmount (hide) with a delay
    let unmountTimerId: ReturnType<typeof setTimeout> | undefined;

    export function scheduleButtonsUnmount(delayMs: number = 100) {
        if (destroyed) return;
        if (unmountTimerId) clearTimeout(unmountTimerId);
        unmountTimerId = setTimeout(() => {
            if (destroyed) return;
            showButtons = false;
        }, delayMs);
    }

    // Allow parent to cancel any pending unmount (e.g., when cycling/opening again)
    export function cancelButtonsUnmount() {
        if (unmountTimerId) {
            clearTimeout(unmountTimerId);
            unmountTimerId = undefined;
        }
    }

    function startAnimationLoop() {
        logger.debug("Pie Menu opened. Starting Mouse Tracking loop.");
        // Prevent creating multiple RAF loops
        if (isMouseTrackingActive) return;
        if (!showButtons) return;
        isMouseTrackingActive = true;
        const update = async () => {
            if (!isMouseTrackingActive || !showButtons) return;
            try {
                let {slice, mouseAngle} = await detectActivePieSlice(deadzoneRadius);
                if (!isMouseTrackingActive || !showButtons) return; // Stopped while awaiting or hidden
                activeSlice = slice;
                indicatorRotation = mouseAngle;
                // Mark indicator as ready after first computed rotation
                if (!indicatorReady) indicatorReady = true;
            } catch (error) {
                logger.error("Error in animation loop:", error);
            }

            if (!isMouseTrackingActive || !showButtons) return;
            animationFrameId = requestAnimationFrame(update);
        };
        animationFrameId = requestAnimationFrame(update);
    }

    function stopAnimationLoop() {
        isMouseTrackingActive = false;
        if (animationFrameId !== null) {
            cancelAnimationFrame(animationFrameId);
            animationFrameId = null;
            logger.debug("Pie Menu closed. Stopped Mouse Tracking loop.");
        }
    }

    onMount(() => {
        // Ensure fresh state on mount
        activeSlice = -1;
        indicatorRotation = 0;
        indicatorReady = false;
        let newButtonPositions: { x: number; y: number }[] = [];

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculatePieButtonOffsets(i, buttonWidth, buttonHeight);
            const {x, y} = calculatePieButtonPosition(i, numButtons, offsetX, offsetY, radius, width, height);
            newButtonPositions = [...newButtonPositions, {x: x, y: y}];
        }
        buttonPositions = newButtonPositions;

        // Show buttons after a short delay
        setTimeout(() => {
            showButtons = true;
        }, 10);
    });

    onDestroy(() => {
        stopAnimationLoop();
    });

    // Start/stop the animation loop based on button visibility AND actual menu open state
    $effect(() => {
        if (showButtons && isMenuOpen) {
            startAnimationLoop();
        } else {
            stopAnimationLoop();
        }
    });

    // Improved flyAndScale transition - uses buttonIndex to calculate delay
    function flyAndScale(_node: HTMLElement, {
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
        const start = 0.1;
        const duration = 150;
        const baseDelay = 0;
        const settings = getSettings();
        const delayIncrement = settings.pieMenuAnimationDelayIncrement?.value ?? 5;
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
        const flyX = width / 2 - x;
        const flyY = height / 2 - y;

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
    <!-- Center ring overlaying the deadzone (no rotation) -->
    <div class="absolute left-1/2 top-1/2"
         style="transform: translate(-50%, -50%); z-index: 9;">
        {#await indicatorRingSVG}
            <span>Loading...</span>
        {:then ring}
            <img alt="indicator ring" height="300" src={ring} width="300"/>
        {:catch error}
            <span>Error loading ring: {error && error.message ? error.message : error}</span>
        {/await}
    </div>
    {#if showButtons && activeSlice !== -1 && indicatorReady}
        <div class="absolute left-1/2 top-1/2 z-10"
             style="transform: translate(-50%, -50%) rotate({indicatorRotation}deg);"
             transition:fade={{ duration: 120 }}>
            {#await indicatorSVG}
                <span>Loading...</span>
            {:then svg}
                <img alt="indicator" height="300" src={svg} width="300"/>
            {:catch error}
                <span>Error loading indicator: {error && error.message ? error.message : error}</span>
            {/await}
        </div>
    {/if}

    {#if showButtons && buttonPositions.length >= 1}
        {@const button = createPieButtonContainer(0)}
        <div
                class="button-container"
                style="position: absolute; left: {button.x}px; top: {button.y}px;"
                in:flyAndScale={{ x: button.flyX, y: button.flyY, buttonIndex: 0 }}
        >
            <PieButton
                    bind:this={buttonRefs[0]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={0}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[1]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={1}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[2]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={2}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[3]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={3}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[4]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={4}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[5]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={5}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[6]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={6}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
        >
            <PieButton
                    bind:this={buttonRefs[7]}
                    menuID={menuID}
                    pageID={pageID}
                    buttonID={7}
                    x={0}
                    y={0}
                    width={buttonWidth}
                    height={buttonHeight}
                    onClose={onChildCloseRequested}
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
