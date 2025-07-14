<!-- PieButtonBase.svelte -->
<script lang="ts">
    import { ButtonType } from '$lib/data/piebuttonTypes.ts';
    import { composePieButtonClasses, fetchSvgIcon } from './pieButtonUtils';
    import type { PieButtonBaseProps } from '$lib/data/pieButtonSharedTypes';

    // Base props for pie buttons
    let {
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        allowSelectWhenDisabled = false,
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
        buttonContent = undefined
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
    }>();

    // SVG icon handling
    let svgPromise = $state<Promise<string> | undefined>(undefined);

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

    function handleMouseEnter() { hovered = true; }
    function handleMouseLeave() { hovered = false; pressedLeft = false; }
    function handleMouseDown(e: MouseEvent) { if (e.button === 0) pressedLeft = true; }
    function handleMouseUp(e: MouseEvent) { if (e.button === 0) pressedLeft = false; }

    // Class handling - simplified to use direct classes to fix styling issues
    const finalButtonClasses = $derived.by(() => {
        // Use exact same logic as original PieButton
        const isDisabled = taskType === ButtonType.Disabled || (
            taskType === ButtonType.ShowAnyWindow &&
            (properties as any)?.window_handle === -1
        );
        
        return composePieButtonClasses({ 
            isDisabled, 
            taskType: taskType ?? "default",
            allowSelectWhenDisabled: true
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
        const customEvent = new CustomEvent(event, { detail });
        buttonElement?.dispatchEvent(customEvent);
    }

    // Make buttonElement reactive
    let buttonElement = $state<HTMLButtonElement | null>(null);
</script>

<style>
    button {
        transition: background-color 0.15s, border-color 0.15s;
    }
    
    /* Ensure these classes are recognized by Svelte */
    :global(.hovered),
    :global(.pressed-left),
    :global(.pressed-middle),
    :global(.pressed-right) {
        /* Empty to ensure Svelte recognizes these classes */
    }
</style>

<!-- Hidden element to declare dynamic classes for IDE analyzers -->
<span style="display:none" class="hovered pressed-left pressed-middle pressed-right select-none"></span>

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
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">
                            ⌛ <!-- Loading indicator -->
                        </div>
                    {:then svgContent}
                        <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">{@html svgContent}</span>
                    {:catch error}
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500" style="aspect-ratio: 1/1;" title="{error instanceof Error ? error.message : 'Error loading SVG'}">
                            ⚠️ <!-- Error indicator -->
                        </div>
                    {/await}
                {:else if properties.icon_path}
                    <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1" style="aspect-ratio: 1/1;"/>
                {/if}
            {/if}

            <span class="flex flex-col flex-1 pl-1 min-w-0 items-start text-left">
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
                {#if buttonTextLower}
                    <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {buttonTextUpper ? 'text-xs' : 'text-sm'}">{buttonTextLower}</span>
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
                    <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">
                        ⌛ <!-- Loading indicator -->
                    </div>
                {:then svgContent}
                    <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">{@html svgContent}</span>
                {:catch error}
                    <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500" style="aspect-ratio: 1/1;" title="{error instanceof Error ? error.message : 'Error loading SVG'}">
                        ⚠️ <!-- Error indicator -->
                    </div>
                {/await}
            {:else if properties.icon_path}
                <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1" style="aspect-ratio: 1/1;"/>
            {/if}
        {/if}

        <span class="flex flex-col flex-1 pl-1 min-w-0 items-start text-left">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
            {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {buttonTextUpper ? 'text-xs' : 'text-sm'}">{buttonTextLower}</span>
            {/if}
        </span>
    {/if}
</button>
{/if}
