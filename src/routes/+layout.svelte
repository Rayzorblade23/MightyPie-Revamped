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
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO
    } from '$env/static/public';
    import type {ConfigData} from '$lib/data/piebuttonTypes.ts';
    import {parseShortcutLabelsMessage, updateShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';

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
            console.log("Received installed apps list:", installedAppsInfo);
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