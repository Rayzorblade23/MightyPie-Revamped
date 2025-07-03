<!-- QuickMenuPieButton.svelte -->
<script lang="ts">
    import type {IPieButtonExecuteMessage} from '$lib/data/piebuttonTypes.ts';
    import {ButtonType} from '$lib/data/piebuttonTypes.ts';
    import {composePieButtonClasses, fetchSvgIcon} from '../piebutton/pieButtonUtils';
    import {PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE} from "$env/static/public";
    import {publishMessage} from '$lib/natsAdapter.svelte.ts';

    // Helper type for properties (can be moved to piebuttonTypes.ts if used elsewhere)
    type ButtonPropertiesUnion =
        | import('$lib/data/piebuttonTypes.ts').ShowProgramWindowProperties
        | import('$lib/data/piebuttonTypes.ts').ShowAnyWindowProperties
        | import('$lib/data/piebuttonTypes.ts').CallFunctionProperties
        | import('$lib/data/piebuttonTypes.ts').LaunchProgramProperties;

    // Using $props rune for props declaration

    let {
        width,
        height,
        pageID,
        buttonID,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
    } = $props<{
        width: number;
        height: number;
        pageID: number,
        buttonID: number,
        taskType: ButtonType | 'empty';
        properties: ButtonPropertiesUnion | undefined;
        buttonTextUpper?: string;
        buttonTextLower?: string;
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

    function publishButtonClick() {
        if (!properties || !taskType) return;
        const message: IPieButtonExecuteMessage = {
            page_index: pageID, button_index: buttonID, button_type: taskType,
            properties: properties, click_type: "left_up"
        };
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }

    const finalButtonClasses = () => {
        const isDisabled = taskType === ButtonType.Disabled;
        return composePieButtonClasses({isDisabled, taskType: taskType ?? "default", allowSelectWhenDisabled: true});
    };
</script>

<!-- The outer div handles absolute positioning using x, y passed as props -->
<!-- These x, y are offsets from the center of the pie menu container -->
<button
        class={finalButtonClasses()}
        class:hovered={hovered}
        class:pressed-left={pressedLeft}
        onclick={publishButtonClick}
        onmousedown={handleMouseDown}
        onmouseenter={handleMouseEnter}
        onmouseleave={handleMouseLeave}
        onmouseup={handleMouseUp}
        style="width: {width}rem; height: {height}rem;"
        type="button"
>
    {#if properties?.icon_path}
        {#if properties.icon_path.endsWith('.svg')}
            {#if svgPromise}
                {#await svgPromise}
                    <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">
                        ⌛ <!-- Loading indicator -->
                    </div>
                {:then svgContent}
                    <span class="h-full flex-shrink-0 flex items-center justify-center p-0.5"
                          style="aspect-ratio: 1/1;">{@html svgContent}</span>
                {:catch error}
                    <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5 text-red-500"
                         style="aspect-ratio: 1/1;" title={error?.message || 'Error loading SVG'}>
                        ⚠️ <!-- Error icon -->
                    </div>
                {/await}
            {/if}
        {:else}
            <img src={properties.icon_path} alt="button icon" class="h-full flex-shrink-0 object-contain p-1"
                 style="aspect-ratio: 1/1;"/>
        {/if}
    {/if}

    <span class="flex flex-col flex-1 pl-1 min-w-0 items-start text-left">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
        {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis leading-tight {buttonTextUpper ? 'text-xs' : 'text-sm'}">{buttonTextLower}</span>
            {/if}
        </span>
</button>

<style>
    button {
        transition: background 0.15s, border-color 0.15s;
    }
</style>