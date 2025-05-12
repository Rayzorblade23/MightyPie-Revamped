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
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUESTUPDATE,
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_UPDATE
    } from '$env/static/public';
    import type {ConfigData} from '$lib/components/piebutton/piebuttonTypes.ts';
    import {
        parseNestedRawConfig,
        updateProfilesConfiguration
    } from '$lib/components/piebutton/piebuttonConfig.svelte.ts';

    let {children} = $props();
    let displayStatus = $state('Idle');
    let initialUpdateRequestSent = $state(false);


    // Effect to monitor connection status
    $effect(() => {
        const currentStatus = getConnectionStatus();
        displayStatus = currentStatus;
        console.log("NATS connection status:", displayStatus); // Log the actual value/object

        // Corrected comparison (assuming 'connected' is the right string literal):
        if (currentStatus === 'connected' && !initialUpdateRequestSent) {
            console.log('NATS connected, sending initial button manager update request.');
            try {
                publishMessage<{}>(PUBLIC_NATSSUBJECT_BUTTONMANAGER_REQUESTUPDATE, {});
                initialUpdateRequestSent = true;
            } catch (error) {
                console.error('[+layout.svelte] Failed to publish initial button manager update request:', error);
            }
        }

        // Resetting the flag (using the same correct status literal)
        if (currentStatus !== 'connected' && initialUpdateRequestSent) {
            console.log('NATS disconnected/not connected, resetting initial update request flag.');
            initialUpdateRequestSent = false;
        }
    });

    const handleButtonUpdateMessage = async (message: string) => {
        let parsedJsonPayload: any;

        try {
            parsedJsonPayload = JSON.parse(message);
        } catch (parseError) {
            console.error(
                '[+layout.svelte] Failed to parse incoming button manager update JSON:',
                parseError,
                'Raw message:',
                message
            );
            return;
        }

        // Handle cases where the payload is explicitly "null" from the wire.
        if (parsedJsonPayload === null) {
            console.error(
                '[+layout.svelte] Received null as button manager update configuration. Raw message:',
                message
            );
            return;
        }

        // Assuming parsedJsonPayload is valid ConfigData at this point.
        // For enhanced safety, consider runtime validation (e.g., with Zod or a type guard)
        // if the data source is not fully trusted or the ConfigData structure is complex.
        const configData: ConfigData = parsedJsonPayload as ConfigData;

        try {
            const newParsedConfig = parseNestedRawConfig(configData);
            updateProfilesConfiguration(newParsedConfig);
        } catch (applyError) {
            console.error(
                '[+layout.svelte] Failed to apply parsed button manager configuration:',
                applyError,
                'Parsed data:',
                configData
            );
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

</script>


{@render children()}