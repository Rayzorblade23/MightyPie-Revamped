<script lang="ts">
    let {
        isOpen = $bindable(false),
        title = 'Confirmation',
        message = 'Are you sure you want to proceed?',
        confirmText = 'Confirm',
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
    let cancelButtonRef = $state<HTMLButtonElement | null>(null);

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
                role="document"
        >
            <h2 id="dialog-title" class="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-2">{title}</h2>
            <p id="dialog-description" class="text-zinc-700 dark:text-zinc-300 mb-4">{message}</p>
            <div class="flex justify-end space-x-2">
                <button
                        bind:this={cancelButtonRef}
                        onclick={handleCancel}
                        class="px-4 py-2 rounded bg-zinc-200 dark:bg-zinc-700 text-zinc-800 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 border border-zinc-300 dark:border-zinc-600 transition-colors"
                >
                    {cancelText}
                </button>
                <button
                        onclick={handleConfirm}
                        class="px-4 py-2 rounded bg-blue-600 dark:bg-blue-500 text-white dark:text-zinc-100 hover:bg-blue-700 dark:hover:bg-blue-600 border border-blue-700 dark:border-blue-600 transition-colors"
                >
                    {confirmText}
                </button>
            </div>
        </div>
    </div>
{/if}