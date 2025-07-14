<!-- QuickMenuPieButton.svelte -->
<script lang="ts">
    import { PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE } from "$env/static/public";
    import type { ButtonPropertiesUnion } from '$lib/data/pieButtonSharedTypes';
    import type { IPieButtonExecuteMessage } from "$lib/data/piebuttonTypes";
    import { publishMessage } from "$lib/natsAdapter.svelte.ts";
    import PieButtonBase from '$lib/components/piebutton/PieButtonBase.svelte';

    let {
        pageID,
        buttonID,
        taskType,
        properties,
        buttonTextUpper,
        buttonTextLower,
        width,
        height,
        x,
        y,
    } = $props<{
        pageID: number,
        buttonID: number,
        taskType: string,
        properties: ButtonPropertiesUnion,
        buttonTextUpper: string,
        buttonTextLower: string,
        width: number,
        height: number,
        x?: number,
        y?: number,
    }>();

    function publishButtonClick() {
        if (!properties || !taskType) return;
        const message: IPieButtonExecuteMessage = {
            page_index: pageID, button_index: buttonID, button_type: taskType,
            properties: properties, click_type: "left_up"
        };
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }
</script>

<PieButtonBase
    {x}
    {y}
    {width}
    {height}
    {taskType}
    {properties}
    {buttonTextUpper}
    {buttonTextLower}
    onclick={publishButtonClick}
/>