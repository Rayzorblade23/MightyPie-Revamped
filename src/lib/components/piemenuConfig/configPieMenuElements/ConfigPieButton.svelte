<!-- ConfigPieButton.svelte -->
<script lang="ts">
    import type {ButtonPropertiesUnion} from '$lib/data/types/pieButtonSharedTypes.ts';
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
        onclick = undefined,
        oncontextmenu = undefined,
        allowSelectWhenDisabled = true,
        active = false,
    } = $props<{
        x: number,
        y: number,
        width: number,
        height: number,
        taskType: string,
        properties: ButtonPropertiesUnion,
        buttonTextUpper?: string,
        buttonTextLower?: string,
        onclick?: (event: MouseEvent) => void,
        oncontextmenu?: (event: MouseEvent) => void,
        allowSelectWhenDisabled?: boolean,
        active?: boolean
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
        {onclick}
        {oncontextmenu}
        {allowSelectWhenDisabled}
        {active}
/>