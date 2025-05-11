import type {NatsConnection} from 'nats.ws';
import {connect, Events, JSONCodec, StringCodec} from 'nats.ws';
import {getPrivateEnvVar} from "$lib/env.ts";

const sc = StringCodec();
const jc = JSONCodec(); // Example if using JSON


// NATS Server WebSocket URL (replace with your actual URL)
const natsServerUrl = await getPrivateEnvVar('NATS_SERVER_URL');
const natsAuthToken = await getPrivateEnvVar('NATS_AUTH_TOKEN');

let natsConnection: NatsConnection | null = null;
let errorMessage: string = '';

async function connectToNats() {
    try {
        const nc = await connect({
            servers: [natsServerUrl],
            reconnectTimeWait: 5000,
            maxReconnectAttempts: 5,
            token: natsAuthToken,
        });

        natsConnection = nc;
        errorMessage = '';
        console.log(`Connected to NATS server: ${nc.getServer()}`);

        // Handle connection status events
        let promise = (async () => {
            for await (const status of nc.status()) {
                console.info(`NATS status event: ${status.type}`, status.data);
                switch (status.type) {
                    case Events.Disconnect:
                        break;
                    case Events.Reconnect:
                        break;
                    case Events.Error:
                        errorMessage = (status.data as { message?: string })?.message || 'Unknown NATS Error';
                        console.error('NATS Error:', status.data);
                        break;
                }
            }
        })();

    } catch (err: unknown) {
        console.error('Failed to connect to NATS:', err);
        errorMessage = (err as Error).message || 'Could not connect to NATS server.';
        natsConnection = null;
    }
}

export async function subscribeToTopic(
    subject: string,
    handleMessage: (decodedMsg: string) => void) {
    if (!natsConnection || natsConnection.isClosed()) {
        errorMessage = 'Cannot subscribe: Not connected to NATS.';
        console.error(errorMessage);
        await new Promise(f => setTimeout(f, 1000));
        await subscribeToTopic(subject, handleMessage);
        return;
    }

    try {
        const sub = natsConnection.subscribe(subject);
        console.log(`Subscribed to topic: ${subject}`);

        (async () => {
            for await (const msg of sub) {
                const decodedString = sc.decode(msg.data);
                console.log(`Received message on '${msg.subject}': ${decodedString}`);
                handleMessage(decodedString);
            }
            console.log(`Subscription to ${subject} closed.`);
        })().catch((err: unknown) => {
            console.error(`Subscription error for ${subject}:`, err);
            errorMessage = `Subscription error: ${(err as Error).message}`;
        });
    } catch (err: unknown) {
        console.error(`Failed to subscribe to topic '${subject}':`, err);
        errorMessage = `Failed to subscribe: ${(err as Error).message}`;
    }
}

function closeNatsConnection() {
    if (natsConnection) {
        console.log('Closing NATS connection...');
        natsConnection
            .drain()
            .then(() => console.log('NATS connection drained.'))
            .catch((err: unknown) => console.error('Error draining NATS connection:', err))
            .finally(() => {
                if (!natsConnection?.isClosed()) {
                    natsConnection?.close();
                }
                console.log('NATS connection closed.');
            });
    }
}


export function publishMessage<T>(subject: string, message: T) {
    if (!natsConnection || natsConnection.isClosed()) {
        errorMessage = 'Cannot publish: Not connected to NATS.';
        return;
    }

    try {
        // Ensure message is JSON stringified
        const payload = JSON.stringify(message);
        // Publish the message using the StringCodec
        natsConnection.publish(subject, sc.encode(payload));
        console.log(`Published to ${subject}: ${payload}`);
        errorMessage = '';
    } catch (err: unknown) {
        console.error('Publish error:', err);
        errorMessage = `Publish error: ${(err as Error).message}`;
    }
}


// Automatically connect to NATS when this module is loaded
connectToNats();

// Ensure the connection is closed when the module is unloaded
export function cleanup() {
    closeNatsConnection();
}