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
            class="fixed inset-0 flex items-center justify-center p-4 z-50"
            style="background: rgba(0,0,0,0.1);"
            role="dialog"
            aria-modal="true"
            aria-labelledby="dialog-title"
            aria-describedby="dialog-description"
            tabindex="-1"
            bind:this={dialogRef}
            onkeydown={handleKeyDown}
    >
        <div
                class="bg-white rounded-lg shadow-xl max-w-md w-full p-6"
                role="document"
        >
            <h2 id="dialog-title" class="text-xl font-semibold mb-4">{title}</h2>
            <p id="dialog-description" class="mb-6 text-gray-700">{message}</p>
            <div class="flex justify-end space-x-3">
                <button
                        bind:this={cancelButtonRef}
                        onclick={handleCancel}
                        class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
                >
                    {cancelText}
                </button>
                <button
                        onclick={handleConfirm}
                        class="px-4 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 transition-colors"
                >
                    {confirmText}
                </button>
            </div>
        </div>
    </div>
{/if}