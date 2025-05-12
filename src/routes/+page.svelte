<!-- +page.svelte -->
<script lang="ts">
    // Import subscribeToSubject AND the getter for connection status
    import PieMenu from "$lib/components/piemenu/PieMenu.svelte";
    import type {IShortcutPressedMessage} from "$lib/components/piemenu/piemenuTypes.ts";
    import {centerWindowAtCursor} from "$lib/components/piemenu/piemenuUtils.ts";
    import {useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED} from "$env/static/public";
    import {hasMenuForProfile} from "$lib/components/piebutton/piebuttonConfig.svelte.ts";
    // Optional: import { getCurrentWindow } from '@tauri-apps/api/window';

    // --- State ---
    let isVisible = $state(false);
    let monitorScaleFactor = $state(1);
    let menuID = $state(0);
    let profileID = $state(0);

    // --- NATS Message Handler ---
    const handleShortcutMessage = async (message: string) => {
        try {
            const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);
            if (shortcutDetectedMsg.shortcutPressed === 1) {
                // --- Determine the target menuID ---
                if (!isVisible) {
                    menuID = 0;
                } else {
                    const nextPotentialMenuID = menuID + 1;
                    if (hasMenuForProfile(profileID, nextPotentialMenuID)) {
                        menuID = nextPotentialMenuID;
                    } else {
                        menuID = 0;
                    }
                }
                console.log("Menu ID: " + menuID);
                console.log("Profile ID: " + profileID);

                console.log("Shortcut received, centering and showing menu...");
                monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                isVisible = true;
                // Optional: Show/focus Tauri window
                // await getCurrentWindow().show();
                // await getCurrentWindow().setFocus();
            } else {
                console.log(`Received shortcut message, but value is not 1: ${shortcutDetectedMsg.shortcutPressed}`);
            }
        } catch (e) {
            console.error('Failed to parse shortcut message:', e);
        }
    };

    const subscription_shortcut_pressed = useNatsSubscription(
        PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED,
        handleShortcutMessage
    );

    $effect(() => {
        console.log("subscription_button_click Status:", subscription_shortcut_pressed.status); // e.g., 'subscribing', 'subscribed', 'failed'
        if (subscription_shortcut_pressed.error) {
            console.error("subscription_button_click Error:", subscription_shortcut_pressed.error);
        }
    });

</script>

<main>
    {#if isVisible}
        <div
                class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2"
                role="dialog"
                aria-modal="true"
                aria-labelledby="piemenu-title"
        >
            <h2 id="piemenu-title" class="sr-only">Pie Menu</h2>
            <PieMenu menuID={menuID}/>
        </div>
    {/if}
</main>