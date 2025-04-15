<script lang="ts">
    import '../../app.css';
    import PieMenu from '$lib/components/PieMenu.svelte';
    import {getCurrentWindow, LogicalPosition} from '@tauri-apps/api/window';
    import {getOpenAtMousePosition} from '$lib/natsAdapter.svelte.ts';

    let position: { x: number, y: number };

    $effect(() => {
        // Define a synchronous function that calls the asynchronous logic
        const updatePosition = async () => {
            position = getOpenAtMousePosition();
            const window = getCurrentWindow();
            const size = await window.outerSize();

            const centeredX = position.x - size.width / 2;
            const centeredY = position.y - size.height / 2;

            await window.setPosition(new LogicalPosition(centeredX, centeredY));

            console.log(`This is position ${position.x} ${position.y}`);
        };

        updatePosition();
    });

</script>

<main>
    <div class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2">
        <div class="absolute z-10 pt-0.5 top-19/20 left-19/20">
            <button
                    data-tauri-drag-region
                    class="w-[140px] h-[34px] flex items-center justify-center-safe"
            >
                <span data-tauri-drag-region class="text-white">Drag Here</span>
            </button>
        </div>
        <PieMenu/>
    </div>
</main>