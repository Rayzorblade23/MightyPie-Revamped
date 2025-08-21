<!-- +page.svelte (Main Page for Pie Menu -->
<script lang="ts">
    // Import subscribeToSubject AND the getter for connection status
    import PieMenu from "$lib/components/piemenu/PieMenu.svelte";
    import type {IPiemenuOpenedMessage, IShortcutPressedMessage} from "$lib/data/types/piemenuTypes.ts";
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_PIEMENU_HEARTBEAT,
        PUBLIC_NATSSUBJECT_PIEMENU_ESCAPE,
        PUBLIC_NATSSUBJECT_PIEMENU_NAVIGATE,
        PUBLIC_NATSSUBJECT_PIEMENU_OPENED,
        PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED,
        PUBLIC_PIEMENU_SIZE_X,
        PUBLIC_PIEMENU_SIZE_Y
    } from "$env/static/public";
    import {hasPageForMenu} from "$lib/data/configManager.svelte.ts";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {onDestroy, onMount} from "svelte";
    import {centerWindowAtCursor, moveCursorToWindowCenter} from "$lib/components/piemenu/piemenuUtils.ts";
    import {getSettings} from "$lib/data/settingsManager.svelte.ts";
    import {goto} from "$app/navigation";
    import {createLogger} from "$lib/logger";

    // Create a logger for this component
    const logger = createLogger('PieMenu');

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
    // Track last known document visibility to avoid redundant handling/logging
    let lastDocHidden: boolean | null = $state(null);

    // Heartbeat interval for mouse hook safety
    let heartbeatInterval: ReturnType<typeof setInterval> | null = $state(null);
    const HEARTBEAT_INTERVAL = 3000; // Send heartbeat every 3 seconds
    // Diagnostics: count heartbeats sent in the current interval
    let heartbeatSendCount = 0;

    let keepPieMenuAnchored = $state(false);

    $effect(() => {
        const settings = getSettings();
        keepPieMenuAnchored = settings.keepPieMenuAnchored?.value ?? false;

        if (
            settings?.startInPieMenuConfig?.value && !sessionStorage.getItem('alreadyRedirectedToConfig')
        ) {
            sessionStorage.setItem('alreadyRedirectedToConfig', '1');
            setPieMenuState(false);

            goto('/piemenuConfig', {replaceState: true});
        }
    });

    // Centralized function to set pie menu state, window visibility, and button unmount timing
    async function setPieMenuState(isOpen: boolean, newPageID?: number) {
        // Idempotency guard: avoid redundant transitions
        if (isOpen) {
            const samePage = newPageID === undefined || newPageID === pageID;
            if (isPieMenuVisible && samePage) {
                return;
            }
        } else {
            if (!isPieMenuVisible) {
                // Already logically hidden; ensure native window is hidden only if currently visible
                const win = getCurrentWindow();
                if (await win.isVisible()) {
                    await win.hide();
                }
                return;
            }
        }
        if (isOpen) {
            // Opening the pie menu
            // Cancel any pending unmount from a previous close/cycle
            if (pieMenuComponent?.cancelButtonsUnmount) {
                pieMenuComponent.cancelButtonsUnmount();
            }
            if (newPageID !== undefined) {
                pageID = newPageID;
            }
            isPieMenuVisible = true;
            logger.debug("PieMenu state: VISIBLE, PageID:", pageID);
            if (isNatsReady) {
                publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: true});
                // Start heartbeat when pie menu is visible and mouse hook is enabled
                startHeartbeat();
            }
            // Set opacity to 1 after a short delay to ensure animations have started
            setTimeout(() => {
                pieMenuOpacity = 1;
            }, 50);
            {
                const win = getCurrentWindow();
                if (!(await win.isVisible())) {
                    await win.show();
                }
            }
        } else {
            // Closing the pie menu
            // Always set opacity to 0 when hiding so the UI fades immediately (even if already hidden)
            pieMenuOpacity = 0;

            // Only perform unmount and notifications if we were actually visible
            if (isPieMenuVisible) {
                // Schedule button unmount via child to allow click-up processing
                if (pieMenuComponent?.scheduleButtonsUnmount) {
                    pieMenuComponent.scheduleButtonsUnmount(100);
                }

                isPieMenuVisible = false;
                logger.debug("PieMenu state: HIDDEN");
                if (isNatsReady) {
                    publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
                    // Stop heartbeat when pie menu is hidden and mouse hook is disabled
                    stopHeartbeat("setPieMenuState: closing");
                }
            } else {
                // Ensure internal state remains consistent when already hidden (suppress noisy log)
                isPieMenuVisible = false;
            }

            // Ensure the native window is hidden only if currently visible to reduce churn
            {
                const win = getCurrentWindow();
                if (await win.isVisible()) {
                    await win.hide();
                }
            }
        }
    }

    // Start sending heartbeats when pie menu is visible
    function startHeartbeat() {
        // Clear any existing interval first (diagnostic reason)
        stopHeartbeat("startHeartbeat: pre-clear existing interval");

        heartbeatSendCount = 0;
        logger.debug(`Starting mouse hook safety heartbeat (creating interval). isNatsReady=${isNatsReady}`);

        // Start sending heartbeats
        heartbeatInterval = setInterval(() => {
            if (isNatsReady) {
                heartbeatSendCount++;
                publishMessage(PUBLIC_NATSSUBJECT_PIEMENU_HEARTBEAT, {
                    timestamp: Date.now(),
                    count: heartbeatSendCount
                });
                logger.debug(`[HB] Sent heartbeat #${heartbeatSendCount} to ${PUBLIC_NATSSUBJECT_PIEMENU_HEARTBEAT}`);
            } else {
                logger.debug(`[HB] Skipped send (NATS not ready). count=${heartbeatSendCount}`);
            }
        }, HEARTBEAT_INTERVAL);

    }

    // Stop sending heartbeats when pie menu is hidden
    function stopHeartbeat(reason?: string) {
        const hadInterval = Boolean(heartbeatInterval);
        if (heartbeatInterval) {
            clearInterval(heartbeatInterval);
            heartbeatInterval = null;
        }
        if (!reason?.startsWith('startHeartbeat')) {
            const msg = `Stopped mouse hook safety heartbeat. reason=${reason ?? '(none)'} hadInterval=${hadInterval} finalCount=${heartbeatSendCount}`;
            logger.debug(msg);
        }
    }

    // Use the centralized function instead of direct handlers
    const handlePieMenuVisible = async (newPageID?: number) => {
        await setPieMenuState(true, newPageID);
    };

    const handlePieMenuHidden = async () => {
        await setPieMenuState(false);
    };

    // Ensure backend knows menu is closed and heartbeat stops on unmount
    onDestroy(() => {
        if (isNatsReady) {
            publishMessage<IPiemenuOpenedMessage>(PUBLIC_NATSSUBJECT_PIEMENU_OPENED, {piemenuOpened: false});
            logger.debug("[onDestroy] Failsafe: sent piemenuOpened=false");
        }
        stopHeartbeat("onDestroy: component unmount");
    });

    const handleShortcutMessage = async (message: string) => {
        const currentWindow = getCurrentWindow();
        // Defensive: if a close scheduled an unmount, cancel it immediately on any new shortcut action
        if (pieMenuComponent?.cancelButtonsUnmount) {
            pieMenuComponent.cancelButtonsUnmount();
        }
        try {
            const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);

            // The new backend message now includes pageID. We parse it, but do not use it yet.
            if (shortcutDetectedMsg.shortcutPressed >= 0) {
                logger.debug("[NATS] Shortcut (" + shortcutDetectedMsg.shortcutPressed + "): Show/Cycle.");
                // Only cycle if the same menu shortcut is pressed
                let newPageID: number;

                // Track if we're changing page or showing a new menu
                const isChangingPage = menuID === shortcutDetectedMsg.shortcutPressed &&
                    await currentWindow.isVisible() &&
                    isPieMenuVisible;

                // Set opacity to 0 immediately
                pieMenuOpacity = 0;

                // Small delay to ensure DOM updates
                await new Promise(resolve => setTimeout(resolve, 20));

                if (shortcutDetectedMsg.openSpecificPage) {
                    menuID = shortcutDetectedMsg.shortcutPressed;
                    newPageID = shortcutDetectedMsg.pageID;
                    logger.info(`Displaying menu ${menuID}, page ${newPageID} (specific page)`);
                } else if (isChangingPage) {
                    // Cycle to the next page
                    const nextPotentialPageID = pageID + 1;
                    newPageID = hasPageForMenu(menuID, nextPotentialPageID) ? nextPotentialPageID : 0;
                    logger.info(`Displaying menu ${menuID}, page ${newPageID} (cycling pages)`);
                    if (!keepPieMenuAnchored) {
                        monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                    }
                } else {
                    // Open the menu
                    menuID = shortcutDetectedMsg.shortcutPressed;
                    newPageID = 0; // Always start with page 0 when switching menus or opening initially
                    logger.info(`Displaying menu ${menuID}, page ${newPageID} (just opened)`);
                    monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
                }
                await moveCursorToWindowCenter();

                // Ensure any pending unmount is cancelled before triggering animations
                if (pieMenuComponent?.cancelButtonsUnmount) {
                    pieMenuComponent.cancelButtonsUnmount();
                }
                // Increment animation key to trigger button animations
                animationKey++;

                // Do not show the window directly; let setPieMenuState handle show
                if (!document.hidden) {
                    await handlePieMenuVisible(newPageID);
                } else {
                    logger.warn("[NATS] Document is hidden. Skipping open.");
                    // Do not force-close here; visibilitychange handler will handle actual hide if needed
                }
            } else {
                logger.debug(`[NATS] Shortcut (${shortcutDetectedMsg.shortcutPressed}): Hide.`);
                await handlePieMenuHidden();
            }
        } catch (e) {
            logger.error('[NATS] Error in handleShortcutMessage:', e);
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
            logger.debug("NATS subscription_shortcut_pressed ready.");
        }
        if (subscription_shortcut_pressed.error) {
            logger.error("NATS subscription_shortcut_pressed error:", subscription_shortcut_pressed.error);
        }
    });

    const subscription_navigate_to_page = useNatsSubscription(
        PUBLIC_NATSSUBJECT_PIEMENU_NAVIGATE,
        async (message: string) => {
            const navigateToPageMsg: string = JSON.parse(message);

            logger.debug(`[NATS] Navigate to page: ${navigateToPageMsg}`);
            setPieMenuState(false);

            setTimeout(() => {
                goto(`/${navigateToPageMsg}`, {replaceState: true});
            }, 0);
        }
    );

    $effect(() => {
        if (subscription_navigate_to_page.status === "subscribed" && !isNatsReady) {
            isNatsReady = true;
            logger.debug("NATS subscription_navigate_to_page ready.");
        }
        if (subscription_navigate_to_page.error) {
            logger.error("NATS subscription_navigate_to_page error:", subscription_navigate_to_page.error);
        }
    });

    $effect(() => {
        const handleVisibilityChange = async () => {
            // Only react when visibility actually changes to reduce churn
            if (lastDocHidden === document.hidden) return;
            lastDocHidden = document.hidden;

            const win = getCurrentWindow();
            if (document.hidden) {
                logger.debug("Document visibility: HIDDEN");
                // Only close if we believe we are visible
                if (isPieMenuVisible) {
                    await handlePieMenuHidden();
                } else {
                    // Ensure native window is hidden if needed
                    if (await win.isVisible()) await win.hide();
                }
            } else {
                logger.debug("Document visibility: VISIBLE");
                // Do not force-open here. If already logically visible but native window is hidden, ensure it's shown.
                if (isPieMenuVisible && !(await win.isVisible())) {
                    await win.show();
                }
            }
        };

        document.addEventListener("visibilitychange", handleVisibilityChange);

        return () => {
            document.removeEventListener("visibilitychange", handleVisibilityChange);
        };
    });

    onMount(() => {
        logger.info('PieMenu Mounted');
    });

    // Subscribe to backend Escape event (independent of window focus)
    const subscription_escape = useNatsSubscription(
        PUBLIC_NATSSUBJECT_PIEMENU_ESCAPE,
        async (_msg: string) => {
            if (!isPieMenuVisible) return;
            logger.debug("[NATS] Escape published: closing PieMenu and hiding window.");
            await setPieMenuState(false);
        }
    );

    $effect(() => {
        if (subscription_escape.error) {
            logger.error("NATS subscription_escape error:", subscription_escape.error);
        }
    });

    // Block browser back/forward triggered by mouse X1/X2 buttons
    onMount(() => {
        const block = (event: Event) => {
            const button = (event as MouseEvent).button;
            // Quick guard: ignore everything except X1/X2
            if (button !== 3 && button !== 4) return;
            event.preventDefault();
            event.stopImmediatePropagation();
        };

        // Use capture + non-passive so preventDefault works before default navigation
        const opts: AddEventListenerOptions = {capture: true, passive: false};
        window.addEventListener('pointerdown', block, opts);
        window.addEventListener('pointerup', block, opts);
        window.addEventListener('mousedown', block, opts);
        window.addEventListener('mouseup', block, opts);
        window.addEventListener('auxclick', block, opts);

        return () => {
            window.removeEventListener('pointerdown', block, opts);
            window.removeEventListener('pointerup', block, opts);
            window.removeEventListener('mousedown', block, opts);
            window.removeEventListener('mouseup', block, opts);
            window.removeEventListener('auxclick', block, opts);
        };
    });

    onMount(async () => {
        const currentWindow = getCurrentWindow();
        await currentWindow.setSize(new LogicalSize(Number(PUBLIC_PIEMENU_SIZE_X), Number(PUBLIC_PIEMENU_SIZE_Y)));
        logger.debug("[onMount] Forcing initial hidden state.");
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
        <PieMenu
                animationKey={animationKey}
                bind:this={pieMenuComponent}
                isMenuOpen={isPieMenuVisible}
                menuID={menuID}
                onClose={() => setPieMenuState(false)}
                opacity={pieMenuOpacity}
                pageID={pageID}
        />

    </div>
</main>