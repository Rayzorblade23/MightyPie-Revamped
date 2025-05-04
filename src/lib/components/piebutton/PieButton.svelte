<script lang="ts">
    import {getTaskProperties, getTaskType,} from "$lib/components/piebutton/piebuttonConfig.svelte.ts";
    import type {IPieButtonExecuteMessage} from "$lib/components/piebutton/piebuttonTypes.ts";
    import {publishMessage} from "$lib/natsAdapter.ts";
    import {getEnvVar} from "$lib/envHandler.ts";

    interface MouseState {
        hovered: boolean;
        leftDown: boolean;
        leftUp: boolean;
        rightDown: boolean;
        rightUp: boolean;
        middleDown: boolean;
        middleUp: boolean;
    }

    let {menu_index, button_index, x, y, width, height, mouseState}: {
        menu_index: number,
        button_index: number,
        x: number,
        y: number,
        width: number,
        height: number,
        mouseState: MouseState
    } = $props();

    const taskType = $derived(getTaskType(menu_index, button_index));
    const properties = $derived(getTaskProperties(menu_index, button_index));

    const buttonTextUpper = $derived(properties?.button_text_upper ?? `Button ${button_index + 1}`);
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
            console.log(`Left click on Button ${button_index}: Task type is ${taskType}`);
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
            console.log(`Middle click on Button ${button_index}: Task type is ${taskType}`);
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
            console.log(`Right click on Button ${button_index}: Task type is ${taskType}`);
            publishButtonClick("right_up");
            wasRightDown = false;
        }
        prevRightUp = mouseState.rightUp;
    });

    function publishButtonClick(clickType: string) {
        if (!properties || !taskType) return;

        const message: IPieButtonExecuteMessage = {
            menu_index,
            button_index,
            task_type: taskType,
            properties: properties,
            click_type: clickType
        };

        publishMessage<IPieButtonExecuteMessage>(getEnvVar("NATSSUBJECT_PIEBUTTON_EXECUTE"), message);
    }

</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button class="bg-amber-400 flex flex-col items-center justify-center p-0.5"
            style="width: {width}rem; height: {height}rem;"
            class:bg-blue-700={mouseState.hovered}
            class:bg-blue-900={mouseState.leftDown}
            class:bg-green-900={mouseState.middleDown}
            class:bg-red-900={mouseState.rightDown}
    >
        <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
        {#if buttonTextLower}
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-xs leading-tight">{buttonTextLower}</span>
        {/if}
    </button>
</div>