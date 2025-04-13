<script lang="ts">
    import {Button} from "flowbite-svelte";

    export let index: number;
    export let x: number;
    export let y: number;

    // Customize the appearance based on index
    let buttonText = `Button ${index + 1}`;

</script>

<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <Button class="w-[8.75rem] h-[2.125rem] flex items-center justify-normal">
        {buttonText}
    </Button>
</div>