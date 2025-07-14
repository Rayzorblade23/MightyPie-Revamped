<!--PieButton.svelte-->
<script lang="ts">
    import {getButtonProperties, getButtonType,} from "$lib/data/configHandler.svelte.ts";
    import type {IPieButtonExecuteMessage} from "$lib/data/piebuttonTypes.ts";
    import {ButtonType} from "$lib/data/piebuttonTypes.ts";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE} from "$env/static/public";
    import PieButtonBase from './PieButtonBase.svelte';
    import type { MouseState } from '$lib/data/pieButtonSharedTypes';

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

    // Function to publish click events to NATS
    function publishButtonClick(clickType: string) {
        if (!properties || !taskType) return;
        
        const message: IPieButtonExecuteMessage = {
            page_index: pageID,
            button_index: buttonID,
            button_type: taskType,
            properties: properties,
            click_type: clickType,
        };
        
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }

    // Handle the actual click detection logic
    // We maintain this logic here because it's more complex than what PieButtonBase handles
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
    
    // Compute states to pass to the base component
    const isHovered = $derived(mouseState.hovered && !mouseState.leftDown && !mouseState.middleDown && !mouseState.rightDown);
</script>

<PieButtonBase
    {x}
    {y}
    {width}
    {height}
    taskType={taskType || 'empty'}
    {properties}
    {buttonTextUpper}
    {buttonTextLower}
    forceHovered={isHovered}
    forcePressedLeft={mouseState.leftDown}
    forcePressedRight={mouseState.rightDown}
    forcePressedMiddle={mouseState.middleDown}
/>