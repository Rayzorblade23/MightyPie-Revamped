<script lang="ts">
    import ExpandedButton from '$lib/components/ExpandedButton.svelte';

    let {
        isOpen = $bindable(false),
        title = 'Elevation Required',
        message = 'This operation requires administrator privileges. Do you want to restart the application with elevated privileges?',
        confirmText = 'Restart as Admin',
        cancelText = 'Cancel',
        onConfirm,
        onCancel,
        onClose
    } = $props<{
        isOpen: boolean;
        title?: string;
        message?: string;
        confirmText?: string;
        cancelText?: string;
        onConfirm?: () => void;
        onCancel?: () => void;
        onClose?: () => void;
    }>();

    let dialogRef = $state<HTMLElement | null>(null);
    let cancelButtonRef = $state<any>(null);
    let confirmButtonRef = $state<any>(null);

    // Focus the dialog when opened
    $effect(() => {
        if (isOpen && cancelButtonRef) {
            setTimeout(() => cancelButtonRef?.focus(), 0);
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
            handleConfirm();
            return;
        }
        if (event.key === 'Tab' && dialogRef) {
            const focusable = dialogRef.querySelectorAll<HTMLElement>(
                'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
            );
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (event.shiftKey) {
                if (document.activeElement === first) {
                    event.preventDefault();
                    last.focus();
                }
            } else {
                if (document.activeElement === last) {
                    event.preventDefault();
                    first.focus();
                }
            }
        }
    }

    function handleConfirm() {
        isOpen = false;
        onConfirm?.();
    }

    function handleCancel() {
        isOpen = false;
        onCancel?.();
    }

    function handleClose() {
        isOpen = false;
        if (onClose) {
            onClose();
        } else {
            onCancel?.();
        }
    }
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
        >
            <h2 id="dialog-title" class="text-lg font-semibold text-zinc-900 dark:text-white mb-2">{title}</h2>
            <p id="dialog-description" class="text-zinc-700 dark:text-zinc-300 mb-6">{message}</p>
            <div class="flex justify-end space-x-2">
                <ExpandedButton
                        label={cancelText}
                        bind:this={cancelButtonRef}
                        onClick={handleCancel}
                        class="bg-zinc-200 hover:bg-zinc-300 dark:bg-zinc-700 dark:hover:bg-zinc-600 text-zinc-900 dark:text-white"
                >
                    {cancelText}
                </ExpandedButton>
                <ExpandedButton
                        label={confirmText}
                        bind:this={confirmButtonRef}
                        onClick={handleConfirm}
                        class="bg-amber-500 hover:bg-amber-600 text-white"
                >
                    {confirmText}
                </ExpandedButton>
            </div>
        </div>
    </div>
{/if}
