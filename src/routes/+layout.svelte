<script lang="ts">
    import {
        getBaseMenuConfiguration,
        parseButtonConfig,
        updateBaseMenuConfiguration,
        updateMenuConfiguration
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
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
        PUBLIC_NATSSUBJECT_SETTINGS_UPDATE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO
    } from '$env/static/public';
    import type {ConfigData} from '$lib/data/types/pieButtonTypes.ts';
    import {parseShortcutLabelsMessage, updateShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';
    import {goto} from '$app/navigation';
    import {listen} from '@tauri-apps/api/event';
    import {getSettings, type SettingsMap, updateSettings} from '$lib/data/settingsManager.svelte.ts';
    import {saturateHexColor} from "$lib/colorUtils.ts";
    import {createLogger} from "$lib/logger";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils";
    import {getCurrentWindow} from '@tauri-apps/api/window';
    import {exitApp} from "$lib/generalUtil.ts";

    // Create a logger for this component
    const logger = createLogger('Layout');

    let validationHasRun = false;
    $effect(() => {
        const baseMenuConfiguration = getBaseMenuConfiguration();
        const apps = getInstalledAppsInfo();
        
        if (!validationHasRun && baseMenuConfiguration.size > 0 && apps.size > 0) {
            validateAndSyncConfig();
            validationHasRun = true;
        }
    });

    let {children} = $props();
    let connectionStatus = $state('Idle');

    $effect(() => {
        connectionStatus = getConnectionStatus();
        logger.debug("NATS connection status:", connectionStatus);
    });

    const handleButtonUpdateMessage = (message: string) => {
        handleJsonMessage<ConfigData>(
            message,
            (configData) => {
                const newParsedConfig = parseButtonConfig(configData);
                updateMenuConfiguration(newParsedConfig);
            },
            '+layout.svelte: Button Update'
        );
    };

    const handleBaseConfigUpdateMessage = (message: string) => {
        handleJsonMessage<ConfigData>(
            message,
            (configData) => {
                const newParsedConfig = parseButtonConfig(configData);
                updateBaseMenuConfiguration(newParsedConfig);
            },
            '+layout.svelte: Base Config Update'
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

    const handleShortcutLabelsUpdateMessage = (msg: string) => {
        try {
            const newLabels = parseShortcutLabelsMessage(msg);
            updateShortcutLabels(newLabels);
        } catch (error) {
            logger.error("[+layout.svelte] Failed to process shortcut labels message:", error);
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
        let stopButtonUpdate: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopButtonUpdate = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
                    handleButtonUpdateMessage
                );
            })();
        }
        return () => stopButtonUpdate?.();
    });

    $effect(() => {
        let stopBaseConfig: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopBaseConfig = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG,
                    handleBaseConfigUpdateMessage
                );
            })();
        }
        return () => stopBaseConfig?.();
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

    $effect(() => {
        let stopShortcutLabels: (() => void) | null = null;
        if (getConnectionStatus() === "connected") {
            (async () => {
                stopShortcutLabels = await fetchLatestFromStream(
                    PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
                    handleShortcutLabelsUpdateMessage
                );
            })();
        }
        return () => stopShortcutLabels?.();
    });

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

    onMount(() => {
        if (browser) {
            const initializeConnection = async () => {
                try {
                    // Center the window on startup
                    await centerAndSizeWindowOnMonitor(getCurrentWindow(), 400, 300);

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
        listen('show-piemenuconfig', () => goto('/piemenuConfig')).then(unlisten => {
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

{#if connectionStatus === "connected"}
    {@render children()}
{:else if connectionStatus === 'error'}
    <div class="w-full min-h-screen flex items-center justify-center bg-zinc-100 dark:bg-zinc-900 rounded-2xl shadow-lg relative">
        <div class="flex flex-col items-center space-y-9">
            <h1 class="text-2xl font-bold text-zinc-900 dark:text-white">{PUBLIC_APPNAME}</h1>
            <div class="bg-red-800 p-4 rounded-lg max-w-md text-center text-white">
                <p class="mb-2">Error: Could not connect to the backend service.</p>
                <p>Please try restarting the application.</p>
            </div>
            <button class="w-auto px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                    onclick={handleExit}
            >
                No, thank you.
            </button>
        </div>
    </div>
{:else}
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