// PieMenuSettings.svelte
<script lang="ts">
    import { calculatePieButtonOffsets, calculatePieButtonPosition } from "$lib/components/piemenu/piemenuUtils.ts";
    import type {PageConfiguration, ButtonData} from "$lib/components/piebutton/piebuttonTypes.ts";

    // Props
    let {
        pageLabel,
        buttonsOnPage,
        selectedButtonIndex,
        onButtonSelect,
    }: {
        pageLabel: string;
        buttonsOnPage: PageConfiguration; // Map<number, ButtonData>
        selectedButtonIndex: number | null;
        onButtonSelect: (buttonIndex: number, taskData: ButtonData) => void;
    } = $props();

    const PIE_DISPLAY_CONFIG = {
        numButtons: 8,
        radius: 110,
        buttonWidthRem: 9,
        buttonHeightRem: 2.5,
        containerSizePx: 320,
    };

    let calculatedButtonPositions = $state<{ x: number; y: number }[]>([]);

    $effect(() => {
        const newPositions: { x: number; y: number }[] = [];
        for (let i = 0; i < PIE_DISPLAY_CONFIG.numButtons; i++) {
            const { offsetX, offsetY } = calculatePieButtonOffsets(i, PIE_DISPLAY_CONFIG.buttonWidthRem, PIE_DISPLAY_CONFIG.buttonHeightRem);
            const { x, y } = calculatePieButtonPosition(
                i,
                PIE_DISPLAY_CONFIG.numButtons,
                offsetX,
                offsetY,
                PIE_DISPLAY_CONFIG.radius,
                PIE_DISPLAY_CONFIG.containerSizePx,
                PIE_DISPLAY_CONFIG.containerSizePx
            );
            newPositions.push({ x, y });
        }
        calculatedButtonPositions = newPositions;
    });

    // Define the type for items in displayableButtons
    type DisplayableButton = {
        buttonIndex: number;
        taskData: ButtonData | undefined;
        position: { x: number; y: number };
    };

    const displayableButtons = $derived<DisplayableButton[]>(() => {
        if (calculatedButtonPositions.length !== PIE_DISPLAY_CONFIG.numButtons) {
            return [];
        }
        const buttonsArray: DisplayableButton[] = []; // Explicitly type the array being built
        for (let i = 0; i < PIE_DISPLAY_CONFIG.numButtons; i++) {
            buttonsArray.push({
                buttonIndex: i,
                taskData: buttonsOnPage.get(i), // .get() on a Map correctly returns ButtonData | undefined
                position: calculatedButtonPositions[i] // This should have a defined type
            });
        }
        return buttonsArray;
    });

</script>

<!-- Template remains the same -->
<div
        class="relative rounded-full bg-slate-800/70 shadow-lg"
        style="width: {PIE_DISPLAY_CONFIG.containerSizePx}px; height: {PIE_DISPLAY_CONFIG.containerSizePx}px;"
        aria-label={`Pie menu settings for ${pageLabel}`}
>
    {#if displayableButtons.length > 0}
        {#each displayableButtons as btn (btn.buttonIndex)} <!-- btn should now be correctly typed as DisplayableButton -->
            <div
                    class="absolute"
                    style="
                    left: {btn.position.x}px;
                    top: {btn.position.y}px;
                    transform: translate(-50%, -50%);
                    width: {PIE_DISPLAY_CONFIG.buttonWidthRem}rem;
                    height: {PIE_DISPLAY_CONFIG.buttonHeightRem}rem;
                "
            >
                <PieButtonSettings
                        taskData={btn.taskData}
                        isSelected={selectedButtonIndex === btn.buttonIndex}
                        onClick={() => {
                        if (btn.taskData) { // taskData is ButtonData | undefined, so this check is important
                            onButtonSelect(btn.buttonIndex, btn.taskData);
                        }
                    }}
                />
            </div>
        {/each}
    {/if}

    <div
            class="absolute left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2
               flex items-center justify-center
               w-16 h-16 rounded-full bg-slate-700/80"
    >
        <span class="text-xs text-slate-300 font-semibold">{pageLabel}</span>
    </div>
</div>