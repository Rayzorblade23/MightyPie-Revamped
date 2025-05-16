<!-- +page.svelte -->
<script lang="ts">
    // Import subscribeToSubject AND the getter for connection status
    import PieMenu from "$lib/components/piemenu/PieMenu.svelte";
    import type {IPiemenuOpenedMessage, IShortcutPressedMessage} from "$lib/components/piemenu/piemenuTypes.ts";
    import {centerWindowAtCursor} from "$lib/components/piemenu/piemenuUtils.ts";
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_PIEMENU_OPENED, PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED} from "$env/static/public";
    import {hasMenuForProfile} from "$lib/components/piebutton/piebuttonConfig.svelte.ts";
    import {getCurrentWindow} from "@tauri-apps/api/window";
    import {onMount} from "svelte";

    // --- Core State ---
    let isPieMenuVisible = $state(false);
    let menuID = $state(0);
    let profileID = $state(0);
    let monitorScaleFactor = $state(1);
    let isNatsReady = $state(false);

    async function handlePieMenuVisible(newMenuID?: number) {
        if (newMenuID !== undefined) {
            menuID = newMenuID;
        }
        if (!isPieMenuVisible) {
            isPieMenuVisible = true;
            console.log("PieMenu state: VISIBLE, MenuID:", menuID);
            if (isNatsReady) {
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: true});
            }
        }
    }

    async function handlePieMenuHidden() {
        if (isPieMenuVisible) {
            isPieMenuVisible = false;
            menuID = 0;
            console.log("PieMenu state: HIDDEN");
            if (isNatsReady) {
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
            }
        }
    }

    const handleShortcutMessage = async (message: string) => {
        const currentWindow = getCurrentWindow();
        try {
            const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);

            if (shortcutDetectedMsg.shortcutPressed === 1) {
                console.log("[NATS] Shortcut (1): Show/Cycle.");
                let newMenuID: number;

                // Simplified logic: if window is visible AND pie menu is visible, then cycle
                if (await currentWindow.isVisible() && isPieMenuVisible) {
                    const nextPotentialMenuID = menuID + 1;
                    newMenuID = hasMenuForProfile(profileID, nextPotentialMenuID) ? nextPotentialMenuID : 0;
                } else {
                    newMenuID = 0;  // Always start with menu 0 when showing initially
                    monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                }

                await currentWindow.show();

                if (!document.hidden) {
                    await handlePieMenuVisible(newMenuID);
                } else {
                    console.warn("[NATS] Document is hidden. UI remains hidden.");
                    await handlePieMenuHidden();
                }
            } else {
                console.log(`[NATS] Shortcut (${shortcutDetectedMsg.shortcutPressed}): Hide.`);
                await currentWindow.hide();
                await handlePieMenuHidden();
            }
        } catch (e) {
            console.error('[NATS] Error in handleShortcutMessage:', e);
            await handlePieMenuHidden();
        }
    };
    const subscription_shortcut_pressed = useNatsSubscription(
        PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED,
        handleShortcutMessage
    );

    $effect(() => {
        if (subscription_shortcut_pressed.status === "subscribed" && !isNatsReady) {
            isNatsReady = true;
            console.log("NATS subscription ready.");
        }
        if (subscription_shortcut_pressed.error) {
            console.error("NATS subscription error:", subscription_shortcut_pressed.error);
        }
    });

    $effect(() => {
        const handleVisibilityChange = async () => {
            const currentWindow = getCurrentWindow();
            if (document.hidden) {
                console.log("Document visibility: HIDDEN");
                await handlePieMenuHidden();
            } else {
                console.log("Document visibility: VISIBLE");
                const tauriWindowIsProgrammaticallyVisible = await currentWindow.isVisible();
                if (tauriWindowIsProgrammaticallyVisible) {
                    await handlePieMenuVisible(menuID);
                } else {
                    await handlePieMenuHidden();
                }
            }
        };

        document.addEventListener("visibilitychange", handleVisibilityChange);

        (async () => {
            const currentWindow = getCurrentWindow();
            const tauriWindowInitiallyVisible = await currentWindow.isVisible();
            if (document.hidden) {
                if (isPieMenuVisible) await handlePieMenuHidden();
            } else {
                if (tauriWindowInitiallyVisible) {
                    if (!isPieMenuVisible) await handlePieMenuVisible(0);
                } else {
                    if (isPieMenuVisible) await handlePieMenuHidden();
                }
            }
        })();

        return () => {
            document.removeEventListener("visibilitychange", handleVisibilityChange);
        };
    });

    onMount(async () => {
        console.log("[onMount] Forcing initial hidden state.");
        await getCurrentWindow().hide();
        await handlePieMenuHidden();
    });

</script>

<main>
    <div
            aria-labelledby="piemenu-title"
            aria-modal="true"
            class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2"
            role="dialog"
    >
        <h2 class="sr-only" id="piemenu-title">Pie Menu</h2>
        <PieMenu menuID={menuID}/>
    </div>
</main>