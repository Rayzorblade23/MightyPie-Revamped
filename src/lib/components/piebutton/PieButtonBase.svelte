<!-- PieButtonBase.svelte -->
<script lang="ts">
    import {ButtonType} from '$lib/data/piebuttonTypes.ts';
    import {composePieButtonClasses, fetchSvgIcon} from './pieButtonUtils';
    import type {PieButtonBaseProps} from '$lib/data/pieButtonSharedTypes';
    import {getSettings} from "$lib/data/settingsHandler.svelte.ts";

    // Base props for pie buttons
    let {
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        // Optional positioning props - some buttons position themselves, others are positioned by parent
        x = undefined,
        y = undefined,
        // Optional styling states - allow external control of hover/pressed states
        active = false,
        forceHovered = false,
        forcePressedLeft = false,
        forcePressedRight = false,
        forcePressedMiddle = false,
        // Optional events - will be forwarded if provided
        onclick = undefined,
        // Children content (replaces slots in Svelte 5)
        buttonContent = undefined,
        allowSelectWhenDisabled = false,
    } = $props<PieButtonBaseProps & {
        x?: number;
        y?: number;
        active?: boolean;
        forceHovered?: boolean;
        forcePressedLeft?: boolean;
        forcePressedRight?: boolean;
        forcePressedMiddle?: boolean;
        onclick?: (event: MouseEvent) => void;
        buttonContent?: any;
        allowSelectWhenDisabled?: boolean;
    }>();

    // Make textSize and subTextSize internal only
    let textSize = 0.775; // default 0.875rem (equivalent to text-sm)
    let subTextSize = 0.65; // default 0.75rem (equivalent to text-xs)

    // SVG icon handling
    let svgPromise = $state<Promise<string> | undefined>(undefined);
    let autoScrollOverflow = $state(false);

    $effect(() => {
        const settings = getSettings();
        const entry = settings.autoScrollOverflow;
        const value = entry?.value ?? entry?.defaultValue ?? entry?.options?.[0];

        if (!entry || !entry.options) {
            autoScrollOverflow = false;
            return;
        }

        // Use the index of the selected value in the options array to determine behavior
        const idx = entry.options.indexOf(value);

        switch (idx) {
            case 0: // First option ("Auto-scroll")
                autoScrollOverflow = true;
                break;
            case 1: // Second option ("Auto-scroll on hover")
                autoScrollOverflow = isHovered;
                break;
            case 2:
                autoScrollOverflow = false;
                break;
            default:
                autoScrollOverflow = false;
        }
    });

    $effect(() => {
        if (properties?.icon_path?.endsWith('.svg')) {
            svgPromise = fetchSvgIcon(properties.icon_path);
        } else {
            svgPromise = undefined;
        }
    });

    // Mouse state management
    let hovered = $state(false);
    let pressedLeft = $state(false);

    function handleMouseEnter() {
        hovered = true;
    }

    function handleMouseLeave() {
        hovered = false;
        pressedLeft = false;
    }

    function handleMouseDown(e: MouseEvent) {
        if (e.button === 0) pressedLeft = true;
    }

    function handleMouseUp(e: MouseEvent) {
        if (e.button === 0) pressedLeft = false;
    }

    // Class handling - simplified to use direct classes to fix styling issues
    const {buttonClass: finalButtonClasses, subtextClass: finalSubtextClass} = $derived.by(() => {
        // Use exact same logic as original PieButton
        const isDisabled = taskType === ButtonType.Disabled || (
            taskType === ButtonType.ShowAnyWindow &&
            (properties as any)?.window_handle === -1
        );
        return composePieButtonClasses({
            isDisabled,
            taskType: taskType ?? "default",
            allowSelectWhenDisabled
        });
    });

    // Compute final hover/pressed states by combining internal and forced states
    const isHovered = $derived(forceHovered || (hovered && !pressedLeft));
    const isPressedLeft = $derived(forcePressedLeft || pressedLeft);

    // Expose internal state to parent components
    const buttonState = $derived({
        hovered,
        pressedLeft
    });

    // Forward to parent
    $effect(() => {
        dispatch('stateChange', buttonState);
    });

    function dispatch(event: string, detail?: any) {
        const customEvent = new CustomEvent(event, {detail});
        buttonElement?.dispatchEvent(customEvent);
    }

    // Make buttonElement reactive
    let buttonElement = $state<HTMLButtonElement | null>(null);

    // Rename prop and add logic for auto-scrolling overflow text
    let upperTextEl = $state<HTMLSpanElement | null>(null);
    let containerEl = $state<HTMLSpanElement | null>(null);

    // Force text element remount whenever text changes to ensure animation restarts
    let textKey = $state(0);

    $effect(() => {
        void buttonTextUpper; // Track the text
        void shouldScroll;    // Track if scrolling is needed
        setTimeout(() => textKey = (textKey + 1) % 1000, 0);
    });

    // Svelte 5: use $derived for reactive DOM-based value
    let shouldScroll = $derived.by(() => {
        if (!autoScrollOverflow) return false;
        if (!upperTextEl || !containerEl) return false;
        void textKey;
        return upperTextEl.scrollWidth > containerEl.offsetWidth;
    });

    // Ensure scrollDistance and scrollDuration are always accurate and consistent
    let scrollDistance = $derived.by(() => {
        if (!autoScrollOverflow || !upperTextEl || !containerEl) return 0;
        void textKey;
        const textWidth = upperTextEl.scrollWidth;
        const containerWidth = containerEl.offsetWidth;
        if (textWidth <= containerWidth) return 0;
        // Round to whole pixels for more consistent animation
        return Math.round(textWidth - containerWidth);
    });
    
    // --- SCROLL DURATION LOGIC ---
    const pxPerSecond = 80;
    const pauseDuration = 2;

    let scrollDuration = $derived.by(() => {
        const distance = scrollDistance;
        if (distance <= 0) return pauseDuration * 2;
        
        // Total duration: scrollTime + pauseDuration*2
        const scrollTime = distance / pxPerSecond;
        return parseFloat(((scrollTime / 0.8) + 2 * pauseDuration).toFixed(2));
    });

    // Add debug to see what's happening
    $effect(() => {
        if (autoScrollOverflow && shouldScroll) {
            console.log({
                scroll: 'active',
                distance: scrollDistance,
                duration: scrollDuration,
                text: buttonTextUpper,
                textWidth: upperTextEl?.scrollWidth,
                containerWidth: containerEl?.offsetWidth
            });
        }
    });
</script>

<style>
    button {
        transition: background-color 0.15s, border-color 0.3s;
    }

    @keyframes scroll-horizontal {
        0%   { margin-left: 0; }
        10%  { margin-left: 0; }
        90%  { margin-left: calc(-1 * var(--scroll-distance, 0px)); }
        100% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }
    }

    @keyframes scroll-horizontal-no-start {
        0%   { margin-left: 0; }
        80%  { margin-left: calc(-1 * var(--scroll-distance, 0px)); }
        100% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }
    }

    .scroll-clip {
        display: flex;
        align-items: center;
        overflow: hidden;
        width: 100%;
        height: 1.2em;
    }
    .scrolling-text,
    .truncate-text {
        display: inline-block;
        vertical-align: middle;
        font-size: inherit;
        white-space: nowrap;
    }
    .scrolling-text {
        animation: scroll-horizontal var(--scroll-duration, 2s) linear infinite;
    }
    .scrolling-text.no-start-pause {
        animation: scroll-horizontal-no-start var(--scroll-duration, 2s) linear infinite;
    }
    .truncate-text {
        text-overflow: ellipsis;
        overflow: hidden;
    }

    .piebutton-flex-parent {
        box-sizing: border-box;
        padding-right: 0.5em;
    }
</style>

<!-- Hidden element to declare dynamic classes for IDE analyzers -->
<span class="hovered pressed-left pressed-middle pressed-right select-none" style="display:none"></span>

{#if x !== undefined && y !== undefined}
    <!-- The outer div handles absolute positioning using x, y passed as props -->
    <div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
        <button
                bind:this={buttonElement}
                type="button"
                class="flex items-center p-0.5 min-w-0 border-solid border rounded-lg {finalButtonClasses}"
                class:hovered={isHovered}
                class:pressed-left={isPressedLeft}
                class:pressed-right={forcePressedRight}
                class:pressed-middle={forcePressedMiddle}
                class:active-btn={active}
                class:select-none={false}
                style="width: {width}rem; height: {height}rem;"
                onclick={onclick}
                onmouseenter={handleMouseEnter}
                onmouseleave={handleMouseLeave}
                onmousedown={handleMouseDown}
                onmouseup={handleMouseUp}
        >
            {#if buttonContent}
                {@render buttonContent()}
            {:else}
                {#if properties?.icon_path}
                    {#if properties.icon_path.endsWith('.svg') && svgPromise}
                        {#await svgPromise}
                            <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5"
                                 style="aspect-ratio: 1/1;">
                                ⌛ <!-- Loading indicator -->
                            </div>
                        {:then svgContent}
                            <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5"
                                  style="aspect-ratio: 1/1;">{@html svgContent}</span>
                        {:catch error}
                            <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500"
                                 style="aspect-ratio: 1/1;"
                                 title="{error instanceof Error ? error.message : 'Error loading SVG'}">
                                ⚠️ <!-- Error indicator -->
                            </div>
                        {/await}
                    {:else if properties.icon_path}
                        <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1"
                             style="aspect-ratio: 1/1;"/>
                    {/if}
                {/if}

                <span class="piebutton-flex-parent flex flex-col flex-1 pl-1 min-w-0 items-start text-left"
                      style="font-size: {textSize}rem;">
                    <span class="scroll-clip" bind:this={containerEl}>
                        {#if autoScrollOverflow && shouldScroll}
                            {#key textKey}
                                <span class="scrolling-text" 
                                      class:no-start-pause={hovered && getSettings().autoScrollOverflow?.value === getSettings().autoScrollOverflow?.options?.[1]}
                                      bind:this={upperTextEl}
                                      style="font-size: inherit; --scroll-distance: {scrollDistance}px; --scroll-duration: {scrollDuration}s;">
                                    {buttonTextUpper}
                                </span>
                            {/key}
                        {:else}
                            <span class="truncate-text" bind:this={upperTextEl}
                                  style="font-size: inherit; text-overflow:ellipsis; overflow:hidden;">{buttonTextUpper}</span>
                        {/if}
                    </span>
                    {#if buttonTextLower}
                        <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {finalSubtextClass}"
                              style="font-size: {buttonTextUpper ? subTextSize : textSize}rem;">{buttonTextLower}</span>
                    {/if}
                </span>
            {/if}
        </button>
    </div>
{:else}
    <!-- Just the button without positioning -->
    <button
            bind:this={buttonElement}
            type="button"
            class="flex items-center p-0.5 min-w-0 border-solid border rounded-lg {finalButtonClasses}"
            class:hovered={isHovered}
            class:pressed-left={isPressedLeft}
            class:pressed-right={forcePressedRight}
            class:pressed-middle={forcePressedMiddle}
            class:active-btn={active}
            class:select-none={false}
            style="width: {width}rem; height: {height}rem;"
            onclick={onclick}
            onmouseenter={handleMouseEnter}
            onmouseleave={handleMouseLeave}
            onmousedown={handleMouseDown}
            onmouseup={handleMouseUp}
    >
        {#if buttonContent}
            {@render buttonContent()}
        {:else}
            {#if properties?.icon_path}
                {#if properties.icon_path.endsWith('.svg') && svgPromise}
                    {#await svgPromise}
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5"
                             style="aspect-ratio: 1/1;">
                            ⌛ <!-- Loading indicator -->
                        </div>
                    {:then svgContent}
                        <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5"
                              style="aspect-ratio: 1/1;">{@html svgContent}</span>
                    {:catch error}
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500"
                             style="aspect-ratio: 1/1;"
                             title="{error instanceof Error ? error.message : 'Error loading SVG'}">
                            ⚠️ <!-- Error indicator -->
                        </div>
                    {/await}
                {:else if properties.icon_path}
                    <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1"
                         style="aspect-ratio: 1/1;"/>
                {/if}
            {/if}

            <span class="piebutton-flex-parent flex flex-col flex-1 pl-1 min-w-0 items-start text-left"
                  style="font-size: {textSize}rem;">
                <span class="scroll-clip" bind:this={containerEl}>
                    {#if autoScrollOverflow && shouldScroll}
                        {#key textKey}
                            <span class="scrolling-text" 
                                  class:no-start-pause={hovered && getSettings().autoScrollOverflow?.value === getSettings().autoScrollOverflow?.options?.[1]}
                                  bind:this={upperTextEl}
                                  style="font-size: inherit; --scroll-distance: {scrollDistance}px; --scroll-duration: {scrollDuration}s;">
                                {buttonTextUpper}
                            </span>
                        {/key}
                    {:else}
                        <span class="truncate-text" bind:this={upperTextEl}
                              style="font-size: inherit; text-overflow:ellipsis; overflow:hidden;">{buttonTextUpper}</span>
                    {/if}
                </span>
                {#if buttonTextLower}
                    <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {finalSubtextClass}"
                          style="font-size: {buttonTextUpper ? subTextSize : textSize}rem;">{buttonTextLower}</span>
                {/if}
            </span>
        {/if}
    </button>
{/if}
