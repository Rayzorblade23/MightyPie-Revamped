<script lang="ts">
    import {onMount} from "svelte";
    import {browser} from "$app/environment";
    import "../app.css";

    import {
        connectToNats,
        disconnectFromNats,
        getConnectionStatus,
        publishMessage,
        useNatsSubscription,
    } from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_BASECONFIG,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_UPDATE,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_REQUEST_UPDATE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO,
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_REQUEST_INSTALLEDAPPSINFO
    } from '$env/static/public';
    import type {ConfigData} from '$lib/data/piebuttonTypes.ts';
    import {
        parseButtonConfig,
        updateBaseMenuConfiguration,
        updateMenuConfiguration
    } from '$lib/data/configHandler.svelte.ts';
    import {parseInstalledAppsInfo, updateInstalledAppsInfo} from "$lib/data/installedAppsInfoManager.svelte.ts";
    import {parseShortcutLabelsMessage, updateShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';

    let {children} = $props();
    let displayStatus = $state('Idle');

    let initialRequestsSent = $state(false);

    function sendInitialRequests() {
        if (!initialRequestsSent) {
            console.log("NATS connected, sending initial requests for buttonManager and windowManager.");
            try {
                publishMessage<{}>(PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_UPDATE, {});
                publishMessage<{}>(PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUEST_BASECONFIG, {});
                publishMessage<{}>(PUBLIC_NATSSUBJECT_WINDOWMANAGER_REQUEST_INSTALLEDAPPSINFO, {});
                publishMessage<{}>(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_REQUEST_UPDATE, {});
                initialRequestsSent = true;
            } catch (error) {
                console.error("[+layout.svelte] Failed to publish initial requests:", error);
            }
        }
    }

    function resetRequestFlag() {
        console.log("NATS disconnected/not connected, resetting initial request flag.");
        initialRequestsSent = false;
    }

    $effect(() => {
        const currentStatus = getConnectionStatus();
        displayStatus = currentStatus;
        console.log("NATS connection status:", displayStatus);

        if (currentStatus === "connected") {
            sendInitialRequests();
        } else {
            resetRequestFlag();
        }
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

    const subscription_button_update = useNatsSubscription(
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
        handleButtonUpdateMessage
    );

    const subscription_button_baseconfig = useNatsSubscription(
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_BASECONFIG,
        handleBaseConfigUpdateMessage
    );

    const subscription_installed_apps = useNatsSubscription(
        PUBLIC_NATSSUBJECT_WINDOWMANAGER_INSTALLEDAPPSINFO,
        handleInstalledAppsMessage
    );

    const subscription_shortcutsetter_update = useNatsSubscription(
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE,
        handleShortcutLabelsUpdateMessage
    );

    $effect(() => {
        if (subscription_button_baseconfig.error) {
            console.error(
                "[+layout.svelte] Error with base configuration subscription:",
                subscription_button_baseconfig.error
            );
        }
    });

    $effect(() => {
        if (subscription_button_update.error) {
            console.error(
                "[+layout.svelte] Error with button configuration subscription:",
                subscription_button_update.error
            );
        }
    });

    $effect(() => {
        if (subscription_installed_apps.error) {
            console.error(
                "[+layout.svelte] Error with installed apps subscription:",
                subscription_installed_apps.error
            );
        }
    });

    $effect(() => {
        if (subscription_shortcutsetter_update.error) {
            console.error(
                '[+layout.svelte] Error with shortcut labels subscription:',
                subscription_shortcutsetter_update.error
            );
        }
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