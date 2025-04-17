<script lang="ts">
    import PieButton from './PieButton.svelte';
    import {onMount} from 'svelte';



    // Configuration
    const numButtons = 8;
    const radius = 150;
    const buttonWidth = 8.75;
    const buttonHeight = 2.125;
    const width = 600;
    const height = 500;


    let props = $props();
    let buttonPositions: { x: number; y: number }[] = $state([]);

    function convertRemToPixels(rem: number) {
        return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
    }

    function calculateOffsets(i: number): { offsetX: number; offsetY: number } {
        const buttonWidthPx = convertRemToPixels(buttonWidth)
        const buttonHeightPx = convertRemToPixels(buttonHeight)

        const nudgeX = buttonWidthPx / 2 - buttonHeightPx / 2;
        const nudgeY = buttonHeightPx / 2;

        let offsetX = 0;
        let offsetY = 0;

        if (i === 1) {
            offsetX += nudgeX;
            offsetY -= nudgeY;
        } else if (i === 2) {
            offsetX += nudgeX;
            offsetY += 0;
        } else if (i === 3) {
            offsetX += nudgeX;
            offsetY += nudgeY;
        } else if (i === 5) {
            offsetX -= nudgeX;
            offsetY += nudgeY;
        } else if (i === 6) {
            offsetX -= nudgeX;
            offsetY += 0;
        } else if (i === 7) {
            offsetX -= nudgeX;
            offsetY -= nudgeY;
        }
        return {offsetX, offsetY};
    }

    function calculateButtonPosition(
        index: number,
        numButtons: number,
        offsetX: number,
        offsetY: number,
        radius: number):
        {
            x: number;
            y: number
        } {
        const centerX = width / 2;
        const centerY = height / 2;

        const angleInRad = (index / numButtons) * 2 * Math.PI;

        const x = centerX + offsetX + radius * Math.sin(angleInRad);
        const y = centerY - offsetY - radius * Math.cos(angleInRad);
        return {x, y};
    }

    onMount(() => {
        console.log("PieMenu.svelte: onMount hook running");  // Check if onMount is executed

        let newButtonPositions: { x: number; y: number }[] = []; // Create a new array

        for (let i = 0; i < numButtons; i++) {
            const {offsetX, offsetY} = calculateOffsets(i);
            const {x, y} = calculateButtonPosition(i, numButtons, offsetX, offsetY, radius);
            newButtonPositions = [...newButtonPositions, {x: x, y: y}]; // Add to the new array
        }
        buttonPositions = newButtonPositions; // Assign the new array to buttonPositions

    });
</script>

<div class="relative" style="width: {width}px; height: {height}px;">
    {#each buttonPositions as position, i}
        <PieButton index={i} x={position.x} y={position.y} hovered={props.slice === i}/>
    {/each}
</div>

<style>
    /* Consider adding a backdrop style here to dim the background */
    .relative {
        /* Center the pie menu */
        display: flex;
        justify-content: center;
        align-items: center;
        backdrop-filter: brightness(50%);
    }
</style>