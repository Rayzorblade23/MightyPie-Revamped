<script lang="ts">


    import {getTaskProperties} from "$lib/buttonData/buttonConfig.svelte.ts";

    let {index, x, y, hovered}: { index: number, x: number, y: number, hovered: boolean } = $props()

    const taskProperties = $derived(getTaskProperties(0, index));

    const buttonTextUpper = $derived(taskProperties?.button_text_upper ?? `Button ${index + 1}`);
    const buttonTextLower = $derived(taskProperties?.button_text_lower ?? "");


</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button class="bg-amber-400 w-[8.75rem] h-[2.125rem] flex flex-col items-center justify-center"
            class:bg-blue-700={hovered}
    >
        <span>{buttonTextUpper}</span>
        {#if buttonTextLower}
            <span class="text-sm">{buttonTextLower}</span>
        {/if}
    </button>
</div>