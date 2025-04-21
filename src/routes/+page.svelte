<script lang="ts">
    import '../app.css'; // Import from src/app.css
    import {
        type IPieMenuMessage,
        publishMessage,
        SHORTCUT_DETECTED_EVENT,
        subscribeToTopic,
        WINDOW_OPEN_EVENT,
    } from "$lib/natsAdapter.ts";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalPosition} from "@tauri-apps/api/window";
    import {goto} from "$app/navigation";
    import {StringCodec} from "nats.ws";
    import {invoke} from '@tauri-apps/api/core';


    subscribeToTopic(SHORTCUT_DETECTED_EVENT, message => {
        const messageText = StringCodec().decode(message.data);
        console.log(`Received message on '${message.subject}': ${messageText}`);


        try {
            // Example of extracting values
            const pie_menu_message: IPieMenuMessage = JSON.parse(messageText);
            if (pie_menu_message.shortcutDetected != 0) {
                console.log("Shortcut pressed!");
            }

            if (pie_menu_message.shortcutDetected == 1) {
                goto('/pie_menu/');
            }

        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    })

    onMount(async () => {
        await getCurrentWindow().setPosition(new LogicalPosition(100, 100));
    });

    const openOverlay = async () => {
        try {
            await invoke('create_overlay', {
                position: [0, 0],
                size: [3440, 1440],
                parent: 'main' // Must match the label of the main window
            });
        } catch (e) {
            console.error('Failed to open overlay:', e);
        }
    };
</script>


<main>

    <div class="bg-amber-950 w-screen h-screen flex items-center justify-center">
        <button class="absolute top-4 right-4 bg-amber-200" onclick={
        () => publishMessage(WINDOW_OPEN_EVENT,{
            name:"Peter",
            handle: "myHandle",
            something: 3.14})
            }>
            Publish some message, I guess.
        </button>

        <button onclick={openOverlay} class="absolute bottom-4 right-4">
            Launch Pie Menu Overlay
        </button>


        <div class="text-center">
            <h1 class="mb-4 text-blue-100">Hello and welcome to my MightyPie!</h1>
            <a class="text-blue-700 font-bold" href="/pie_menu">&gt; Open Pie Menu &lt;</a>
        </div>
    </div>
</main>