import {connect, type ConnectionOptions, Events, type NatsConnection, StringCodec, type Subscription} from 'nats.ws';

import {getPrivateEnvVar} from "$lib/generalUtil.ts"; // Ensure this path is correct for your project
import {createLogger} from "$lib/logger";
import {PUBLIC_NATS_STREAM} from "$env/static/public";

// --- Constants ---
const logger = createLogger('NATS');
const sc = StringCodec();
// const jc = JSONCodec(); // Optional: use if payloads are strictly JSON

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
    logger.debug('Loading configuration...');
    const [serverUrl, authToken] = await Promise.all([
        getPrivateEnvVar('NATS_SERVER_URL'),
        getPrivateEnvVar('NATS_AUTH_TOKEN')
    ]);

    if (!serverUrl || !authToken) {
        const missing = [];
        if (!serverUrl) missing.push('NATS_SERVER_URL');
        if (!authToken) missing.push('NATS_AUTH_TOKEN');
        const errorMsg = `Configuration missing: ${missing.join(', ')}.`;
        logger.error(errorMsg);
        throw new Error(errorMsg);
    }

    natsConfig = {serverUrl, authToken};
    logger.debug('Configuration loaded.');
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
        logger.debug(`Connection attempt skipped, status: ${connectionStatus}`);
        return;
    }

    connectionStatus = 'connecting';
    logger.info('Attempting to connect...');

    const maxRetries = 15; // Increased retries for robustness
    const retryDelay = 1000; // 1-second delay

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
        if (connectionStatus === 'connected') {
            logger.debug('Already connected, aborting retry loop.');
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
            logger.info(`Connected to server: ${nc.getServer()} on attempt ${attempt}`);

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
                logger.error('CRITICAL: NATS status listener failed unexpectedly:', err);
                // If the listener fails, the connection state is unreliable.
                if (natsConnection === nc) {
                    connectionStatus = 'error';
                }
            });

            return; // Exit the function successfully

        } catch (err) {
            logger.warn(`Connection attempt ${attempt} of ${maxRetries} failed:`, err);
            if (attempt === maxRetries) {
                logger.error('All connection attempts failed.');
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
        logger.info('Draining NATS connection...');
        try {
            await currentConnection.drain(); // Wait for drain, then connection closes.
            logger.info('Connection drained successfully.');
            // Listener's finally block should handle setting state to 'closed'.
            // We just ensure the reference is cleaned up if it was the active one.
            if (natsConnection === currentConnection) {
                natsConnection = null;
                // Safeguard state if listener's finally is somehow delayed/missed
                if (connectionStatus !== 'error' && connectionStatus !== 'closed') {
                    logger.warn('Setting status \'closed\' post-drain as safeguard.');
                    connectionStatus = 'closed';
                }
            }
        } catch (err: unknown) {
            logger.error('Error draining NATS connection:', err);
            // If drain fails, connection is likely unusable.
            if (natsConnection === currentConnection) {
                connectionStatus = 'error';
                natsConnection = null;
            }
        }
    } else {
        logger.debug('Connection already closed or not established.');
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
        logger.warn(`Subscription skipped for ${subject}: Connection not ready (Status: ${connectionStatus}).`);
        // Return a promise resolving to a function that does nothing
        return Promise.resolve(() => {
            // No-op
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
                    logger.error(`Subscription Error CB for ${subject}:`, _err);
                    // Optionally, you might want to trigger an external error handler here
                    return;
                }
                try {
                    const decodedString = sc.decode(msg.data);
                    // Use Promise.resolve to handle both sync/async handlers safely
                    Promise.resolve(handleMessage(decodedString)).catch((handlerError) => {
                        logger.error(`HandleMessage Error for ${subject} processing msg:`, handlerError);
                    });
                } catch (decodeError) {
                    logger.error(`Decode Error for ${subject}:`, decodeError);
                }
            }
        });

        logger.debug(`Successfully subscribed to topic: ${subject}`);

        // 3. Return the actual unsubscribe function
        // This specific subscription_button_click instance is captured in the closure.
        const actualSubscription = subscription; // Capture instance for the closure
        return () => {
            if (actualSubscription && !actualSubscription.isClosed()) {
                logger.debug(`Unsubscribing from ${subject}...`);
                actualSubscription.unsubscribe();
            }
        };

    } catch (err: unknown) {
        // 4. Handle errors during the *initial* subscribe call (e.g., invalid subject)
        logger.error(`Failed to subscribe to topic '${subject}':`, err);
        // Re-throw the error so the caller knows the attempt failed.
        throw new Error(`Subscription attempt failed for ${subject}: ${err instanceof Error ? err.message : String(err)}`);
        // Note: We throw here because the *attempt* failed, unlike the "not connected" case where we didn't even attempt.
    }
}

/**
 * Fetches the latest snapshot from JetStream for a subject, then subscribes to live updates
 * using a core NATS subscription. Relies on the caller's reactive effect to re-run on reconnects.
 */
export async function fetchLatestFromStream(
    subject: string,
    handler: (msg: string) => void | Promise<void>
) {
    // If not connected, no-op cleanup to satisfy caller cleanup expectations
    if (!isNatsConnected() || !natsConnection) {
        logger.warn(`fetchLatestFromStream skipped for ${subject}: Connection not ready (Status: ${connectionStatus}).`);
        return async () => { /* no-op */
        };
    }

    // 1) Snapshot: try to fetch the last message from JetStream for this subject
    try {
        const jsm = await natsConnection.jetstreamManager();
        try {
            const lastMsg = await jsm.streams.getMessage(PUBLIC_NATS_STREAM, {last_by_subj: subject});
            if (lastMsg) {
                const decoded = sc.decode(lastMsg.data);
                logger.debug(`Fetched latest snapshot for ${subject}`);
                // Fan-out to the provided handler
                await handler(decoded);
            } else {
                logger.debug(`No snapshot found for ${subject}`);
            }
        } catch (e: any) {
            const msg = typeof e?.message === 'string' ? e.message.toLowerCase() : '';
            if (msg.includes('no message found')) {
                logger.debug(`No last message yet for subject: ${subject}`);
            } else {
                logger.warn(`Failed to get last message for subject: ${subject}`, e);
            }
        }
    } catch (e) {
        logger.error(`Error during snapshot retrieval for subject: ${subject}`, e);
    }

    // 2) Live updates: use a normal core subscription which survives reconnects
    const unsubscribe = await subscribeToSubject(subject, handler);
    return async () => {
        try {
            unsubscribe?.();
        } catch (e) {
            logger.error(`Error unsubscribing from ${subject}:`, e);
        }
    };
}

/** Publishes a message to a NATS subject. */
export function publishMessage<T>(subject: string, message: T): void {
    if (!isNatsConnected()) {
        throw new Error(`Cannot publish: Connection not ready (Status: ${connectionStatus}).`);
    }
    if (!natsConnection) throw new Error("Internal NATS error: connection null despite connected status.");

    try {
        const payloadString = JSON.stringify(message);
        const encodedPayload = sc.encode(payloadString);
        natsConnection.publish(subject, encodedPayload);
        logger.debug("Message sent on subject:", subject);
    } catch (err: unknown) {
        logger.error(`Publish error to ${subject}:`, err);
        throw new Error(`Publish failed for ${subject}: ${err instanceof Error ? err.message : String(err)}`);
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

            // Attempt to subscribe
            subscribeToSubject(currentTopic, handler)
                .then((unsub) => {
                    if (unsub) {
                        unsubscribe = unsub;
                        status = 'subscribed';
                    }
                })
                .catch((err) => {
                    status = 'failed';
                    error = err instanceof Error ? err : new Error(String(err));
                    logger.error(`Subscription to ${currentTopic} failed:`, err);
                });
        } else {
            // Not connected, so we're in a waiting state
            status = connectionState === 'error' ? 'failed' : 'idle';
            error = connectionState === 'error'
                ? new Error(`NATS connection in error state: ${connectionState}`)
                : null;
        }

        // --- Cleanup Function ---
        return () => {
            if (unsubscribe) {
                try {
                    unsubscribe();
                } catch (err) {
                    logger.error(`Error during unsubscribe from ${currentTopic}:`, err);
                }
                unsubscribe = null;
            }
        };
    });

    // Return a read-only view of the internal state
    return {
        get status() {
            return status;
        },
        get error() {
            return error;
        }
    };
}
