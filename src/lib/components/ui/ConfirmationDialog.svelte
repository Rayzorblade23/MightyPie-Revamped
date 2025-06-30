<script lang="ts">
    let {
        isOpen = $bindable(false),
        title = 'Confirmation',
        message = 'Are you sure you want to proceed?',
        confirmText = 'Confirm',
        cancelText = 'Cancel',
        onConfirm,
        onCancel
    } = $props<{
        isOpen: boolean;
        title?: string;
        message?: string;
        confirmText?: string;
        cancelText?: string;
        onConfirm?: () => void;
        onCancel?: () => void;
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
            handleCancel();
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
                class="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 max-w-sm w-full border border-gray-200 dark:border-gray-700"
                role="document"
        >
            <h2 id="dialog-title" class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">{title}</h2>
            <p id="dialog-description" class="text-gray-700 dark:text-gray-300 mb-4">{message}</p>
            <div class="flex justify-end space-x-2">
                <button
                        bind:this={cancelButtonRef}
                        onclick={handleCancel}
                        class="px-4 py-2 rounded bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600 border border-gray-300 dark:border-gray-600 transition-colors"
                >
                    {cancelText}
                </button>
                <button
                        onclick={handleConfirm}
                        class="px-4 py-2 rounded bg-blue-600 dark:bg-blue-500 text-white dark:text-gray-100 hover:bg-blue-700 dark:hover:bg-blue-600 border border-blue-700 dark:border-blue-600 transition-colors"
                >
                    {confirmText}
                </button>
            </div>
        </div>
    </div>
{/if}