<script lang="ts">
    import '../app.css'; // Import from src/app.css
    import {publishMessage, subscribeToTopic,} from "$lib/natsAdapter.ts";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalPosition} from "@tauri-apps/api/window";
    import {goto} from "$app/navigation";
    import {PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED} from "$env/static/public";


    interface IShortcutPressedMessage {
        shortcutPressed: number;
    }


    onMount(async () => {
        await subscribeToTopic(PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED, message => {
            try {
                const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);

                if (shortcutDetectedMsg.shortcutPressed === 1) {
                    goto('/pie_menu/');
                }
            } catch (e) {
                console.error('Failed to parse message:', e);
            }
        })

        await getCurrentWindow().setPosition(new LogicalPosition(100, 100));
    });

</script>


<main>
    <div class="bg-amber-950 w-screen h-screen flex items-center justify-center">
        <button class="absolute top-4 right-4 bg-amber-200"
                on:click={() => publishMessage<IShortcutPressedMessage>(PUBLIC_NATSSUBJECT_SHORTCUT_PRESSED, {shortcutPressed: 1})}
        >
            Publish some message, I guess.
        </button>


        <div class="text-center">
            <h1 class="mb-4 text-blue-100">Hello and welcome to my MightyPie!</h1>
            <a class="text-blue-700 font-bold" href="/pie_menu">&gt; Open Pie Menu &lt;</a>
        </div>
    </div>
</main>