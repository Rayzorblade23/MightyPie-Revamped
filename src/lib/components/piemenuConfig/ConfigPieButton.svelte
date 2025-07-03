<!-- ConfigPieButton.svelte -->
<script lang="ts">
    import { ButtonType } from '$lib/data/piebuttonTypes.ts';
    import { composePieButtonClasses, fetchSvgIcon } from '../piebutton/pieButtonUtils';

    // Helper type for properties (can be moved to piebuttonTypes.ts if used elsewhere)
    type ButtonPropertiesUnion =
        | import('$lib/data/piebuttonTypes.ts').ShowProgramWindowProperties
        | import('$lib/data/piebuttonTypes.ts').ShowAnyWindowProperties
        | import('$lib/data/piebuttonTypes.ts').CallFunctionProperties
        | import('$lib/data/piebuttonTypes.ts').LaunchProgramProperties;

    // Using $props rune for props declaration
    let {
        x,
        y,
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        onclick,
        active = false
    } = $props<{
        // Layout Props
        x: number;
        y: number;
        width: number; // Width in rem
        height: number; // Height in rem

        // Display Props
        taskType: ButtonType | 'empty';
        properties: ButtonPropertiesUnion | undefined;
        buttonTextUpper?: string;
        buttonTextLower?: string;
        onclick?: (event: MouseEvent) => void;
        active?: boolean;
    }>();

    let svgPromise = $state<Promise<string> | undefined>(undefined);

    $effect(() => {
        if (properties?.icon_path?.endsWith('.svg')) {
            svgPromise = fetchSvgIcon(properties.icon_path);
        } else {
            svgPromise = undefined;
        }
    });

    let hovered = $state(false);
    let pressedLeft = $state(false);

    function handleMouseEnter() { hovered = true; }
    function handleMouseLeave() { hovered = false; pressedLeft = false; }
    function handleMouseDown(e: MouseEvent) { if (e.button === 0) pressedLeft = true; }
    function handleMouseUp(e: MouseEvent) { if (e.button === 0) pressedLeft = false; }

    const finalButtonClasses = () => {
        const isDisabled = taskType === ButtonType.Disabled;
        return composePieButtonClasses({ isDisabled, taskType: taskType ?? "default", allowSelectWhenDisabled: true });
    };
</script>

<!-- The outer div handles absolute positioning using x, y passed as props -->
<!-- These x, y are offsets from the center of the pie menu container -->
<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button
        type="button"
        class={finalButtonClasses()}
        class:hovered={hovered}
        class:pressed-left={pressedLeft}
        class:active-btn={active}
        style="width: {width}rem; height: {height}rem;"
        onclick={onclick}
        onmouseenter={handleMouseEnter}
        onmouseleave={handleMouseLeave}
        onmousedown={handleMouseDown}
        onmouseup={handleMouseUp}
    >
        {#if properties?.icon_path}
            {#if properties.icon_path.endsWith('.svg')}
                {#if svgPromise}
                    {#await svgPromise}
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">
                            ⌛ <!-- Loading indicator -->
                        </div>
                    {:then svgContent}
                        <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">{@html svgContent}</span>
                    {:catch error}
                        <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500" style="aspect-ratio: 1/1;" title={error?.message || 'Error loading SVG'}>
                            ⚠️ <!-- Error icon -->
                        </div>
                    {/await}
                {/if}
            {:else}
                <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1" style="aspect-ratio: 1/1;" />
            {/if}
        {/if}

        <span class="flex flex-col flex-1 pl-1 min-w-0 items-start text-left">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
            {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {buttonTextUpper ? 'text-xs' : 'text-sm'}">{buttonTextLower}</span>
            {/if}
        </span>
    </button>
</div>

<style>
    button {
        transition: background 0.15s, border-color 0.15s;
    }
    /* Make .active-btn trigger the same styles as :hover */
    button.active-btn,
    button.hovered {
        /* No explicit color/background here; rely on existing hover styles (e.g. Tailwind or parent styles) */
    }
</style>