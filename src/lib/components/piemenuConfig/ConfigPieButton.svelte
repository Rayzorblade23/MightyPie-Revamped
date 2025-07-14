<!-- ConfigPieButton.svelte -->
<script lang="ts">
    import type { ButtonPropertiesUnion } from '$lib/data/pieButtonSharedTypes';
    import PieButtonBase from '$lib/components/piebutton/PieButtonBase.svelte';

    let {
        x,
        y,
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        allowSelectWhenDisabled = false,
        onclick = undefined,
    } = $props<{
        x: number,
        y: number,
        width: number,
        height: number,
        taskType: string,
        properties: ButtonPropertiesUnion,
        buttonTextUpper?: string,
        buttonTextLower?: string,
        allowSelectWhenDisabled?: boolean,
        onclick?: (event: MouseEvent) => void
    }>();

    // Forward all props to PieButtonBase which handles all the shared functionality
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
    {allowSelectWhenDisabled}
    {onclick}
/>