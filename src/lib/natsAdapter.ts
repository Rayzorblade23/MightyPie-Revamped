import type {NatsConnection} from 'nats.ws';
import {connect, Events, JSONCodec, StringCodec} from 'nats.ws';
import type {Msg} from "nats";

const sc = StringCodec();
const jc = JSONCodec(); // Example if using JSON


// Subjects
export const WINDOW_OPEN_EVENT = "mightyPie.events.window.open";
export const SHORTCUT_DETECTED_EVENT = "mightyPie.events.shortcut.detected";

// NATS Server WebSocket URL (replace with your actual URL)
const natsServerUrl = 'ws://localhost:9090'; // Use wss:// for secure
const authToken = '5LQ5V4KWPKGRC2LJ8JQGS';

let natsConnection: NatsConnection | null = null;
let errorMessage: string = '';

async function connectToNats() {
    try {
        const nc = await connect({
            servers: [natsServerUrl],
            reconnectTimeWait: 5000,
            maxReconnectAttempts: 5,
            token: authToken,
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

export async function subscribeToTopic(topic: string, handleMessage: (message: Msg) => void) {
    if (!natsConnection || natsConnection.isClosed()) {
        errorMessage = 'Cannot subscribe: Not connected to NATS.';
        console.error(errorMessage);
        await new Promise(f => setTimeout(f, 1000));
        await subscribeToTopic(topic, handleMessage);
        return;
    }

    try {
        const sub = natsConnection.subscribe(topic);
        console.log(`Subscribed to topic: ${topic}`);

        (async () => {
            for await (const msg of sub) {
                const messageText = sc.decode(msg.data);
                console.log(`Received message on '${msg.subject}': ${messageText}`);
                handleMessage(msg); // Use the provided callback to handle the message
            }
            console.log(`Subscription to ${topic} closed.`);
        })().catch((err: unknown) => {
            console.error(`Subscription error for ${topic}:`, err);
            errorMessage = `Subscription error: ${(err as Error).message}`;
        });
    } catch (err: unknown) {
        console.error(`Failed to subscribe to topic '${topic}':`, err);
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

export interface INatsMessage {
    header: string;
    body: string;
}

interface MousePosition {
    x: number;
    y: number;
}

export interface IPieMenuMessage {
    shortcutDetected: number;
    mousePosition: MousePosition;
}

export function publishMessage(subject: string, message: INatsMessage) {
    if (!natsConnection || natsConnection.isClosed()) {
        errorMessage = 'Cannot publish: Not connected to NATS.';
        return;
    }
    try {
        const payload = JSON.stringify(message);
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