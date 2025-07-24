import {
    AckPolicy,
    connect,
    type ConnectionOptions,
    DeliverPolicy,
    Events,
    type NatsConnection,
    StringCodec,
    type Subscription
} from 'nats.ws';

import {getPrivateEnvVar} from "$lib/generalUtil.ts"; // Ensure this path is correct for your project

// --- Constants ---
const NATS_LOG_PREFIX = '[NATS]';
const sc = StringCodec();
// const jc = JSONCodec(); // Optional: use if payloads are strictly JSON

// --- Types ---

/** Define possible statuses for NATS subscription_button_click */
type UseNatsSubscriptionStatus = 'idle' | 'subscribing' | 'subscribed' | 'failed' | 'disconnected';

/** Defines the possible states of the NATS connection. */
export type NatsConnectionStatus =
    | 'idle'
    | 'connecting'
    | 'connected'
    | 'reconnecting'
    | 'error'
    | 'closed';

// --- State ---
let connectionStatus = $state<NatsConnectionStatus>('idle');
let natsConnection: NatsConnection | null = null;
let natsConfig: { serverUrl: string; authToken: string } | null = null;

// --- Reactive Status Accessor ---
/**
 * Gets the current reactive NATS connection status.
 * @returns {NatsConnectionStatus} The current connection status.
 */
export function getConnectionStatus(): NatsConnectionStatus {
    return connectionStatus;
}

// --- Configuration Management ---
/**
 * Fetches NATS configuration from environment variables if not already cached.
 * @returns {Promise<{ serverUrl: string, authToken: string }>} The NATS configuration.
 * @throws {Error} If configuration variables are missing.
 */
async function getOrLoadNatsConfig(): Promise<{ serverUrl: string; authToken: string }> {
    if (natsConfig) {
        return natsConfig;
    }
    console.log(`${NATS_LOG_PREFIX} Loading configuration...`);
    const [serverUrl, authToken] = await Promise.all([
        getPrivateEnvVar('NATS_SERVER_URL'),
        getPrivateEnvVar('NATS_AUTH_TOKEN')
    ]);

    if (!serverUrl || !authToken) {
        const missing = [];
        if (!serverUrl) missing.push('NATS_SERVER_URL');
        if (!authToken) missing.push('NATS_AUTH_TOKEN');
        const errorMsg = `${NATS_LOG_PREFIX} Configuration missing: ${missing.join(', ')}.`;
        console.error(errorMsg);
        throw new Error(errorMsg);
    }

    natsConfig = {serverUrl, authToken};
    console.log(`${NATS_LOG_PREFIX} Configuration loaded.`);
    return natsConfig;
}

// --- Connection Management ---

/**
 * Establishes a connection to the NATS server.
 * @throws {Error} If connection fails during the initial attempt.
 */
export async function connectToNats(): Promise<void> {
    // Prevent concurrent connection attempts
    if (
        connectionStatus === 'connected' ||
        connectionStatus === 'connecting' ||
        connectionStatus === 'reconnecting'
    ) {
        console.log(`${NATS_LOG_PREFIX} Connection attempt skipped, status: ${connectionStatus}`);
        return;
    }

    connectionStatus = 'connecting';
    console.log(`${NATS_LOG_PREFIX} Attempting to connect...`);

    const maxRetries = 15; // Increased retries for robustness
    const retryDelay = 1000; // 1-second delay

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
        if (connectionStatus === 'connected') {
            console.log(`${NATS_LOG_PREFIX} Already connected, aborting retry loop.`);
            return;
        }

        try {
            const config = await getOrLoadNatsConfig();
            const connectionOpts: ConnectionOptions = {
                servers: [config.serverUrl], token: config.authToken,
                reconnectTimeWait: 5000, maxReconnectAttempts: -1, // Reconnect indefinitely
                timeout: 10000, name: 'TauriSvelteClient'
            };

            const nc = await connect(connectionOpts);
            natsConnection = nc;
            connectionStatus = 'connected';
            console.log(`${NATS_LOG_PREFIX} Connected to server: ${nc.getServer()} on attempt ${attempt}`);

            // Start monitoring status events
            (async () => {
                const currentNcInstance = nc;
                if (!currentNcInstance || currentNcInstance.isClosed()) {
                    if (natsConnection === currentNcInstance) {
                        natsConnection = null;
                        connectionStatus = 'closed';
                    }
                    return;
                }

                try {
                    for await (const status of currentNcInstance.status()) {
                        if (natsConnection !== currentNcInstance || currentNcInstance.isClosed()) {
                            break;
                        }
                        switch (status.type) {
                            case Events.Disconnect:
                                connectionStatus = 'reconnecting';
                                break;
                            case Events.Reconnect:
                                connectionStatus = 'connected';
                                break;
                            case Events.Error:
                                connectionStatus = 'error';
                                break;
                        }
                    }
                } catch (err) {
                    if (natsConnection === currentNcInstance) {
                        connectionStatus = 'error';
                    }
                } finally {
                    if (currentNcInstance.isClosed()) {
                        if (natsConnection === currentNcInstance) {
                            natsConnection = null;
                            if (connectionStatus !== 'error') {
                                connectionStatus = 'closed';
                            }
                        }
                    }
                }
            })().catch(err => {
                // This catches errors related to *launching* the async IIFE itself or an unhandled crash within.
                console.error(`${NATS_LOG_PREFIX} CRITICAL: NATS status listener failed unexpectedly:`, err);
                // If the listener fails, the connection state is unreliable.
                if (natsConnection === nc) {
                    connectionStatus = 'error';
                }
            });

            return; // Exit the function successfully

        } catch (err) {
            console.warn(`${NATS_LOG_PREFIX} Connection attempt ${attempt} of ${maxRetries} failed:`, err);
            if (attempt === maxRetries) {
                console.error(`${NATS_LOG_PREFIX} All connection attempts failed.`);
                connectionStatus = 'error';
                natsConnection = null;
                throw new Error("Failed to connect to NATS after multiple retries.");
            }
            // Wait before the next retry
            await new Promise(resolve => setTimeout(resolve, retryDelay));
        }
    }
}

/** Gracefully disconnects from the NATS server. */
export async function disconnectFromNats(): Promise<void> {
    const currentConnection = natsConnection;
    if (currentConnection && !currentConnection.isClosed()) {
        console.log(`${NATS_LOG_PREFIX} Draining NATS connection ${currentConnection.getServer()}...`);
        try {
            await currentConnection.drain(); // Wait for drain, then connection closes.
            console.log(`${NATS_LOG_PREFIX} Connection drained successfully for ${currentConnection.getServer()}.`);
            // Listener's finally block should handle setting state to 'closed'.
            // We just ensure the reference is cleaned up if it was the active one.
            if (natsConnection === currentConnection) {
                natsConnection = null;
                // Safeguard state if listener's finally is somehow delayed/missed
                if (connectionStatus !== 'error' && connectionStatus !== 'closed') {
                    console.warn(`${NATS_LOG_PREFIX} Setting status 'closed' post-drain as safeguard.`);
                    connectionStatus = 'closed';
                }
            }
        } catch (err: unknown) {
            console.error(`${NATS_LOG_PREFIX} Error draining NATS connection ${currentConnection.getServer()}:`, err);
            // If drain fails, connection is likely unusable.
            if (natsConnection === currentConnection) {
                connectionStatus = 'error';
                natsConnection = null;
            }
        }
    } else {
        console.log(`${NATS_LOG_PREFIX} Connection already closed or not established.`);
        if (connectionStatus !== 'error' && connectionStatus !== 'closed') {
            connectionStatus = 'closed';
        }
        if (natsConnection) { // If somehow state and object are desynced
            natsConnection = null;
        }
    }
}

/** Checks if the NATS connection is currently established and active. */
export function isNatsConnected(): boolean {
    return connectionStatus === 'connected' && natsConnection != null && !natsConnection.isClosed();
}

// --- Messaging ---

/**
 * Subscribes to a NATS subject if the connection is active.
 * Returns a promise resolving to an unsubscribe function.
 * If not connected, resolves to a no-op unsubscribe function.
 * Throws an error ONLY if the subscription_button_click attempt fails *while connected*.
 */
export async function subscribeToSubject(
    subject: string,
    handleMessage: (decodedMsg: string) => void | Promise<void>
): Promise<() => void> {
    // 1. Handle "Not Connected" case gracefully
    if (!isNatsConnected() || !natsConnection) {
        console.warn(`${NATS_LOG_PREFIX} Subscription skipped for ${subject}: Connection not ready (Status: ${connectionStatus}).`);
        // Return a promise resolving to a function that does nothing
        return Promise.resolve(() => {
            // No-op: console.debug(`${NATS_LOG_PREFIX} No-op unsubscribe called for ${subject} (was not connected).`);
        });
    }

    // At this point, we assume we are connected and natsConnection is not null.
    const currentNatsConnection = natsConnection;
    let subscription: Subscription | null = null;

    try {
        // 2. Attempt the actual subscription_button_click
        subscription = currentNatsConnection.subscribe(subject, {
            callback: (_err, msg) => {
                if (_err) {
                    // Error delivered *to* the subscription_button_click (e.g., permissions)
                    console.error(`${NATS_LOG_PREFIX} Subscription Error CB for ${subject}:`, _err);
                    // Optionally, you might want to trigger an external error handler here
                    return;
                }
                try {
                    const decodedString = sc.decode(msg.data);
                    // Use Promise.resolve to handle both sync/async handlers safely
                    Promise.resolve(handleMessage(decodedString)).catch((handlerError) => {
                        console.error(`${NATS_LOG_PREFIX} HandleMessage Error for ${subject} processing msg:`, handlerError);
                    });
                } catch (decodeError) {
                    console.error(`${NATS_LOG_PREFIX} Decode Error for ${subject}:`, decodeError);
                }
            }
        });

        console.log(`${NATS_LOG_PREFIX} Successfully subscribed to topic: ${subject}`);

        // 3. Return the actual unsubscribe function
        // This specific subscription_button_click instance is captured in the closure.
        const actualSubscription = subscription; // Capture instance for the closure
        return () => {
            if (actualSubscription && !actualSubscription.isClosed()) {
                console.log(`${NATS_LOG_PREFIX} Unsubscribing from ${subject}...`);
                actualSubscription.unsubscribe();
            } else {
                // Optional: Log why unsubscribe isn't happening
                // console.debug(`${NATS_LOG_PREFIX} Unsubscribe skipped for ${subject} (already closed or null).`);
            }
        };

    } catch (err: unknown) {
        // 4. Handle errors during the *initial* subscribe call (e.g., invalid subject)
        console.error(`${NATS_LOG_PREFIX} Failed to subscribe to topic '${subject}':`, err);
        // Re-throw the error so the caller knows the attempt failed.
        throw new Error(`${NATS_LOG_PREFIX} Subscription attempt failed for ${subject}: ${err instanceof Error ? err.message : String(err)}`);
        // Note: We throw here because the *attempt* failed, unlike the "not connected" case where we didn't even attempt.
    }
    // 5. Removed the internal subscription_button_click closure monitoring IIFE for simplicity.
}

/**
 * Manages an ephemeral JetStream consumer for any stream/subject, always delivering the latest message.
 * @param subject - Subject filter
 * @param handler - Handler for the decoded latest message
 * @returns Cleanup function
 */
export async function fetchLatestFromStream(
    subject: string,
    handler: (msg: string) => void | Promise<void>
) {
    if (!isNatsConnected() || !natsConnection) throw new Error("Not connected to NATS");
    const js = natsConnection.jetstream();
    const jsm = await natsConnection.jetstreamManager();
    const stream = "MIGHTYPIE_EVENTS";
    // Always create ephemeral consumer with deliver_policy: 'last'
    const consumerConfig = {
        deliver_policy: DeliverPolicy.Last,
        filter_subject: subject,
        ack_policy: AckPolicy.Explicit,
    };
    const consumerInfo = await jsm.consumers.add(stream, consumerConfig);
    const consumer = await js.consumers.get(stream, consumerInfo.name);
    let cancelled = false;
    const consumeLoop = async () => {
        for await (const msg of await consumer.consume()) {
            if (cancelled) break;
            try {
                await handler(sc.decode(msg.data));
            } catch (e) {
                console.error("[fetchLatestFromStream] Handler error:", e);
            }
            msg.ack();
        }
    };
    await consumeLoop();
    // Cleanup function: cancel loop and delete ephemeral consumer
    return async () => {
        cancelled = true;
        await jsm.consumers.delete(stream, consumerInfo.name);
    };
}

/** Publishes a message to a NATS subject. */
export function publishMessage<T>(subject: string, message: T): void {
    if (!isNatsConnected()) {
        throw new Error(`${NATS_LOG_PREFIX} Cannot publish: Connection not ready (Status: ${connectionStatus}).`);
    }
    if (!natsConnection) throw new Error("Internal NATS error: connection null despite connected status.");

    try {
        const payloadString = JSON.stringify(message);
        const encodedPayload = sc.encode(payloadString);
        natsConnection.publish(subject, encodedPayload);
        console.log("Message sent on subject:", subject);
    } catch (err: unknown) {
        console.error(`${NATS_LOG_PREFIX} Publish error to ${subject}:`, err);
        throw new Error(`${NATS_LOG_PREFIX} Publish failed for ${subject}: ${err instanceof Error ? err.message : String(err)}`);
    }
}

/**
 * A Svelte 5 Rune to manage a NATS subscription reactively based on connection status.
 * Handles subscribing when connected and unsubscribing on cleanup or disconnect/disable.
 * Intended for production use (minimal logging).
 *
 * @param topic - The NATS subject to subscribe to. Can be a reactive rune ($state) or getter function.
 * @param handler - The function to call when a message is received. Handles sync or async functions.
 * @param enabled - Optional. A reactive boolean ($state) or getter function to enable/disable the subscription effect. Defaults to true.
 * @returns An object with reactive, read-only `status` and `error` properties.
 */
export function useNatsSubscription(
    topic: string | (() => string),
    handler: (message: string) => void | Promise<void>,
    enabled: boolean | (() => boolean) = true
) {
    // Internal state exposed via getters
    let status = $state<UseNatsSubscriptionStatus>('idle');
    let error = $state<Error | null>(null);

    // The core effect managing the subscription lifecycle
    $effect(() => {
        // --- Resolve Reactive Inputs ---
        const currentTopic = typeof topic === 'function' ? topic() : topic;
        const isEnabled = typeof enabled === 'function' ? enabled() : enabled;
        const connectionState = getConnectionStatus(); // Primary reactivity trigger

        // --- Handle Disabled State ---
        if (!isEnabled) {
            status = 'idle'; // Reset status when disabled
            error = null;
            // No cleanup needed specifically for this transition *from* enabled,
            // as the cleanup function from the *previous* enabled run would have executed.
            // Implicitly returns undefined (no cleanup function for the disabled state itself).
            return;
        }

        let unsubscribe: (() => void) | null = null;

        // --- Handle Connected State: Attempt Subscription ---
        if (connectionState === 'connected') {
            status = 'subscribing';
            error = null;

            const setupSubscription = async () => {
                try {
                    // Attempt subscription using the potentially improved adapter function
                    unsubscribe = await subscribeToSubject(currentTopic, handler);

                    // --- Post-Subscription Sanity Check ---
                    // Verify connection/enabled status hasn't changed *during* the async subscribe call.
                    // This prevents setting status to 'subscribed' if disconnect happened mid-flight.
                    if (getConnectionStatus() === 'connected' && (typeof enabled === 'function' ? enabled() : enabled)) {
                        status = 'subscribed';
                        // Minimal log: console.debug(`[useNatsSubscription] Subscribed: ${currentTopic}`);
                    } else {
                        // Status changed while subscribing, clean up immediately if subscription succeeded
                        console.warn(`[useNatsSubscription] Status changed during subscription attempt for ${currentTopic}. Cleaning up.`);
                        unsubscribe?.(); // Clean up potential zombie subscription
                        unsubscribe = null; // Prevent cleanup function from running again later
                        // Reflect the *actual* current state
                        status = getConnectionStatus() === 'connected' ? 'idle' : 'disconnected';
                    }
                } catch (err: unknown) {
                    console.error(`[useNatsSubscription] Failed subscription attempt for ${currentTopic}:`, err);
                    error = err instanceof Error ? err : new Error(String(err));
                    status = 'failed';
                    unsubscribe = null; // Ensure cleanup function won't try to unsubscribe
                }
            };

            // Fire off the async setup. Use 'void' to explicitly signal
            // that we are intentionally not awaiting the promise here.
            // Error handling is managed internally within setupSubscription.
            void setupSubscription();

            // --- Return Cleanup Function ---
            // This runs when the effect re-runs (due to dependency changes like
            // connectionState, currentTopic, isEnabled) or when the component unmounts.
            return () => {
                if (unsubscribe) {
                    // Minimal log: console.debug(`[useNatsSubscription] Unsubscribing: ${currentTopic}`);
                    unsubscribe();
                }
            };

        } else {
            // --- Handle Non-Connected State ---
            status = 'disconnected'; // Reflect NATS connection status
            error = null;
            // No subscription attempted, so no cleanup needed for this specific run.
            // Implicitly returns undefined.
            return;
        }
    }); // End of $effect

    // --- Return Read-Only Reactive State ---
    return {
        get status() {
            return status;
        },
        get error() {
            return error;
        }
    };
}
