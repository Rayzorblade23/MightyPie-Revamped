<script lang="ts">
    import '../../app.css';
    import {Button} from "flowbite-svelte";
    import PieMenu from "$lib/components/PieMenu.svelte";
    import {invoke} from "@tauri-apps/api/core";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalPosition} from "@tauri-apps/api/window";

    export let variant: 'primary' | 'secondary' = 'primary';
    export let size: "xs" | "sm" | "md" | "lg" | "xl" | undefined = 'xs';
    export let disabled: boolean = false;
    export let type: 'button' | 'submit' | 'reset' = 'button';

    export let position: LogicalPosition = new LogicalPosition(1000, 500)

    async function getMousePos() {
        const mousePos = await invoke("get_mouse_pos");
        console.log(mousePos);
    }

    let color: "primary" | "red" | "yellow" | "green" | "purple" | "blue" | "light" | "dark" | "none" | "alternative" | undefined;
    let x: number;

    $: {
        switch (variant) {
            case 'primary':
                color = 'primary';
                break;
            case 'secondary':
                color = 'alternative';
                break;

            default:
                color = 'primary';
        }
    }

    function setWindowPosition() {
        getCurrentWindow().setPosition(position);
        console.log("MOVED!");
    }

    onMount(() => {
        console.log("Mounted!");
        // Set up the interval when the component mounts.
        // let intervalId = setInterval(getMousePos, 5000);
        // Call updateCount every 1000ms (1 second)
        getCurrentWindow().setPosition(position);

    });


</script>
<Button on:click={setWindowPosition}>Move</Button>

<main>
    <div class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2">
        <div class="absolute z-10 pt-0.5 top-1/20 left-1/20">
            <Button
                    color="dark"
                    data-tauri-drag-region
                    class="w-[140px] h-[34px] flex items-center justify-center-safe"
            >
                <span data-tauri-drag-region class="text-white">Drag Here</span>
            </Button>
        </div>
        <PieMenu/>
    </div>
</main><!---->