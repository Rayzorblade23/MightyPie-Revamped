<script lang="ts">
    import {onMount} from "svelte";
    import {browser} from "$app/environment";
    import "../app.css";

    import {
        connectToNats,
        disconnectFromNats,
        getConnectionStatus,
        useNatsSubscription,
    } from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE} from '$env/static/public';
    import type {ConfigData} from '$lib/components/piebutton/piebuttonTypes.ts';
    import {
        parseNestedRawConfig,
        updateProfilesConfiguration
    } from '$lib/components/piebutton/piebuttonConfig.svelte.ts';

    let {children} = $props();
    let displayStatus = $state('Idle');

    $effect(() => {
        displayStatus = getConnectionStatus();
        console.log("NATS connection status:", displayStatus);
    });

    const handleButtonUpdateMessage = async (message: string) => {
        try {
            const configData: ConfigData = JSON.parse(message);
            const newParsedConfig = parseNestedRawConfig(configData);
            updateProfilesConfiguration(newParsedConfig);
        } catch (e) {
            console.error('[+layout.svelte] Failed to parse or apply button manager update:', e);
        }
    };

    const subscription_button_update = useNatsSubscription(
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE,
        handleButtonUpdateMessage
    );

    $effect(() => {
        if (subscription_button_update.error) {
            console.error(
                "[+layout.svelte] Error with button configuration subscription:",
                subscription_button_update.error
            );
            // Consider implementing user-facing error reporting here if appropriate
        }
    });

    onMount(() => {
        let connectionAttempted = false;

        if (browser && !connectionAttempted) {
            connectionAttempted = true;
            const initializeConnection = async () => {
                try {
                    await connectToNats();
                } catch (error) {
                    console.error("Layout onMount: Failed to connect to NATS:", error);
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

</script>


{@render children()}