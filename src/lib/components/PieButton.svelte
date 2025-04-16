<script lang="ts">

    let {index, x, y, hovered}: { index: number, x: number, y: number, hovered: boolean } = $props()

    // Customize the appearance based on index
    let buttonText = `Button ${index + 1}`;

</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button class="bg-amber-400 w-[8.75rem] h-[2.125rem] flex items-center justify-normal"
            class:bg-blue-700={hovered}
    >
        {buttonText}
    </button>
</div>