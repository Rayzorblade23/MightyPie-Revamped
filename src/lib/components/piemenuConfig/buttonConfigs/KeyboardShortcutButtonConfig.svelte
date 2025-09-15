<!-- src/lib/components/piemenuConfig/buttonConfigs/KeyboardShortcutButtonConfig.svelte -->
<script lang="ts">
    import {type Button, ButtonType, type KeyboardShortcutProperties} from "$lib/data/types/pieButtonTypes.ts";
    import {createLogger} from "$lib/logger";
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_CAPTURE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_UPDATE
    } from "$env/static/public";
    import StandardButton from "$lib/components/StandardButton.svelte";

    // Create a logger for this component
    const logger = createLogger('KeyboardShortcutButtonConfig');

    let {
        button,
        onUpdate,
        isDialogOpen = $bindable(),
        errorMessage = $bindable<string | null>(null)
    } = $props<{
        button: Button;
        onUpdate: (updatedButton: Button) => void;
        isDialogOpen?: boolean;
        errorMessage?: string | null;
    }>();

    // Cast properties for type safety
    let keyboardShortcutProps = $derived(button.button_type === ButtonType.KeyboardShortcut
        ? button.properties as KeyboardShortcutProperties
        : null);

    // Ensure we're working with a KeyboardShortcut button
    $effect(() => {
        if (button.button_type !== ButtonType.KeyboardShortcut) {
            logger.error('KeyboardShortcutButtonConfig received non-KeyboardShortcut button:', button.button_type);
        }
    });

    // Local state for capture functionality (dialog state comes from parent via isDialogOpen)
    let capturedKeys = $state(''); // RobotGo-compatible keys for execution
    let displayLabel = $state(''); // Friendly display label for UI
    let allowAutoRestart = $state(false);

    // Initialize local state from button properties
    $effect(() => {
        if (keyboardShortcutProps) {
            capturedKeys = keyboardShortcutProps.keys || '';
            // Use existing display text if present
            displayLabel = (keyboardShortcutProps as any).button_text_upper || '';
        }
    });

    // Listen for button shortcut capture updates from the backend
    const subscription_buttonshortcut_update = useNatsSubscription(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_UPDATE, (msg: string) => {
        logger.debug('ButtonShortcut UPDATE raw message:', msg);
        try {
            const obj = JSON.parse(msg);
            if (!obj || typeof obj !== 'object') {
                logger.warn('ButtonShortcut UPDATE ignored: payload not an object');
                return;
            }

            // If backend signals an error (e.g., unmappable key), keep dialog open and restart capture
            if (obj.error) {
                const label = typeof obj.label === 'string' ? obj.label : '';
                logger.warn('ButtonShortcut UPDATE error:', obj.error, label ? `(${label})` : '');
                errorMessage = label ? `This shortcut can't be used: ${label}` : `This shortcut can't be used.`;
                // auto-clear after 3s (but dialog stays open)
                setTimeout(() => {
                    if (errorMessage) errorMessage = null;
                }, 3000);
                // Backend stops hook after first event; restart capture so user can try again immediately
                setTimeout(() => {
                    if (isDialogOpen && allowAutoRestart) {
                        publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_CAPTURE, {});
                        logger.debug('Restarted button shortcut capture after error');
                    }
                }, 100);
                return;
            }

            // Expect both execution keys and display label from backend
            if (obj.keys && typeof obj.keys === 'string') {
                capturedKeys = obj.keys;
            }
            if (obj.label && typeof obj.label === 'string') {
                displayLabel = obj.label;
            }
            if (capturedKeys) {
                isDialogOpen = false; // Close dialog when shortcut is captured
                errorMessage = null;
                allowAutoRestart = false;
                updateButton();
                logger.log('Button shortcut captured:', capturedKeys, displayLabel ? `(${displayLabel})` : '');
            } else {
                logger.warn('ButtonShortcut UPDATE ignored: no valid keys found');
            }
        } catch (error) {
            logger.error('Failed to parse ButtonShortcut UPDATE message:', error);
            isDialogOpen = false; // Close dialog on error
        }
    });

    // Track subscription status and errors, consistent with other pages
    $effect(() => {
        if (subscription_buttonshortcut_update.status === "subscribed") {
            logger.debug("NATS subscription_buttonshortcut_update ready.");
        }
        if (subscription_buttonshortcut_update.error) {
            logger.error("NATS subscription_buttonshortcut_update error:", subscription_buttonshortcut_update.error);
            // If we were capturing a shortcut, close the dialog on error to avoid stuck UI
            if (isDialogOpen) {
                isDialogOpen = false;
            }
        }
    });

    // Start capturing keyboard shortcut - opens dialog and starts capture
    function handleStartCapture() {
        isDialogOpen = true;
        allowAutoRestart = true;
        publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_CAPTURE, {});
        logger.info("Started button shortcut capture");
    }

    // Update button when captured keys change
    function updateButton() {
        if (button.button_type !== ButtonType.KeyboardShortcut) return;

        const displayText = displayLabel.trim() || (capturedKeys.trim() ? capturedKeys.trim() : 'No shortcut set');

        const updatedButton: Button = {
            button_type: ButtonType.KeyboardShortcut,
            properties: {
                button_text_upper: displayText, // Use server-provided label for display
                button_text_lower: '', // Always empty for KeyboardShortcut
                icon_path: 'tabler_icons\\keyboard.svg', // Use keyboard icon
                keys: capturedKeys.trim(), // Keep in RobotGo format from backend
            } as KeyboardShortcutProperties
        };

        onUpdate(updatedButton);
    }

</script>

<div class="mt-4 space-y-4">
    <!-- Keyboard Shortcut Capture -->
    <div class="flex flex-row justify-between items-center w-full">
        <div class="flex flex-col">
            <span class="text-zinc-700 dark:text-zinc-200">Set Keyboard Shortcut to execute:</span>
        </div>
        <StandardButton
                label={displayLabel.trim() || (capturedKeys.trim() ? capturedKeys.trim() : 'Set Shortcut')}
                onClick={handleStartCapture}
                variant="special"
        />
    </div>
</div>

<!-- Dialog is rendered at page level for full opacity -->
