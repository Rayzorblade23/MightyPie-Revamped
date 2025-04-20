<script lang="ts">
    import '../app.css'; // Import from src/app.css
    import {
        type IPieMenuMessage,
        publishMessage,
        SHORTCUT_DETECTED_EVENT,
        subscribeToTopic,
        WINDOW_OPEN_EVENT,
    } from "$lib/natsAdapter.svelte.ts";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalPosition} from "@tauri-apps/api/window";
    import {goto} from "$app/navigation";
    import {StringCodec} from "nats.ws";


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
</script>

<button class="bg-amber-200" onclick={
        () => publishMessage(WINDOW_OPEN_EVENT,{
            name:"Peter",
            handle: "myHandle",
            something: 3.14
        })
    }>
    Publish some message, I guess.
</button>


<main>
    <div class="bg-amber-950 w-screen h-screen flex items-center justify-center">
        <div class="text-center">
            <h1 class="mb-4 text-blue-100">Hello and welcome to my MightyPie!</h1>
            <a href="/pie_menu" class="text-blue-700 font-bold">&gt; Open Pie Menu &lt;</a>
        </div>
    </div>
</main>