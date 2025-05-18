import {connect, type ConnectionOptions, Events, type NatsConnection, StringCodec, type Subscription} from 'nats.ws';

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

    let nc: NatsConnection | null = null; // Define nc here for broader scope

    try {
        const config = await getOrLoadNatsConfig();
        const connectionOpts: ConnectionOptions = {
            servers: [config.serverUrl], token: config.authToken,
            reconnectTimeWait: 5000, maxReconnectAttempts: 10,
            timeout: 10000, name: 'TauriSvelteClient'
        };

        nc = await connect(connectionOpts);
        natsConnection = nc; // Assign to module variable *before* listener starts
        connectionStatus = 'connected'; // Set status *after* successful connect
        console.log(`${NATS_LOG_PREFIX} Connected to server: ${nc.getServer()}`);

        // --- Start monitoring status events (async IIFE) ---
        (async () => {
            // Capture the specific connection instance for this listener's scope
            const currentNcInstance = nc;

            // Safeguard: Check for immediate closure after connect but before listener loop starts
            if (!currentNcInstance || currentNcInstance.isClosed()) {
                console.warn(`${NATS_LOG_PREFIX} Listener not started; connection ${currentNcInstance?.getServer()} closed prematurely.`);
                if (natsConnection === currentNcInstance) {
                    natsConnection = null;
                    console.log(`${NATS_LOG_PREFIX} Setting status to 'closed' due to premature connection closure.`);
                    connectionStatus = 'closed';
                }
                return; // Exit this IIFE
            }

            let listenerErrorOccurred = false; // Track errors within the listener's try/catch

            try {
                console.log(`${NATS_LOG_PREFIX} Status listener started for ${currentNcInstance.getServer()}.`);

                // Process status events asynchronously
                for await (const status of currentNcInstance.status()) {
                    // Stop processing if this listener's connection is no longer the active one,
                    // or if the connection instance itself is closed.
                    if (natsConnection !== currentNcInstance || currentNcInstance.isClosed()) {
                        console.log(`${NATS_LOG_PREFIX} Listener for ${currentNcInstance.getServer()} stopping; no longer active or connection closed.`);
                        break; // Exit the for...await loop
                    }

                    console.info(`${NATS_LOG_PREFIX} Status event: ${status.type}`, status.data ?? '');

                    // Update reactive state based on event type
                    switch (status.type) {
                        case Events.Disconnect:
                        case 'disconnect': // String fallback
                            connectionStatus = 'reconnecting';
                            break;
                        case Events.Reconnect:
                        case 'reconnect': // String fallback
                            connectionStatus = 'connected';
                            console.log(`${NATS_LOG_PREFIX} Reconnected to server: ${currentNcInstance.getServer()}`);
                            break;
                        case Events.Error:
                        case 'error': // String fallback
                            console.error(`${NATS_LOG_PREFIX} NATS connection error event:`, status.data);
                            connectionStatus = 'error'; // Reflect the error state
                            break;
                        // No 'close' event case; loop completion handles closure.
                        default:
                            console.log(`${NATS_LOG_PREFIX} Unhandled status event type: ${status.type}`);
                    }
                } // End for await...of loop

                console.log(`${NATS_LOG_PREFIX} Status listener loop finished normally for ${currentNcInstance.getServer()}.`);

            } catch (err) { // Catch errors during the listener's execution (e.g., iterating status)
                console.error(`${NATS_LOG_PREFIX} Status listener CRASHED for ${currentNcInstance.getServer()}:`, err);
                listenerErrorOccurred = true; // Mark that an error happened within the listener
                // If the error occurred for the currently active connection, set global state to error
                if (natsConnection === currentNcInstance) {
                    connectionStatus = 'error';
                }
            } finally { // Runs when the loop exits (normally, break, or error)
                console.log(`${NATS_LOG_PREFIX} Status listener finally block for ${currentNcInstance.getServer()}.`);

                // Check if the connection instance is actually closed when the listener stops
                if (currentNcInstance.isClosed()) {
                    console.log(`${NATS_LOG_PREFIX} Connection ${currentNcInstance.getServer()} is closed.`);
                    // Only finalize state if this listener was for the *currently active* connection
                    if (natsConnection === currentNcInstance) {
                        natsConnection = null; // Clean up the global connection reference

                        // Determine final state based on whether an error occurred *during* the listener run
                        const statusBeforeFinally = connectionStatus; // Check state right before this block
                        if (statusBeforeFinally !== 'error' && !listenerErrorOccurred) {
                            // If no error state before and listener didn't crash, connection is cleanly closed
                            console.log(`${NATS_LOG_PREFIX} Setting status to 'closed' in finally.`);
                            connectionStatus = 'closed';
                        } else {
                            // If state was already error OR the listener crashed, keep/set state to error
                            console.log(`${NATS_LOG_PREFIX} Setting/Keeping status as 'error' in finally due to prior error state or listener crash.`);
                            connectionStatus = 'error'; // Ensure error state if listener crashed
                        }
                    } else {
                        // Listener stopped for an old/inactive connection that is now closed. Do nothing to global state.
                        console.log(`${NATS_LOG_PREFIX} Listener finally: Connection ${currentNcInstance.getServer()} closed, but it wasn't the active connection.`);
                    }
                } else {
                    // Listener stopped (e.g., loop break), but connection is NOT closed.
                    console.log(`${NATS_LOG_PREFIX} Listener finally: Connection ${currentNcInstance.getServer()} is NOT closed.`);
                    // If the listener crashed but connection isn't closed, something is weird. Force error state.
                    if (listenerErrorOccurred && natsConnection === currentNcInstance) {
                        console.warn(`${NATS_LOG_PREFIX} Listener crashed but connection not closed? Forcing 'error' state.`);
                        connectionStatus = 'error';
                    }
                    // Otherwise, the state ('connected' or 'reconnecting') likely remains valid.
                }
            } // End finally
        })().catch(err => { // <--- ADDED .catch() HERE for the IIFE call
            // This catches errors related to *launching* the async IIFE itself.
            // It's unlikely but handles the floating promise lint rule.
            console.error(`${NATS_LOG_PREFIX} CRITICAL: Failed to start NATS status listener:`, err);
            // If starting the listener fails, the connection state is unreliable.
            // Check if nc was assigned and is the current connection before setting error.
            if (nc && natsConnection === nc) {
                connectionStatus = 'error';
            }
        });
        // --- End of status listener IIFE ---

    } catch (err: unknown) { // Catches errors from getOrLoadNatsConfig() or connect()
        console.error(`${NATS_LOG_PREFIX} Failed to connect:`, err);
        // Ensure natsConnection is null if connect() threw or nc wasn't assigned
        if (natsConnection === nc) {
            natsConnection = null;
        }
        connectionStatus = 'error'; // Initial connection failure results in 'error' state
        // Rethrow so the caller knows connection failed
        throw new Error(
            `${NATS_LOG_PREFIX} Connection failed: ${err instanceof Error ? err.message : String(err)}`
        );
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
