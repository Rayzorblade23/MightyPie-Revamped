<!--PieButton.svelte-->
<script lang="ts">
    import {getButtonProperties, getButtonType,} from "$lib/data/configHandler.svelte.ts";
    import type {IPieButtonExecuteMessage} from "$lib/data/piebuttonTypes.ts";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE} from "$env/static/public";

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

    const buttonTextUpper = $derived(properties?.button_text_upper ?? `Button ${buttonID + 1}`);
    const buttonTextLower = $derived(properties?.button_text_lower ?? "");

    let prevLeftUp = false;
    let prevMiddleUp = false;
    let prevRightUp = false;

    let wasLeftDown = false;
    let wasMiddleDown = false;
    let wasRightDown = false;

    $effect(() => {
        if (mouseState.leftDown) {
            wasLeftDown = true;
        }

        if (mouseState.leftUp && !prevLeftUp && wasLeftDown && mouseState.hovered) {
            console.log(`Left click on Button ${buttonID}: Button type is ${taskType}`);
            publishButtonClick("left_up");
            wasLeftDown = false;
        }
        prevLeftUp = mouseState.leftUp;
    });

    $effect(() => {
        if (mouseState.middleDown) {
            wasMiddleDown = true;
        }

        if (mouseState.middleUp && !prevMiddleUp && wasMiddleDown && mouseState.hovered) {
            console.log(`Middle click on Button ${buttonID}: Button type is ${taskType}`);
            publishButtonClick("middle_up");
            wasMiddleDown = false;
        }
        prevMiddleUp = mouseState.middleUp;
    });

    $effect(() => {
        if (mouseState.rightDown) {
            wasRightDown = true;
        }

        if (mouseState.rightUp && !prevRightUp && wasRightDown && mouseState.hovered) {
            console.log(`Right click on Button ${buttonID}: Button type is ${taskType}`);
            publishButtonClick("right_up");
            wasRightDown = false;
        }
        prevRightUp = mouseState.rightUp;
    });

    function publishButtonClick(clickType: string) {
        if (!properties || !taskType) return;

        const message: IPieButtonExecuteMessage = {
            page_index: pageID,
            button_index: buttonID,
            button_type: taskType,
            properties: properties,
            click_type: clickType
        };
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }

    let svgPromise = $state();

    $effect(() => {
        if (properties?.icon_path?.endsWith('.svg')) {
            svgPromise = fetch(properties.icon_path)
                .then(r => r.text())
                .then(text => text.replace(/<svg /, '<svg class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1" '));
        }
    });
</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button class="bg-amber-400 flex items-center p-0.5 min-w-0"
            style="width: {width}rem; height: {height}rem;"
            class:bg-blue-700={mouseState.hovered}
            class:bg-blue-900={mouseState.leftDown}
            class:bg-green-900={mouseState.middleDown}
            class:bg-red-900={mouseState.rightDown}
    >
        {#if properties?.icon_path}
            {#if properties.icon_path.endsWith('.svg')}
                {#await svgPromise}
                    <div class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1">⌛</div>
                {:then svgContent}
                    {@html svgContent}
                {/await}
            {:else}
                <img src={properties.icon_path} alt="button icon" class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1" />
            {/if}
        {/if}

        <div class="flex flex-col flex-1 min-w-0">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
            {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-xs leading-tight">{buttonTextLower}</span>
            {/if}
        </div>
    </button>
</div>