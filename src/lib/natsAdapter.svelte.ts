import type {NatsConnection} from 'nats.ws';
import {connect, Events, JSONCodec, StringCodec} from 'nats.ws';

const sc = StringCodec();
const jc = JSONCodec(); // Example if using JSON


// Subjects
export const WINDOW_OPEN_EVENT = "mightyPie.events.window.open";
export const PIE_MENU_OPEN_EVENT = "mightyPie.events.pie_menu.open";

// NATS Server WebSocket URL (replace with your actual URL)
const natsServerUrl = 'ws://localhost:9090'; // Use wss:// for secure
const authToken = '5LQ5V4KWPKGRC2LJ8JQGS';

let natsConnection: NatsConnection | null = null;
let connectionStatus: string = 'Disconnected';
let receivedMessages: string[] = [];
let errorMessage: string = '';

// Reactive variables (using Runes)
let shortcutDetected = $state(0);
let mousePosition = $state<MousePosition>({x: 0, y: 0});

export function getShortcutDetected() {
    return shortcutDetected;
}

export function getOpenAtMousePosition() {
    return mousePosition;
}

async function connectToNats() {
    try {
        connectionStatus = 'Connecting...';
        const nc = await connect({
            servers: [natsServerUrl],
            reconnectTimeWait: 5000,
            maxReconnectAttempts: 5,
            token: authToken,
        });

        natsConnection = nc;
        connectionStatus = `Connected to ${nc.getServer()}`;
        errorMessage = '';
        console.log(`Connected to NATS server: ${nc.getServer()}`);

        // Handle connection status events
        (async () => {
            for await (const status of nc.status()) {
                console.info(`NATS status event: ${status.type}`, status.data);
                switch (status.type) {
                    case Events.Disconnect:
                        connectionStatus = `Disconnected. ${status.data ? `Reason: ${status.data}` : ''}`;
                        break;
                    case Events.Reconnect:
                        connectionStatus = `Reconnected to ${nc.getServer()}`;
                        break;
                    case Events.Error:
                        connectionStatus = `Connection Error`;
                        errorMessage = (status.data as { message?: string })?.message || 'Unknown NATS Error';
                        console.error('NATS Error:', status.data);
                        break;
                }
            }
        })();

        // Subscribe to a topic
        const sub = nc.subscribe(PIE_MENU_OPEN_EVENT);
        (async () => {
            console.log(`Subscribed to ${sub.getSubject()}`);
            for await (const msg of sub) {
                const messageText = sc.decode(msg.data);
                console.log(`Received message on '${msg.subject}': ${messageText}`);

                try {
                    // Example of extracting values
                    const pie_menu_message: IPieMenuMessage = JSON.parse(messageText);
                    console.log(`Pie_menu_message: ${pie_menu_message}`);
                    shortcutDetected = pie_menu_message.shortcutDetected; // Update with Runes
                    if (pie_menu_message.shortcutDetected != 0) {
                        console.log("Shortcut pressed!");
                        mousePosition = pie_menu_message.mousePosition; // Update with Runes
                    }

                } catch (e) {
                    console.error('Failed to parse message:', e);
                }

                receivedMessages = [...receivedMessages, messageText]; // Update array with spread
            }
            console.log(`Subscription to ${sub.getSubject()} closed.`);
        })().catch((err: unknown) => {
            console.error(`Subscription error for ${sub.getSubject()}:`, err);
            errorMessage = `Subscription error: ${(err as Error).message}`;
        });
    } catch (err: unknown) {
        console.error('Failed to connect to NATS:', err);
        connectionStatus = 'Connection Failed';
        errorMessage = (err as Error).message || 'Could not connect to NATS server.';
        natsConnection = null;
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

export interface IMessage {
    name: string;
    handle: string;
    something: number;
}

interface MousePosition {
    x: number;
    y: number;
}

export interface IPieMenuMessage {
    shortcutDetected: number;
    mousePosition: MousePosition;
}

export function publishMessage(subject: string, message: IMessage) {
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