<!-- +page.svelte (Main Page for Pie Menu -->
<script lang="ts">
    // Import subscribeToSubject AND the getter for connection status
    import PieMenuWithTransitions from "$lib/components/piemenu/PieMenuWithTransitions.svelte";
    import type {IPiemenuOpenedMessage, IShortcutPressedMessage} from "$lib/components/piemenu/piemenuTypes.ts";
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_PIEMENU_NAVIGATE,
        PUBLIC_NATSSUBJECT_PIEMENU_OPENED,
        PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED,
        PUBLIC_PIEMENU_SIZE_X,
        PUBLIC_PIEMENU_SIZE_Y
    } from "$env/static/public";
    import {hasPageForMenu} from "$lib/data/configHandler.svelte.ts";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {onMount} from "svelte";
    import {centerWindowAtCursor, moveCursorToWindowCenter} from "$lib/components/piemenu/piemenuUtils.ts";
    import {getSettings} from "$lib/data/settingsHandler.svelte.ts";
    import {goto} from "$app/navigation";

    // --- Core State ---
    // Temporarily force PieMenu to be visible for debugging
    let isPieMenuVisible = $state(false);
    let pageID = $state(0);
    let menuID = $state(0);
    let isNatsReady = $state(false);
    let monitorScaleFactor = $state(1);
    // Animation key to force component remount
    let animationKey = $state(0);
    // Control PieMenu opacity
    let pieMenuOpacity = $state(0);
    // Reference to PieMenu component
    let pieMenuComponent: any = null;

    let keepPieMenuAnchored = $state(false);

    $effect(() => {
        const settings = getSettings();
        keepPieMenuAnchored = settings.keepPieMenuAnchored?.value ?? false;

        if (
            settings?.startInPieMenuConfig?.value && !sessionStorage.getItem('alreadyRedirectedToConfig')
        ) {
            sessionStorage.setItem('alreadyRedirectedToConfig', '1');
            publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});

            goto('/piemenuConfig', {replaceState: true});
        }
    });

    const handlePieMenuVisible = async (newPageID?: number) => {
        if (newPageID !== undefined) {
            pageID = newPageID;
        }
        isPieMenuVisible = true;
        console.log("PieMenu state: VISIBLE, PageID:", pageID);
        if (isNatsReady) {
            publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: true});
        }
        // Set opacity to 1 after a short delay to ensure animations have started
        setTimeout(() => {
            pieMenuOpacity = 1;
        }, 50);
    };

    const handlePieMenuHidden = async () => {
        if (isPieMenuVisible) {
            isPieMenuVisible = false;
            pageID = 0;
            console.log("PieMenu state: HIDDEN");
            if (isNatsReady) {
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
            }
            // Set opacity to 0 when hiding
            pieMenuOpacity = 0;
        }
    };

    const handleShortcutMessage = async (message: string) => {
        const currentWindow = getCurrentWindow();
        try {
            const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);

            // The new backend message now includes pageID. We parse it, but do not use it yet.
            if (shortcutDetectedMsg.shortcutPressed >= 0) {
                console.log("[NATS] Shortcut (" + shortcutDetectedMsg.shortcutPressed + "): Show/Cycle.");
                // Only cycle if the same menu shortcut is pressed
                let newPageID: number;

                // Track if we're changing page or showing a new menu
                const isChangingPage = menuID === shortcutDetectedMsg.shortcutPressed &&
                    await currentWindow.isVisible() &&
                    isPieMenuVisible;

                // Set opacity to 0 immediately
                pieMenuOpacity = 0;

                // Cancel any running animations
                if (pieMenuComponent?.cancelAnimations) {
                    pieMenuComponent.cancelAnimations();
                }

                // Small delay to ensure DOM updates
                await new Promise(resolve => setTimeout(resolve, 20));

                if (shortcutDetectedMsg.openSpecificPage) {
                    menuID = shortcutDetectedMsg.shortcutPressed;
                    newPageID = shortcutDetectedMsg.pageID;
                    monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                } else if (isChangingPage) {
                    // Cycle to the next page
                    const nextPotentialPageID = pageID + 1;
                    newPageID = hasPageForMenu(menuID, nextPotentialPageID) ? nextPotentialPageID : 0;
                    if (!keepPieMenuAnchored) {
                        monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                    }
                } else {
                    // Open the menu
                    menuID = shortcutDetectedMsg.shortcutPressed;
                    newPageID = 0; // Always start with page 0 when switching menus or opening initially
                    monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                }
                await moveCursorToWindowCenter();

                // Increment animation key to trigger button animations
                animationKey++;

                await currentWindow.show();

                if (!document.hidden) {
                    await handlePieMenuVisible(newPageID);
                } else {
                    console.warn("[NATS] Document is hidden. UI remains hidden.");
                    await handlePieMenuHidden();
                }
            } else {
                // Set opacity to 0 before hiding the window
                pieMenuOpacity = 0;
                console.log(`[NATS] Shortcut (${shortcutDetectedMsg.shortcutPressed}): Hide.`);
                // Cancel all animations before hiding
                if (pieMenuComponent?.cancelAnimations) {
                    pieMenuComponent.cancelAnimations();
                }
                // Small delay to ensure DOM updates
                await new Promise(resolve => setTimeout(resolve, 20));
                await currentWindow.hide();
                await handlePieMenuHidden();
            }
        } catch (e) {
            // Set opacity to 0 before hiding the window
            pieMenuOpacity = 0;
            console.error('[NATS] Error in handleShortcutMessage:', e);
            // Cancel all animations before hiding
            if (pieMenuComponent?.cancelAnimations) {
                pieMenuComponent.cancelAnimations();
            }
            // Small delay to ensure DOM updates
            await new Promise(resolve => setTimeout(resolve, 20));
            await currentWindow.hide();
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

    const subscription_navigate_to_page = useNatsSubscription(
        PUBLIC_NATSSUBJECT_PIEMENU_NAVIGATE,
        async (message: string) => {
            const navigateToPageMsg: string = JSON.parse(message);

            console.log(`[NATS] Navigate to page: ${navigateToPageMsg}`);
            publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});

            setTimeout(() => {
                goto(`/${navigateToPageMsg}`, {replaceState: true});
            }, 0);
        }
    );

    $effect(() => {
        if (subscription_navigate_to_page.status === "subscribed" && !isNatsReady) {
            isNatsReady = true;
            console.log("NATS subscription ready.");
        }
        if (subscription_navigate_to_page.error) {
            console.error("NATS subscription error:", subscription_navigate_to_page.error);
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
                    await handlePieMenuVisible(pageID);
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

    onMount(() => {
        const handleKeyDown = async (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                console.log("Escape pressed: closing PieMenu and hiding window.");
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
                await getCurrentWindow().hide();
            }
        };
        window.addEventListener("keydown", handleKeyDown);
        return () => {
            window.removeEventListener("keydown", handleKeyDown);
        };
    });

    onMount(async () => {
        const currentWindow = getCurrentWindow();
        await currentWindow.setSize(new LogicalSize(Number(PUBLIC_PIEMENU_SIZE_X), Number(PUBLIC_PIEMENU_SIZE_Y)));
        console.log("[onMount] Forcing initial hidden state.");
        await getCurrentWindow().hide();
        await handlePieMenuHidden();
    });
</script>

<main>
    <div
            aria-labelledby="piemenu-title"
            aria-modal="true"
            class="absolute border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2"
            role="dialog"
    >
        <h2 class="sr-only" id="piemenu-title">Pie Menu</h2>
        <PieMenuWithTransitions animationKey={animationKey} bind:this={pieMenuComponent} menuID={menuID}
                                opacity={pieMenuOpacity}
                                pageID={pageID}/>
    </div>
</main>