<script lang="ts">
    import {onDestroy} from 'svelte';
    import StandardButton from '$lib/components/StandardButton.svelte';

    let {isOpen, onCancel, errorMessage} = $props<{
        isOpen: boolean;
        onCancel: () => void;
        errorMessage?: string | null;
    }>();

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            onCancel();
        }
    }

    let cleanup = $state<(() => void) | undefined>(undefined);

    $effect(() => {
        if (isOpen) {
            window.addEventListener('keydown', handleKeydown);
            cleanup = () => window.removeEventListener('keydown', handleKeydown);
        } else if (cleanup) {
            cleanup();
            cleanup = undefined;
        }
    });

    onDestroy(() => {
        if (cleanup) cleanup();
    });
</script>

{#if isOpen}
    <div class="fixed inset-0 flex items-center justify-center bg-black/50 z-50">
        <div class="bg-white dark:bg-zinc-800 rounded-lg shadow-lg p-8 text-center min-w-[280px]">
            <div class="mb-4 text-lg font-semibold text-zinc-900 dark:text-zinc-100">Press Modifier Keys and
                Shortcut
            </div>
            <div class="text-zinc-500 dark:text-zinc-300">Waiting for input...</div>

            {#if errorMessage}
                <div class="mt-3 text-sm text-red-600 dark:text-red-400" role="alert" aria-live="assertive">
                    {errorMessage}
                </div>
            {/if}

            <div class="flex justify-center mt-4">
                <StandardButton
                        label="Cancel"
                        variant="primary"
                        onClick={onCancel}
                        style="margin-top: 1.5rem;"
                />
            </div>
        </div>
    </div>
{/if}

<style>
    /* Optionally, add component-specific styles here */
</style>
