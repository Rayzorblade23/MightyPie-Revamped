<!-- PieButtonBase.svelte -->
<script lang="ts">
    import {ButtonType} from '$lib/data/types/pieButtonTypes.ts';
    import {composePieButtonClasses, fetchSvgIcon} from './pieButtonUtils';
    import type {PieButtonBaseProps} from '$lib/data/types/pieButtonSharedTypes.ts';
    import {getSettings} from "$lib/data/settingsManager.svelte.ts";
    import AutoScrollText from './AutoScrollText.svelte';

    // Base props for pie buttons
    let {
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        x = undefined,
        y = undefined,
        active = false,
        forceHovered,
        forcePressedLeft,
        forcePressedRight,
        forcePressedMiddle,
        onclick = undefined,
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

    // Map enum values to px
    function getBorderWidthFromSetting(setting: string): number {
        switch (setting) {
            case "None":
                return 0;
            case "Thin":
                return 1;
            case "Medium":
                return 1.5;
            case "Thick":
                return 2;
            default:
                return 1.5;
        }
    }

    let borderWidth = $state(1.5);

    // SVG icon handling
    let svgPromise = $state<Promise<string> | undefined>(undefined);
    let autoScrollOverflow = $state(false);

    $effect(() => {
        const settings = getSettings();

        // Handle border thickness
        const thicknessSetting = settings.pieButtonBorderThickness?.value ?? settings.pieButtonBorderThickness?.defaultValue ?? "Medium";
        borderWidth = getBorderWidthFromSetting(thicknessSetting);

        // Handle auto-scroll overflow
        const autoScrollSetting = settings.autoScrollOverflow;
        const value = autoScrollSetting?.value ?? autoScrollSetting?.defaultValue ?? autoScrollSetting?.options?.[0];
        if (!autoScrollSetting || !autoScrollSetting.options) {
            autoScrollOverflow = false;
            return;
        }
        const idx = autoScrollSetting.options.indexOf(value);
        // Always enable for "always" and "hover" modes
        autoScrollOverflow = idx === 0 || idx === 1;
    });

    $effect(() => {
        if (properties?.icon_path?.endsWith('.svg')) {
            svgPromise = fetchSvgIcon(properties.icon_path);
        } else {
            svgPromise = undefined;
        }
    });

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

    // Hybrid hover/pressed logic: use force* props if defined, else fall back to local mouse state
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

    // Fix TS implicit any for isDefined
    function isDefined(val: unknown): boolean {
        return val !== undefined && val !== null;
    }

    const isHovered = $derived(
        isDefined(forceHovered) ? forceHovered : (hovered && !pressedLeft)
    );
    const isPressedLeft = $derived(
        isDefined(forcePressedLeft) ? forcePressedLeft : pressedLeft
    );
    const isPressedRight = $derived(
        isDefined(forcePressedRight) ? forcePressedRight : false
    );
    const isPressedMiddle = $derived(
        isDefined(forcePressedMiddle) ? forcePressedMiddle : false
    );

    // Expose internal state to parent components
    const buttonState = $derived({
        hovered: isHovered,
        pressedLeft: isPressedLeft
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

    // Force text element remount whenever text changes to ensure animation restarts
    let textKey = $state(0);

    $effect(() => {
        void buttonTextUpper; // Track the text
        setTimeout(() => textKey = (textKey + 1) % 1000, 0);
    });

    const instanceNum = $derived.by(() => properties?.instance ?? 0);

</script>

<style>
    button {
        transition: background-color 0.15s, border-color 0.3s;
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
                class="flex items-center p-0.5 min-w-0 border-solid border rounded-lg {finalButtonClasses} overflow-hidden relative"
                class:hovered={isHovered}
                class:pressed-left={isPressedLeft}
                class:pressed-right={isPressedRight}
                class:pressed-middle={isPressedMiddle}
                class:active-btn={active}
                class:select-none={false}
                style="width: {width}rem; height: {height}rem; border-width: {borderWidth}px;"
                onclick={onclick}
                onmouseenter={isDefined(forceHovered) ? undefined : handleMouseEnter}
                onmouseleave={isDefined(forceHovered) ? undefined : handleMouseLeave}
                onmousedown={isDefined(forceHovered) ? undefined : handleMouseDown}
                onmouseup={isDefined(forceHovered) ? undefined : handleMouseUp}
        >

            {#if typeof instanceNum === 'number'
            && instanceNum !== 0
            && taskType !== ButtonType.Disabled
            && properties?.window_handle !== -1}
                <svg class="absolute -top-4 -right-4 z-20 pointer-events-none select-none" width="3.5em" height="3.5em"
                     viewBox="0 0 100 100">
                    <polygon points="100,0 100,100 0,0" fill="currentColor"
                             class="text-purple-500"/>
                    <text x="60" y="45" text-anchor="middle" alignment-baseline="middle" font-size="1.4em"
                          class="fill-white">
                        {instanceNum}
                    </text>
                </svg>
            {/if}
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
                        <img src={properties.icon_path} alt="button icon"
                             class="h-full flex-shrink-0 object-contain p-1"
                             style="aspect-ratio: 1/1;"/>
                    {/if}
                {/if}

                <span class="piebutton-flex-parent flex flex-col flex-1 pl-1 min-w-0 items-start text-left"
                      style="font-size: {textSize}rem;">
                    <AutoScrollText
                            text={buttonTextUpper}
                            enabled={autoScrollOverflow}
                            mode={getSettings().autoScrollOverflow?.value === getSettings().autoScrollOverflow?.options?.[1] ? 'hover' : 'normal'}
                            isButtonHovered={isHovered}
                            className="w-full"
                            style="min-width:0;"
                    />
                    {#if buttonTextLower}
                        <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {finalSubtextClass}"
                              style="font-size: {buttonTextUpper ? subTextSize : textSize}rem; padding-bottom: {borderWidth + 2}px; margin-top: -1px; ">{buttonTextLower}</span>
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
            class="flex items-center p-0.5 min-w-0 border-solid border rounded-lg {finalButtonClasses} overflow-hidden relative"
            class:hovered={isHovered}
            class:pressed-left={isPressedLeft}
            class:pressed-right={isPressedRight}
            class:pressed-middle={isPressedMiddle}
            class:active-btn={active}
            class:select-none={false}
            style="width: {width}rem; height: {height}rem; border-width: {borderWidth}px;"
            onclick={onclick}
            onmouseenter={isDefined(forceHovered) ? undefined : handleMouseEnter}
            onmouseleave={isDefined(forceHovered) ? undefined : handleMouseLeave}
            onmousedown={isDefined(forceHovered) ? undefined : handleMouseDown}
            onmouseup={isDefined(forceHovered) ? undefined : handleMouseUp}
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
                <AutoScrollText
                        text={buttonTextUpper}
                        enabled={autoScrollOverflow}
                        mode={getSettings().autoScrollOverflow?.value === getSettings().autoScrollOverflow?.options?.[1] ? 'hover' : 'normal'}
                        isButtonHovered={isHovered}
                        className="w-full"
                        style="min-width:0;"
                />
                {#if buttonTextLower}
                    <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {finalSubtextClass}"
                          style="font-size: {buttonTextUpper ? subTextSize : textSize}rem; padding-bottom: {borderWidth + 2}px; margin-top: -1px; ">{buttonTextLower}</span>
                {/if}
            </span>
        {/if}
    </button>
{/if}
