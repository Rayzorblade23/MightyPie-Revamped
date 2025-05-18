<!-- src/lib/components/piemenuConfig/SettingsPieMenu.svelte -->
<script lang="ts">
    import type {Button, ButtonPropertiesUnion, ButtonsOnPageMap} from '$lib/data/piebuttonTypes.ts'; // Assuming you add this to your types file
    import {ButtonType} from '$lib/data/piebuttonTypes.ts';
    import {onMount} from 'svelte';
    import {PUBLIC_PIEBUTTON_HEIGHT, PUBLIC_PIEBUTTON_WIDTH, PUBLIC_PIEMENU_RADIUS} from "$env/static/public";
    import {calculatePieButtonOffsets, calculatePieButtonPosition} from "$lib/components/piemenu/piemenuUtils.ts";
    import SettingsPieButton from "$lib/components/piemenuConfig/SettingsPieButton.svelte";
    import RemovePageButton from "$lib/components/piemenuConfig/elements/RemovePageButton.svelte";
    import ConfirmationDialog from '$lib/components/ui/ConfirmationDialog.svelte';

    // --- Component Props (using $props) ---
    let {
        menuID,
        pageID,
        buttonsOnPage,
        onButtonClick,
        onRemovePage,
    } = $props<{
        menuID: number;
        pageID: number;
        buttonsOnPage: ButtonsOnPageMap;
        onButtonClick: (detail: {
            menuID: number;
            pageID: number;
            buttonID: number;
            slotIndex: number;
            button: Button
        }) => void;
        onRemovePage: (pageID: number) => void;
    }>();

    // --- Layout Constants ---
    const numLayoutSlots = 8;
    const radiusPx = Number(PUBLIC_PIEMENU_RADIUS);

    const buttonWidthRem = Number(PUBLIC_PIEBUTTON_WIDTH);
    const buttonHeightRem = Number(PUBLIC_PIEBUTTON_HEIGHT);

    // Container dimensions (in pixels) - for SettingsPieMenu visual boundary and position calculations
    const containerWidthPx = 600;
    const containerHeightPx = 400;

    // --- State for calculated button center positions (offsets from container center) ---
    let internalSlotXYOffsets = $state<{ x: number; y: number }[]>([]);

    onMount(() => {
        let newPositions: { x: number; y: number }[] = [];

        for (let i = 0; i < numLayoutSlots; i++) {

            const {offsetX, offsetY} = calculatePieButtonOffsets(i, buttonWidthRem, buttonHeightRem);

            const {x, y} = calculatePieButtonPosition(
                i,
                numLayoutSlots,
                offsetX,
                offsetY,
                radiusPx,
                containerWidthPx,
                containerHeightPx
            );
            newPositions.push({x: x, y: y});
        }
        internalSlotXYOffsets = newPositions;
    });

    const NUM_SLOTS_ITERATION = numLayoutSlots;

    // --- Display Info Logic (Interface and Function) ---
    interface DisplayButtonInfo {
        actualButton: Button | undefined;
        taskType: ButtonType | 'empty';
        properties: ButtonPropertiesUnion | undefined;
        buttonTextUpper: string;
        buttonTextLower: string;
    }

    function getButtonDisplayInfo(slotIndex: number): DisplayButtonInfo {
        const currentButton = buttonsOnPage.get(slotIndex);

        if (!currentButton) {
            return {
                actualButton: undefined,
                taskType: 'empty' as const,
                properties: undefined,
                buttonTextUpper: `Slot ${slotIndex + 1}`,
                buttonTextLower: 'Empty',
            };
        }

        let buttonTextUpper = '';
        let buttonTextLower = '';

        if (currentButton.button_type !== ButtonType.Disabled && 'properties' in currentButton && currentButton.properties) {
            const props = currentButton.properties;
            if ('button_text_upper' in props && typeof props.button_text_upper === 'string') {
                buttonTextUpper = props.button_text_upper;
            }
            if ('button_text_lower' in props && typeof props.button_text_lower === 'string') {
                buttonTextLower = props.button_text_lower;
            }
        }

        if (currentButton.button_type === ButtonType.Disabled) {
            buttonTextUpper = buttonTextUpper || `Slot ${slotIndex + 1}`;
            buttonTextLower = buttonTextLower || 'Disabled';
        } else if (currentButton.button_type === ButtonType.CallFunction) {
            if ('properties' in currentButton && currentButton.properties) {
                buttonTextUpper = buttonTextUpper || (currentButton.properties as import('$lib/data/piebuttonTypes.ts').CallFunctionProperties).button_text_upper || 'Function';
            } else {
                buttonTextUpper = buttonTextUpper || 'Function';
            }
        } else if (currentButton.button_type === ButtonType.ShowAnyWindow) {
            buttonTextUpper = 'Show Any';
        } else if (currentButton.button_type === ButtonType.ShowProgramWindow) {
            buttonTextUpper = 'Show Program';
        }

        return {
            actualButton: currentButton,
            taskType: currentButton.button_type,
            properties: (currentButton.button_type !== ButtonType.Disabled && 'properties' in currentButton && currentButton.properties)
                ? currentButton.properties as ButtonPropertiesUnion
                : undefined,
            buttonTextUpper,
            buttonTextLower,
        };
    }

    function handleSlotClick(slotIndex: number) {
        const buttonFromConfig = buttonsOnPage.get(slotIndex);
        const buttonForDispatch: Button = buttonFromConfig
            ? buttonFromConfig
            : {button_type: ButtonType.Disabled}; // Default to disabled if somehow not found

        onButtonClick({
            menuID: menuID,
            pageID: pageID,
            buttonID: slotIndex, // Assuming buttonID is the slotIndex for piemenuConfig UI
            slotIndex,
            button: buttonForDispatch
        });
    }

    function hasNonSimpleButtons(buttons: ButtonsOnPageMap): boolean {
        for (const button of buttons.values()) {
            if (button &&
                button.button_type !== ButtonType.ShowAnyWindow &&
                button.button_type !== ButtonType.Disabled) {
                return true;
            }
        }
        return false;
    }

    let showConfirmDialog = $state(false);
    let pendingPageID: number | null = null;

    function handleRemoveThisPage(event: MouseEvent) {
        event.stopPropagation();

        if (hasNonSimpleButtons(buttonsOnPage)) {
            pendingPageID = pageID;
            showConfirmDialog = true;
            return;
        }

        onRemovePage(pageID);
    }

    function handleConfirm() {
        if (pendingPageID !== null) {
            onRemovePage(pendingPageID);
            pendingPageID = null;
        }
        showConfirmDialog = false;
    }

    function handleCancel() {
        pendingPageID = null;
        showConfirmDialog = false;
    }
</script>

<!-- Main container for the Pie Menu visualization -->
<div
        class="pie-menu-settings-view relative"
        style="width: {containerWidthPx}px; height: {containerHeightPx}px"
>
    <RemovePageButton onClick={handleRemoveThisPage} title="Remove this page"/>

    {#if internalSlotXYOffsets.length === 0 && NUM_SLOTS_ITERATION > 0}
        <p class="text-gray-500 text-xs">Calculating positions...</p>
    {/if}
    {#each Array(NUM_SLOTS_ITERATION) as _, slotIndex (slotIndex)}
        {@const displayInfo = getButtonDisplayInfo(slotIndex)}
        {@const positionOffset = internalSlotXYOffsets[slotIndex]}
        {#if positionOffset}
            <SettingsPieButton
                    x={positionOffset.x}
                    y={positionOffset.y}
                    width={buttonWidthRem}
                    height={buttonHeightRem}
                    taskType={displayInfo.taskType}
                    properties={displayInfo.properties}
                    buttonTextUpper={displayInfo.buttonTextUpper}
                    buttonTextLower={displayInfo.buttonTextLower}
                    onclick={() => handleSlotClick(slotIndex)}
            />
        {/if}
    {/each}

    <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
        <span class="text-sm font-medium text-gray-700 bg-white/70 px-2 py-0.5 rounded-md shadow">Page {pageID + 1}</span>
    </div>
</div>

<ConfirmationDialog
        bind:isOpen={showConfirmDialog}
        title="Remove Page"
        message="This page contains buttons that are not simple buttons. Are you sure you want to remove it?"
        confirmText="Remove Page"
        cancelText="Cancel"
        onConfirm={handleConfirm}
        onCancel={handleCancel}
/>