<script lang="ts">
    import '../app.css'; // Import from src/app.css
    import type {IMessage} from "$lib/natsAdapter.ts";
    import {publishMessage, WINDOW_OPEN_EVENT} from "$lib/natsAdapter.ts";

    import {Button} from "flowbite-svelte";


    // import {register, unregister, unregisterAll} from '@tauri-apps/plugin-global-shortcut';
    // import {listen} from '@tauri-apps/api/event';


    // onMount(() => {
    //     listen('ctrl_n_pressed', () => {
    //         console.log('Ctrl+N was pressed in Rust');
    //     });
    //     // Register the global shortcut after the component is mounted
    //     register('CommandOrControl+Shift+C', () => {
    //         console.log('Shortcut triggered');
    //     });
    //
    //     // Optional: you can add cleanup code for unregistering the shortcut when the component is destroyed
    //     return () => {
    //         // unregister(['CommandOrControl+Shift+C','CommandOrControl+C']);
    //         unregisterAll()
    //         console.log('Unregistered them all!');
    //     };
    // });
    const a: IMessage = {
        name: "Peter",
        handle: "myHandle",
        something: 3.14
    }
</script>

<Button on:click={
        () => publishMessage(WINDOW_OPEN_EVENT,{
            name:"Peter",
            handle: "myHandle",
            something: 3.14
        })
    }>
    Publish
</Button>


<main>
    <div class="bg-primary-500 w-screen h-screen flex items-center justify-center">
        <div class="text-center">
            <h1 class="mb-4 text-blue-100">Hello and welcome to my MightyPie!</h1>
            <a href="/pie_menu" class="text-blue-700 font-bold">&gt; Open Pie Menu &lt;</a>
        </div>
    </div>
</main>