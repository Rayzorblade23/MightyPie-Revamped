<script lang="ts">
    import {onDestroy, onMount} from 'svelte';

    export let isOpen: boolean;
    export let onCancel: () => void;

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === 'Escape' && isOpen) {
            event.stopPropagation();
            event.preventDefault();
            onCancel();
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeydown);
    });
    onDestroy(() => {
        window.removeEventListener('keydown', handleKeydown);
    });
</script>

{#if isOpen}
    <div class="fixed inset-0 flex items-center justify-center bg-black/50 z-50">
        <div class="bg-white dark:bg-zinc-800 rounded-lg shadow-lg p-8 text-center min-w-[280px]">
            <div class="mb-4 text-lg font-semibold text-zinc-900 dark:text-zinc-100">Press Modifier Keys and
                Shortcut
            </div>
            <div class="text-zinc-500 dark:text-zinc-300">Waiting for input...</div>
            <button class="mt-6 px-4 py-2 bg-zinc-300 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 rounded hover:bg-zinc-400 dark:hover:bg-zinc-600"
                    on:click={onCancel}>
                Cancel
            </button>
        </div>
    </div>
{/if}

<style>
    /* Optionally, add component-specific styles here */
</style>
