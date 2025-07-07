<!-- PieMenu.svelte -->
<script lang="ts">
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
    // Track active button animations
    let activeAnimations = $state<Map<number, { node: HTMLElement, cancel: () => void }>>(new Map());
    let {menuID, pageID, animationKey = 0, opacity = 1}: { 
        menuID: number; 
        pageID: number; 
        animationKey?: number;
        opacity?: number;
    } = $props();

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
            currentMouseEvent = mouseEvents.left_down;
            // Wait for the DOM/reactivity to process the state change
            setTimeout(() => {
                currentMouseEvent = mouseEvents.left_up;
                // Optionally close the menu here
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
                // Set opacity to 0 before hiding the window
                opacity = 0;
                // Cancel all animations before hiding
                cancelAllAnimations();
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

    // Function to cancel all active animations
    function cancelAllAnimations() {
        activeAnimations.forEach((animation) => {
            animation.cancel();
            
            // Immediately reset the node to its final state
            if (animation.node) {
                animation.node.style.transform = '';
                animation.node.style.opacity = '1';
            }
        });
        activeAnimations.clear();
    }

    // Export the cancelAllAnimations function for external use
    // This function is used via component binding in +page.svelte
    // @ts-ignore
    export function cancelAnimations() {
        cancelAllAnimations();
    }

    // Watch for animationKey changes to cancel existing animations
    $effect(() => {
        // When animationKey changes, cancel all active animations
        cancelAllAnimations();
    });

    // Custom action to animate buttons from center
    function animateButtonFromCenter(node: HTMLElement, {
        centerX,
        centerY,
        duration = 150,
        delay = 0
    }: {
        centerX: number,
        centerY: number,
        duration?: number,
        delay?: number
    }) {
        // Get the final position of the node
        const rect = node.getBoundingClientRect();
        const targetX = rect.left + rect.width / 2;
        const targetY = rect.top + rect.height / 2;
        
        // Calculate the distance to animate
        const deltaX = targetX - centerX;
        const deltaY = targetY - centerY;
        
        // Set initial styles - start from center and scaled to 0
        node.style.transform = `translate(${-deltaX}px, ${-deltaY}px) scale(0)`;
        
        // Animation state tracking
        let animationRunning = true;
        let animationTimer: ReturnType<typeof setTimeout> | null = null;
        let animationFrameId: number | null = null;
        
        // Create animation function
        const animate = () => {
            if (!animationRunning) return;
            
            const startTime = performance.now();
            const animateFrame = (currentTime: number) => {
                if (!animationRunning) return;
                
                const elapsed = currentTime - startTime;
                const progress = Math.min(elapsed / duration, 1);
                
                // Cubic easing function
                const easedProgress = 1 - Math.pow(1 - progress, 3);
                
                // Apply transform based on progress
                const currentX = -deltaX * (1 - easedProgress);
                const currentY = -deltaY * (1 - easedProgress);
                node.style.transform = `translate(${currentX}px, ${currentY}px) scale(${easedProgress})`;
                
                // Continue animation if not complete
                if (progress < 1 && animationRunning) {
                    animationFrameId = requestAnimationFrame(animateFrame);
                } else {
                    // Animation complete, reset to final state
                    node.style.transform = '';
                    animationRunning = false;
                    
                    // Remove from active animations
                    const animationId = parseInt(node.dataset.animationId || '0');
                    activeAnimations.delete(animationId);
                }
            };
            
            // Start the animation loop
            animationFrameId = requestAnimationFrame(animateFrame);
        };
        
        // Generate a unique ID for this animation
        const animationId = Date.now() + Math.floor(Math.random() * 1000);
        node.dataset.animationId = animationId.toString();
        
        // Create cancel function
        const cancel = () => {
            animationRunning = false;
            
            if (animationTimer) {
                clearTimeout(animationTimer);
                animationTimer = null;
            }
            
            if (animationFrameId) {
                cancelAnimationFrame(animationFrameId);
                animationFrameId = null;
            }
        };
        
        // Register this animation
        activeAnimations.set(animationId, { node, cancel });
        
        // Start animation after delay
        animationTimer = setTimeout(() => {
            animate();
        }, delay);
        
        // Return destroy function for Svelte action
        return {
            destroy() {
                cancel();
                const animationId = parseInt(node.dataset.animationId || '0');
                activeAnimations.delete(animationId);
            }
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

    {#each buttonPositions as position, i}
        {#key animationKey}
            <div
                class="button-container"
                style="position: absolute; left: {position.x}px; top: {position.y}px;"
                use:animateButtonFromCenter={{
                    centerX: width/2,
                    centerY: height/2,
                    delay: 10,
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
        {/key}
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