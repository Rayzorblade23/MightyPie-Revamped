<!-- src/lib/components/piemenuConfig/ConfigPieMenuPage.svelte -->
<script lang="ts">
    import type {Button, ButtonPropertiesUnion, ButtonsOnPageMap} from '$lib/data/piebuttonTypes.ts'; // Assuming you add this to your types file
    import {ButtonType} from '$lib/data/piebuttonTypes.ts';
    import {getDefaultButton} from "$lib/data/pieButtonDefaults";
    import {onMount} from 'svelte';
    import {PUBLIC_PIEBUTTON_HEIGHT, PUBLIC_PIEBUTTON_WIDTH, PUBLIC_PIEMENU_RADIUS} from "$env/static/public";
    import {calculatePieButtonOffsets, calculatePieButtonPosition} from "$lib/components/piemenu/piemenuUtils.ts";
    import ConfigPieButton from "$lib/components/piemenuConfig/ConfigPieButton.svelte";
    import RemovePageButton from "$lib/components/piemenuConfig/elements/RemovePageButton.svelte";
    import ConfirmationDialog from '$lib/components/ui/ConfirmationDialog.svelte';
    import { getIndicatorSVG } from "$lib/components/piemenu/indicatorSVGLoader.svelte.ts";

    // --- Component Props (using $props) ---
    let {
        menuID,
        pageID,
        buttonsOnPage,
        onButtonClick,
        onRemovePage,
        activeSlotIndex = -1,
        isQuickMenuFavorite = false, // Add new prop for quick menu favorite
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
        activeSlotIndex?: number;
        isQuickMenuFavorite?: boolean;
    }>();

    // --- Layout Constants ---
    const numLayoutSlots = 8;
    const radiusPx = Number(PUBLIC_PIEMENU_RADIUS);

    const buttonWidthRem = Number(PUBLIC_PIEBUTTON_WIDTH);
    const buttonHeightRem = Number(PUBLIC_PIEBUTTON_HEIGHT);

    // Container dimensions (in pixels) - for ConfigPieMenuPage visual boundary and position calculations
    const containerWidthPx = 600;
    const containerHeightPx = 400;

    // --- State for calculated button center positions (offsets from container center) ---
    let internalSlotXYOffsets = $state<{ x: number; y: number }[]>([]);

    // --- Indicator State ---
    let indicator = $state("");
    // Offset so that Button 0 is at -67.5deg (if 8 buttons, 360/8 = 45, so 0 is at -67.5)
    const INDICATOR_START_ANGLE = -67.5;
    const INDICATOR_STEP_ANGLE = 45;
    let indicatorRotation = $state(0);
    $effect(() => {
        if (activeSlotIndex >= 0 && activeSlotIndex < numLayoutSlots) {
            indicatorRotation = INDICATOR_START_ANGLE + (activeSlotIndex * INDICATOR_STEP_ANGLE);
        } else {
            indicatorRotation = 0;
        }
    });

    $effect(() => {
        (async () => {
            indicator = await getIndicatorSVG();
        })();
    });

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

        // Always safe to access properties for all button types
        const props = currentButton.properties;
        if (props && 'button_text_upper' in props && typeof props.button_text_upper === 'string') {
            buttonTextUpper = props.button_text_upper;
        }
        if (props && 'button_text_lower' in props && typeof props.button_text_lower === 'string') {
            buttonTextLower = props.button_text_lower;
        }

        if (currentButton.button_type === ButtonType.Disabled) {
            buttonTextUpper = buttonTextUpper || "Disabled";
            buttonTextLower = buttonTextLower || "";
        } else if (currentButton.button_type === ButtonType.CallFunction) {
            buttonTextUpper = buttonTextUpper || (props as import('$lib/data/piebuttonTypes.ts').CallFunctionProperties).button_text_upper || 'Function';
        } else if (currentButton.button_type === ButtonType.ShowAnyWindow) {
            buttonTextUpper = 'Show Any';
        } else if (currentButton.button_type === ButtonType.ShowProgramWindow) {
            buttonTextUpper = 'Show Program';
        }

        return {
            actualButton: currentButton,
            taskType: currentButton.button_type,
            properties: props as ButtonPropertiesUnion,
            buttonTextUpper,
            buttonTextLower,
        };
    }

    function handleSlotClick(slotIndex: number) {
        const buttonFromConfig = buttonsOnPage.get(slotIndex);
        const buttonForDispatch: Button = buttonFromConfig
            ? buttonFromConfig
            : {
                button_type: ButtonType.Disabled,
                properties: {
                    button_text_upper: "",
                    button_text_lower: "",
                    icon_path: "",
                }
            }; // Default to disabled if somehow not found

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

<style>
    .indicator-animated {
        transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.25s cubic-bezier(0.4, 0, 0.2, 1);
        will-change: transform, opacity;
        opacity: 0;
        pointer-events: none;
    }

    .indicator-visible {
        opacity: 0.9;
        pointer-events: none;
    }
</style>

<!-- Main container for the Pie Menu visualization -->
<div
        class="pie-menu-settings-view relative bg-zinc-300 dark:bg-zinc-700"
        style="width: {containerWidthPx}px; height: {containerHeightPx}px"
>
    <RemovePageButton onClick={handleRemoveThisPage} title="Remove this page"/>

    {#if internalSlotXYOffsets.length === 0 && NUM_SLOTS_ITERATION > 0}
        <p class="text-zinc-500 text-xs">Calculating positions...</p>
    {/if}
    <!-- SVG Indicator: Only visible if this menu has the active button -->
    <div
            class="absolute left-1/2 top-1/2 z-0 indicator-animated"
            class:indicator-visible={indicator && activeSlotIndex >= 0 && internalSlotXYOffsets[activeSlotIndex]}
            style="transform: translate(-50%, -50%) rotate({indicatorRotation}deg);"
    >
        <img alt="indicator" height="300" src={indicator} style="display: block; width: 300px; height: 300px;"
             width="300"/>
    </div>
    {#each Array(NUM_SLOTS_ITERATION) as _, slotIndex (slotIndex)}
        {@const displayInfo = getButtonDisplayInfo(slotIndex)}
        {@const positionOffset = internalSlotXYOffsets[slotIndex]}
        {#if positionOffset}
            <ConfigPieButton
                    x={positionOffset.x}
                    y={positionOffset.y}
                    width={buttonWidthRem}
                    height={buttonHeightRem}
                    taskType={displayInfo.taskType}
                    properties={displayInfo.properties ?? getDefaultButton(ButtonType.ShowAnyWindow).properties}
                    buttonTextUpper={displayInfo.buttonTextUpper}
                    buttonTextLower={displayInfo.buttonTextLower}
                    onclick={() => handleSlotClick(slotIndex)}
                    active={slotIndex === activeSlotIndex}
            />
        {/if}
    {/each}

    <div class="absolute left-3 top-3 z-10 pointer-events-none">
        <div class="flex items-center gap-2 mb-2">
            <span
                    class="text-sm font-medium px-3 py-1 rounded-md shadow bg-zinc-100 text-zinc-700 dark:bg-zinc-600 dark:text-zinc-100"
                    style="display:inline-block;"
            >
                Page {pageID + 1}
            </span>
            {#if isQuickMenuFavorite}
                <span style="vertical-align:middle;"><img src="/tabler_icons/star.svg" alt="star icon"
                                                          class="inline w-5 h-5 align-text-bottom dark:invert"/></span>
            {/if}
        </div>
    </div>
</div>

{#if showConfirmDialog}
<ConfirmationDialog
        bind:isOpen={showConfirmDialog}
        cancelText="Cancel"
        confirmText="Remove Page"
        message="This page contains buttons that are not simple buttons. Are you sure you want to remove it?"
        onCancel={handleCancel}
        onConfirm={handleConfirm}
        title="Remove Page"
/>
{/if}
