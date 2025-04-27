<script lang="ts">
    import {getTaskProperties, getTaskType} from "$lib/buttonData/buttonConfig.svelte.ts";

    interface MouseState {
        hovered: boolean;
        leftDown: boolean;
        leftUp: boolean;
        rightDown: boolean;
        rightUp: boolean;
        middleDown: boolean;
        middleUp: boolean;
    }

    let {menu_index, button_index, x, y, mouseState}: {
        menu_index: number,
        button_index: number,
        x: number,
        y: number,
        mouseState: MouseState
    } = $props();

    let taskType = $derived(getTaskType(menu_index, button_index));

    const taskProperties = $derived(getTaskProperties(menu_index, button_index));

    const buttonTextUpper = $derived(taskProperties?.button_text_upper ?? `Button ${button_index + 1}`);
    const buttonTextLower = $derived(taskProperties?.button_text_lower ?? "");


</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button class="bg-amber-400 w-[8.75rem] h-[2.125rem] flex flex-col items-center justify-center"
            class:bg-blue-700={mouseState.hovered}
            class:bg-blue-900={mouseState.leftDown}
            class:bg-green-900={mouseState.middleDown}
            class:bg-red-900={mouseState.rightDown}
    >
        <span>{buttonTextUpper}</span>
        {#if buttonTextLower}
            <span class="text-sm">{buttonTextLower}</span>
        {/if}
    </button>
</div>