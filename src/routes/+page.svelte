<script lang="ts">
    import '../app.css'; // Import from src/app.css
    import {getShortcutDetected, type IMessage, publishMessage, WINDOW_OPEN_EVENT,} from "$lib/natsAdapter.svelte.ts";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalPosition} from "@tauri-apps/api/window";
    import {goto} from "$app/navigation";


    // import {register, unregister, unregisterAll} from '@tauri-apps/plugin-global-shortcut';
    // import {listen} from '@tauri-apps/api/event';

    const a: IMessage = {
        name: "Peter",
        handle: "myHandle",
        something: 3.14
    }

    $effect(() => {
        if (getShortcutDetected() == 1) {
            goto('/pie_menu/');
        }
    });

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
    {getShortcutDetected()}
</button>


<main>
    <div class="bg-amber-950 w-screen h-screen flex items-center justify-center">
        <div class="text-center">
            <h1 class="mb-4 text-blue-100">Hello and welcome to my MightyPie!</h1>
            <a href="/pie_menu" class="text-blue-700 font-bold">&gt; Open Pie Menu &lt;</a>
        </div>
    </div>
</main>