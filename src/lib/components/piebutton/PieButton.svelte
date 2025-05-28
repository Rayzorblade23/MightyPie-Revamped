<!--PieButton.svelte-->
<script lang="ts">
    import {getButtonProperties, getButtonType,} from "$lib/data/configHandler.svelte.ts";
    import type {IPieButtonExecuteMessage} from "$lib/data/piebuttonTypes.ts";
    import {ButtonType} from "$lib/data/piebuttonTypes.ts";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE} from "$env/static/public";
    import {composePieButtonClasses, fetchSvgIcon} from './pieButtonUtils';

    interface MouseState {
        hovered: boolean;
        leftDown: boolean;
        leftUp: boolean;
        rightDown: boolean;
        rightUp: boolean;
        middleDown: boolean;
        middleUp: boolean;
    }

    let {menuID, pageID, buttonID, x, y, width, height, mouseState}: {
        menuID: number,
        pageID: number,
        buttonID: number,
        x: number,
        y: number,
        width: number,
        height: number,
        mouseState: MouseState
    } = $props();

    const taskType = $derived(getButtonType(menuID, pageID, buttonID));
    const properties = $derived(getButtonProperties(menuID, pageID, buttonID));

    const buttonTextUpper = $derived.by(() => {
        if (taskType === ButtonType.Disabled) {
            return "Disabled";
        }
        if (
            taskType === ButtonType.ShowAnyWindow &&
            (properties as import('$lib/data/piebuttonTypes').ShowAnyWindowProperties)?.window_handle === -1
        ) {
            return "Unassigned";
        }
        return properties?.button_text_upper ?? `Button ${buttonID + 1}`;
    });

    const buttonTextLower = $derived(properties?.button_text_lower ?? "");

    const finalButtonClasses = $derived.by(() => {
        const isDisabled = taskType === ButtonType.Disabled || (
            taskType === ButtonType.ShowAnyWindow &&
            (properties as import('$lib/data/piebuttonTypes').ShowAnyWindowProperties)?.window_handle === -1
        );
        return composePieButtonClasses({ isDisabled, taskType: taskType ?? "default" });
    });

    // Consolidated mouse click handling
    type MouseButtonType = "left" | "middle" | "right";
    const clickState = {
        left: { prevUp: false, wasDown: false },
        middle: { prevUp: false, wasDown: false },
        right: { prevUp: false, wasDown: false },
    };

    $effect(() => {
        const { leftDown, leftUp, middleDown, middleUp, rightDown, rightUp, hovered } = mouseState;

        const processClick = (button: MouseButtonType, isDown: boolean, isUp: boolean) => {
            const state = clickState[button];
            if (isDown) {
                state.wasDown = true;
            }
            if (isUp && !state.prevUp && state.wasDown && hovered) {
                publishButtonClick(`${button}_up`);
                state.wasDown = false;
            }
            state.prevUp = isUp;
        };

        processClick("left", leftDown, leftUp);
        processClick("middle", middleDown, middleUp);
        processClick("right", rightDown, rightUp);
    });

    function publishButtonClick(clickType: string) {
        if (!properties || !taskType) return;
        const message: IPieButtonExecuteMessage = {
            page_index: pageID, button_index: buttonID, button_type: taskType,
            properties: properties, click_type: clickType
        };
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }

    let svgPromise = $state<Promise<string> | undefined>();
    $effect(() => {
        const iconPath = properties?.icon_path;
        if (iconPath?.endsWith('.svg')) {
            svgPromise = fetchSvgIcon(iconPath);
        } else {
            svgPromise = undefined;
        }
    });
</script>

<style>
    button {
        transition: background 0.15s, border-color 0.15s;
    }
</style>

<!-- Hidden element to declare dynamic classes for Svelte/IDE analyzers -->
<span style="display:none" class="hovered pressed-left pressed-middle pressed-right select-none"></span>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button
            class="{finalButtonClasses}"
            class:hovered={mouseState.hovered && !mouseState.leftDown && !mouseState.middleDown && !mouseState.rightDown}
            class:pressed-left={mouseState.leftDown}
            class:pressed-middle={mouseState.middleDown}
            class:pressed-right={mouseState.rightDown}
            style="width: {width}rem; height: {height}rem;"
    >
        {#if properties?.icon_path}
            {#if properties.icon_path.endsWith('.svg')}
                {#await svgPromise}
                    <div class="h-full flex-shrink-0 flex items-center justify-center p-0.5" style="aspect-ratio: 1/1;">
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
</div>