<script lang="ts">
    import {
        getBaseMenuConfiguration,
        parseButtonConfig,
        updateBaseMenuConfiguration,
        updateMenuConfiguration
    } from '$lib/data/configHandler.svelte.ts';
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
        getConnectionStatus,
        manageJetStreamConsumer
    } from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
        PUBLIC_NATSSUBJECT_SETTINGS_UPDATE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO
    } from '$env/static/public';
    import type {ConfigData} from '$lib/data/piebuttonTypes.ts';
    import {parseShortcutLabelsMessage, updateShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';
    import {goto} from '$app/navigation';
    import {listen} from '@tauri-apps/api/event';
    import {getSettings, type SettingsMap, updateSettings} from '$lib/data/settingsHandler.svelte.ts';
    import {saturateHexColor} from "$lib/colorUtils.ts";

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
    let displayStatus = $state('Idle');

    $effect(() => {
        displayStatus = getConnectionStatus();
        console.log("NATS connection status:", displayStatus);
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
        console.log("Received base config update message:", message);
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
            // console.log("Received installed apps list:", installedAppsInfo);
            updateInstalledAppsInfo(installedAppsInfo);
        } catch (error) {
            console.error("[+layout.svelte] Failed to process installed apps message:", error);
        }
    };

    const handleShortcutLabelsUpdateMessage = (msg: string) => {
        try {
            const newLabels = parseShortcutLabelsMessage(msg);
            updateShortcutLabels(newLabels);
        } catch (error) {
            console.error("[+layout.svelte] Failed to process shortcut labels message:", error);
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
            colorAccentAnywin: '--color-accent-anywin',
            colorAccentProgramwin: '--color-accent-programwin',
            colorAccentLaunch: '--color-accent-launch',
            colorAccentFunction: '--color-accent-function',
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
                stopButtonUpdate = await manageJetStreamConsumer(
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
                stopBaseConfig = await manageJetStreamConsumer(
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
                stopInstalledApps = await manageJetStreamConsumer(
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
                stopShortcutLabels = await manageJetStreamConsumer(
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
                stopSettingsUpdate = await manageJetStreamConsumer(
                    PUBLIC_NATSSUBJECT_SETTINGS_UPDATE,
                    handleSettingsUpdateMessage
                );
            })();
        }
        return () => stopSettingsUpdate?.();
    });

    onMount(() => {
        if (browser) {
            const initializeConnection = async () => {
                try {
                    console.log("Attempting to connect to NATS...");
                    await connectToNats();
                    // The $effect watching displayStatus will handle sending the request
                } catch (error) {
                    console.error("[+layout.svelte] Failed to connect to NATS:", error);
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
        context: string
    ): void {
        let parsedJsonPayload: T;

        try {
            parsedJsonPayload = JSON.parse(message);
        } catch (parseError) {
            console.error(`[${context}] Failed to parse JSON:`, parseError, 'Raw message:', message);
            return;
        }

        if (parsedJsonPayload === null) {
            console.error(`[${context}] Received null payload. Raw message:`, message);
            return;
        }

        try {
            onSuccess(parsedJsonPayload);
        } catch (applyError) {
            console.error(`[${context}] Failed to process parsed data:`, applyError, 'Parsed data:', parsedJsonPayload);
        }
    }
</script>


{@render children()}