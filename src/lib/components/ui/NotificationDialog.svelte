<script lang="ts">
    import ExpandedButton from '$lib/components/ExpandedButton.svelte';
    import {onDestroy} from "svelte";

    let {
        isOpen = $bindable(false),
        title = 'Notification',
        message = '',
        buttonText = 'OK',
        onClose
    } = $props<{
        isOpen: boolean;
        title?: string;
        message?: string;
        buttonText?: string;
        onClose?: () => void;
    }>();

    let dialogRef = $state<HTMLElement | null>(null);
    let buttonRef = $state<any>(null);

    // Focus the dialog when opened
    $effect(() => {
        if (isOpen && buttonRef) {
            setTimeout(() => buttonRef?.focus(), 0);
        }
    });

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            handleClose();
            return;
        }
        // Only confirm if dialog itself is focused
        if (event.key === 'Enter' && document.activeElement === dialogRef) {
            handleClose();
            return;
        }
    }

    function handleClose() {
        isOpen = false;
        onClose?.();
    }

    let cleanup = $state<(() => void) | undefined>(undefined);

    $effect(() => {
        if (isOpen) {
            window.addEventListener('keydown', handleKeyDown);
            cleanup = () => window.removeEventListener('keydown', handleKeyDown);
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
    <div
            class="fixed inset-0 z-[100] flex items-center justify-center pointer-events-auto"
            style="background: rgba(0,0,0,0.5);"
            role="dialog"
            aria-modal="true"
            aria-labelledby="dialog-title"
            aria-describedby="dialog-description"
            tabindex="-1"
            bind:this={dialogRef}
            onkeydown={handleKeyDown}
    >
        <div
                class="bg-white dark:bg-zinc-800 rounded-lg shadow-lg p-6 max-w-sm w-full border border-zinc-200 dark:border-zinc-700"
                role="document"
        >
            <h2 id="dialog-title" class="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-2">{title}</h2>
            <p id="dialog-description" class="text-zinc-700 dark:text-zinc-300 mb-4">{message}</p>
            <div class="flex justify-center mt-6">
                <ExpandedButton
                        bind:this={buttonRef}
                        label={buttonText}
                        variant="primary"
                        onClick={handleClose}
                />
            </div>
        </div>
    </div>
{/if}
