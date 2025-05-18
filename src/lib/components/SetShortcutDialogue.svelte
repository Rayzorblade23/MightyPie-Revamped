<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    export let isOpen: boolean;
    export let onCancel: () => void;

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === 'Escape' && isOpen) {
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
    <div class="fixed inset-0 flex items-center justify-center bg-black bg-opacity-40 z-50">
        <div class="bg-white rounded-lg shadow-lg p-8 text-center min-w-[280px]">
            <div class="mb-4 text-lg font-semibold">Press Modifier Keys and Shortcut</div>
            <div class="text-gray-500">Waiting for input...</div>
            <button class="mt-6 px-4 py-2 bg-gray-300 rounded hover:bg-gray-400" on:click={onCancel}>
                Cancel
            </button>
        </div>
    </div>
{/if}

<style>
    /* Optionally, add component-specific styles here */
</style>
