<script lang="ts">
    import {
        getPieMenuConfig,
        parseButtonConfig,
        updateLiveButtonConfig,
        updatePieMenuConfig
    } from '$lib/data/configManager.svelte.ts';
    import {
        getInstalledAppsInfo,
        parseInstalledAppsInfo,
        updateInstalledAppsInfo
    } from '$lib/data/installedAppsInfoManager.svelte.ts';
    import {validateAndSyncConfig} from '$lib/data/configValidation.svelte.ts';

    import {onMount} from "svelte";
    import {browser} from "$app/environment";
    import "../app.css";

    import {
        connectToNats,
        disconnectFromNats,
        fetchLatestFromStream,
        getConnectionStatus
    } from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_APPNAME,
        PUBLIC_NATSSUBJECT_LIVEBUTTONCONFIG,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE,
        PUBLIC_NATSSUBJECT_SETTINGS_UPDATE,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO,
        PUBLIC_PIEMENU_SIZE_X,
        PUBLIC_PIEMENU_SIZE_Y
    } from '$env/static/public';
    import type {MenuConfigData} from '$lib/data/types/pieButtonTypes.ts';
    import type {PieMenuConfig} from '$lib/data/types/piemenuConfigTypes';
    import {goto} from '$app/navigation';
    import {listen} from '@tauri-apps/api/event';
    import {getSettings, type SettingsMap, updateSettings} from '$lib/data/settingsManager.svelte.ts';
    import {saturateHexColor} from "$lib/colorUtils.ts";
    import {createLogger} from "$lib/logger";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils";
    import {getCurrentWindow, UserAttentionType} from '@tauri-apps/api/window';
    import {exitApp} from "$lib/generalUtil.ts";

    // Create a logger for this component
    const logger = createLogger('Layout');

    let validationHasRun = false;

    // Track if we have apps info available for validation
    let hasAppsInfo = $state(false);
    // Track if we ever successfully connected, to avoid showing crash dialog on initial startup
    let hasConnectedOnce = $state(false);
    // Controls visibility of the crash/reconnect dialog
    let showDisconnectDialog = $state(false);

    $effect(() => {
        const pieMenuConfig = getPieMenuConfig();
        const installedAppsInfo = getInstalledAppsInfo();

        // Update apps info tracking
        hasAppsInfo = installedAppsInfo.size > 0;

        // Determine if the full Pie Menu Config contains any buttons
        const hasPieMenuButtons = !!pieMenuConfig && !!pieMenuConfig.buttons && Object.keys(pieMenuConfig.buttons).length > 0;

        // Run validation once when both full config and apps info are present
        if (!validationHasRun && hasPieMenuButtons && hasAppsInfo) {
            logger.debug("Running initial config validation...");
            validateAndSyncConfig();
            validationHasRun = true;
        }
    });

    // Globally block browser back/forward triggered by mouse X1/X2 buttons
    onMount(() => {
        const block = (event: Event) => {
            const button = (event as MouseEvent).button;
            if (button !== 3 && button !== 4) return; // only X1/X2
            event.preventDefault();
            event.stopImmediatePropagation();
        };

        const opts: AddEventListenerOptions = { capture: true, passive: false };
        window.addEventListener('pointerdown', block, opts);
        window.addEventListener('pointerup', block, opts);
        window.addEventListener('mousedown', block, opts);
        window.addEventListener('mouseup', block, opts);
        window.addEventListener('auxclick', block, opts);
        logger.debug('Registered global X1/X2 back-forward mouse blocker');

        return () => {
            window.removeEventListener('pointerdown', block, opts);
            window.removeEventListener('pointerup', block, opts);
            window.removeEventListener('mousedown', block, opts);
            window.removeEventListener('mouseup', block, opts);
            window.removeEventListener('auxclick', block, opts);
            logger.debug('Unregistered global X1/X2 back-forward mouse blocker');
        };
    });

    let {children} = $props();
    let connectionStatus = $state('Idle');
    let minDisplayTimeMs = 3000; // 3 seconds display time for success screen
    let showSuccessScreen = $state(false);
    // Only show the initial connecting screen if it lasts longer than this delay
    const initialConnectingDelayMs = 2000;
    let showInitialConnecting = $state(false);
    let connectingDelayTimer: number | null = null;

    $effect(() => {
        const actualStatus = getConnectionStatus();

        // Handle connection status changes
        if (actualStatus === 'connected' && connectionStatus !== 'connected') {
            // Mark that we connected at least once
            hasConnectedOnce = true;
            // When connection is successful, show the success screen
            connectionStatus = 'connected';
            showSuccessScreen = true;
            showDisconnectDialog = false; // hide any crash dialog

            // After 3 seconds, hide the success screen
            setTimeout(() => {
                showSuccessScreen = false;
            }, minDisplayTimeMs);

            // Clear any pending initial-connecting delay
            if (connectingDelayTimer !== null) {
                clearTimeout(connectingDelayTimer);
                connectingDelayTimer = null;
            }
            showInitialConnecting = false;
        } else if (actualStatus !== 'connected') {
            // For non-connected states, update immediately
            connectionStatus = actualStatus;
            showSuccessScreen = false;
            // Manage delayed initial connecting screen before first successful connect
            if (!hasConnectedOnce && !showDisconnectDialog) {
                if (connectingDelayTimer === null) {
                    connectingDelayTimer = window.setTimeout(() => {
                        showInitialConnecting = true;
                        connectingDelayTimer = null;
                    }, initialConnectingDelayMs);
                }
            } else {
                // Either we've connected once or dialog is showing; do not show the initial connecting
                if (connectingDelayTimer !== null) {
                    clearTimeout(connectingDelayTimer);
                    connectingDelayTimer = null;
                }
                showInitialConnecting = false;
            }
            // Only show dialog if we've connected before (to avoid startup noise)
            const shouldShow = hasConnectedOnce && (actualStatus === 'reconnecting' || actualStatus === 'closed' || actualStatus === 'error');
            if (shouldShow && !showDisconnectDialog) {
                showDisconnectDialog = true;
                logger.debug("Showing disconnect dialog (status:", actualStatus, ")");
                // Bring window to front so the dialog is visible
                ensureWindowVisible();
            }
        }

        logger.debug("NATS connection status:", connectionStatus);
    });

    const handleLiveButtonUpdateMessage = (message: string) => {
        handleJsonMessage<MenuConfigData>(
            message,
            (LiveButtonConfig) => {
                const newParsedConfig = parseButtonConfig(LiveButtonConfig);
                updateLiveButtonConfig(newParsedConfig);
            },
            '+layout.svelte: Button Update'
        );
    };


    // Handle backend full-config messages by storing full config
    const handleBackendConfigMessage = (message: string) => {
        handleJsonMessage<PieMenuConfig>(
            message,
            (piemenuConfig) => {
                updatePieMenuConfig(piemenuConfig);
                // Run validation only when we have apps info and there are buttons in the full config
                const hasButtons = !!piemenuConfig && !!piemenuConfig.buttons && Object.keys(piemenuConfig.buttons).length > 0;
                if (hasAppsInfo && hasButtons) {
                    validateAndSyncConfig();
                }
            },
            '+layout.svelte: Backend Config Update'
        );
    };

    const handleInstalledAppsMessage = (message: string) => {
        try {
            const installedAppsInfo = parseInstalledAppsInfo(message);
            updateInstalledAppsInfo(installedAppsInfo);
        } catch (error) {
            logger.error("[+layout.svelte] Failed to process installed apps message:", error);
        }
    };

    const handleSettingsUpdateMessage = (message: string) => {
        handleJsonMessage<SettingsMap>(
            message,
            (settingsData) => {
                updateSettings(settingsData);
            },
            '+layout.svelte: Settings Update'
        );
    };

    $effect(() => {
        const settings = getSettings();
        if (!settings) return;
        const map = {
            colorAccentAnyWin: '--color-accent-anywin',
            colorAccentProgramWin: '--color-accent-programwin',
            colorAccentLaunch: '--color-accent-launch',
            colorAccentFunction: '--color-accent-function',
            colorAccentOpenPage: '--color-accent-openpage',
            colorAccentResource: '--color-accent-resource',
            colorPieButtonHighlight: '--color-button-hover-bg',
        };
        for (const [key, cssVar] of Object.entries(map)) {
            if (settings[key]?.value) {
                document.documentElement.style.setProperty(cssVar, settings[key].value);
            }
        }

        // Generate and set more saturated highlight for pressed states
        const highlightColor = settings.colorPieButtonHighlight?.value;
        if (highlightColor) {
            const saturated = saturateHexColor(highlightColor, 1.4);
            document.documentElement.style.setProperty('--color-button-pressed-left-bg', saturated);
            document.documentElement.style.setProperty('--color-button-pressed-middle-bg', saturated);
            document.documentElement.style.setProperty('--color-button-pressed-right-bg', saturated);
        }
    });

    $effect(() => {
        let stopLiveButtons: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopLiveButtons = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_LIVEBUTTONCONFIG,
                    handleLiveButtonUpdateMessage
                );
            })();
        }
        return () => stopLiveButtons?.();
    });

    // Subscribe to BACKEND update (full config)
    $effect(() => {
        let stopBackendFull: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopBackendFull = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE,
                    handleBackendConfigMessage
                );
            })();
        }
        return () => stopBackendFull?.();
    });

    $effect(() => {
        let stopInstalledApps: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopInstalledApps = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO,
                    handleInstalledAppsMessage
                );
            })();
        }
        return () => stopInstalledApps?.();
    });

    // Removed redundant shortcut labels subscription; shortcuts are part of the full config


    $effect(() => {
        let stopSettingsUpdate: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopSettingsUpdate = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_SETTINGS_UPDATE,
                    handleSettingsUpdateMessage
                );
            })();
        }
        return () => stopSettingsUpdate?.();
    });

    // Function to exit the application
    const handleExit = () => {
        logger.info("User requested application exit");
        // Add a small delay to ensure log message is processed
        setTimeout(() => {
            exitApp();
        }, 100);
    };

    // Ensure the window is visible and focused (helps when tray-only / hidden)
    async function ensureWindowVisible() {
        try {
            const win = getCurrentWindow();
            await centerAndSizeWindowOnMonitor(win, Number(PUBLIC_PIEMENU_SIZE_X), Number(PUBLIC_PIEMENU_SIZE_Y));
            await win.show();
            await win.unminimize();
            await win.setFocus();
            // Also request user attention to surface the window on some systems
            try {
                await win.requestUserAttention(UserAttentionType.Critical);
            } catch {
            }
        } catch (e) {
            logger.warn("Failed to ensure window visibility:", e);
        }
    }

    onMount(() => {
        if (browser) {
            const initializeConnection = async () => {
                try {
                    // Center the window on startup
                    await centerAndSizeWindowOnMonitor(getCurrentWindow(), Number(PUBLIC_PIEMENU_SIZE_X), Number(PUBLIC_PIEMENU_SIZE_Y));

                    logger.info("Attempting to connect to NATS...");
                    await connectToNats();
                    // The $effect watching connectionStatus will handle sending the request
                } catch (error) {
                    logger.error("[+layout.svelte] Failed to connect to NATS:", error);
                }
            };
            initializeConnection();
        }

        return () => {
            if (browser) {
                disconnectFromNats();
            }
        };
    });

    let unlistenQuickMenu: (() => void) | undefined;
    let unlistenSettings: (() => void) | undefined;
    let unlistenPieMenuConfig: (() => void) | undefined;

    onMount(() => {
        // Tauri tray event listeners
        listen('show-quickMenu', () => goto('/quickMenu')).then(unlisten => {
            unlistenQuickMenu = unlisten;
        });
        listen('show-settings', () => goto('/settings')).then(unlisten => {
            unlistenSettings = unlisten;
        });
        listen('show-piemenuconfig', () => goto('/piemenuConfigEditor')).then(unlisten => {
            unlistenPieMenuConfig = unlisten;
        });
        return () => {
            if (unlistenQuickMenu) unlistenQuickMenu();
            if (unlistenSettings) unlistenSettings();
            if (unlistenPieMenuConfig) unlistenPieMenuConfig();
        };
    });

    function handleJsonMessage<T>(
        message: string,
        onSuccess: (parsedData: T) => void,
        sourceHint: string
    ): void {
        try {
            const parsed = JSON.parse(message) as T;
            onSuccess(parsed);
        } catch (error) {
            logger.error(`[${sourceHint}] Failed to parse JSON message:`, error);
        }
    }
</script>

{#if showSuccessScreen}
    <div class="w-full min-h-screen overflow-hidden flex items-center justify-center bg-zinc-100 dark:bg-zinc-900 rounded-2xl shadow-lg relative">
        <div class="flex flex-col items-center space-y-13">
            <h1 class="text-2xl font-bold text-zinc-900 dark:text-white">{PUBLIC_APPNAME}</h1>

            <div class="flex items-center">
                <div class="flex items-center justify-center mr-3">
                    <div class="relative h-4 w-4">
                        <div class="absolute inset-0 rounded-full bg-green-600 opacity-75 animate-ping"></div>
                        <div class="relative rounded-full h-4 w-4 bg-green-500"></div>
                    </div>
                </div>
                <p class="text-zinc-900 dark:text-white">Startup successful!</p>
            </div>
        </div>
    </div>
{:else if connectionStatus === "connected"}
    {@render children()}
{:else if hasConnectedOnce && !showDisconnectDialog}
    <!-- After first successful connection, render the app unless the disconnect dialog is active -->
    {@render children()}
{:else if showDisconnectDialog}
    <!-- Disconnect state: match full-screen card layout (no overlay/backdrop) -->
    <div class="w-full min-h-screen flex items-center justify-center bg-zinc-100 dark:bg-zinc-900 rounded-2xl shadow-lg relative">
        <div class="flex flex-col items-center space-y-5 text-center">
            <h1 class="text-2xl font-bold text-zinc-900 dark:text-white">{PUBLIC_APPNAME}</h1>

            <div class="flex items-center">
                <div class="flex items-center justify-center mr-3">
                    <div class="relative h-4 w-4">
                        <div class="absolute inset-0 rounded-full bg-red-600 opacity-75 animate-ping"></div>
                        <div class="relative rounded-full h-4 w-4 bg-red-500"></div>
                    </div>
                </div>
                <p class="text-zinc-900 dark:text-white">Backend connection lost</p>
            </div>

            <div class="bg-red-800 p-4 rounded-lg max-w-md text-center text-white">
                <p class="mb-2">Oops, this shouldn't have happened!</p>
                <p>The app will keep trying to reconnect in the background but you should probably just restart the
                    app.</p>
            </div>

            <button class="w-auto px-4 py-2 bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                    onclick={handleExit}
            >
                Backend is dead. Quit.
            </button>
        </div>
    </div>
{:else if connectionStatus === "error"}
    <div class="w-full min-h-screen flex items-center justify-center bg-zinc-100 dark:bg-zinc-900 rounded-2xl shadow-lg relative">
        <div class="flex flex-col items-center space-y-5">
            <h1 class="text-2xl mb-4 font-bold text-zinc-900 dark:text-white">{PUBLIC_APPNAME}</h1>
            <div class="mb-4 text-zinc-700 dark:text-zinc-400">
                <p>(╯°□°）╯︵ ┻━┻</p>
            </div>
            <div class="bg-red-800 p-4 rounded-lg max-w-md text-center text-white">
                <p class="mb-2">Error: Could not connect to the backend service.</p>
                <p>Please try restarting the application.</p>
            </div>
            <button class="w-auto px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                    onclick={handleExit}
            >
                Exit
            </button>
        </div>
    </div>
{:else if !hasConnectedOnce && !showDisconnectDialog && showInitialConnecting}
    <div class="w-full min-h-screen flex items-center justify-center bg-zinc-100 dark:bg-zinc-900 rounded-2xl shadow-lg relative">
        <div class="flex flex-col items-center space-y-13">
            <h1 class="text-2xl font-bold text-zinc-900 dark:text-white">{PUBLIC_APPNAME}</h1>

            <div class="flex items-center">
                <div class="flex items-center justify-center mr-3">
                    <div class="relative h-4 w-4">
                        <div class="absolute inset-0 rounded-full bg-purple-600 opacity-75 animate-ping"></div>
                        <div class="relative rounded-full h-4 w-4 bg-purple-500"></div>
                    </div>
                </div>
                <p class="text-zinc-900 dark:text-white">Connecting to backend...</p>
            </div>

            <button class="w-auto px-4 py-2 bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                    onclick={handleExit}>
                No, thank you.
            </button>
        </div>
    </div>
{/if}