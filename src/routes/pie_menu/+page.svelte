<script lang="ts">
    import '../../app.css';
    import {onMount} from "svelte";
    import {subscribeToTopic} from "$lib/natsAdapter.ts";
    import {getEnvVar} from "$lib/envHandler.ts";
    import PieMenu from "$lib/components/piemenu/PieMenu.svelte";
    import type {IShortcutPressedMessage} from "$lib/components/piemenu/piemenuTypes.ts";
    import {centerWindowAtCursor} from "$lib/components/piemenu/piemenuUtils.ts";

    let monitorScaleFactor: number = 1;

    subscribeToTopic(getEnvVar("NATSSUBJECT_SHORTCUT_PRESSED"), message => {
        try {
            const shortcutDetectedMsg: IShortcutPressedMessage = JSON.parse(message);

            if (shortcutDetectedMsg.shortcutPressed == 1) {
                centerWindowAtCursor(monitorScaleFactor).then(result => {
                    monitorScaleFactor = result;
                });
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    })

    onMount(async () => {
        monitorScaleFactor = await centerWindowAtCursor(monitorScaleFactor);
        console.log("Pie Menu opened!");
    });
</script>

<main>
    <div class="absolute bg-black/20 border-0 left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2">
        <PieMenu/>
    </div>
</main>