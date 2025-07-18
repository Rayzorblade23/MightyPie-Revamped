<script lang="ts">
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {
        PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE,
        PUBLIC_QUICKMENU_SIZE_X,
        PUBLIC_QUICKMENU_SIZE_Y
    } from "$env/static/public";
    import {ensureWindowWithinMonitorBounds} from "$lib/components/piemenu/piemenuUtils.ts";
    import {getMenuConfiguration} from "$lib/data/configHandler.svelte.ts";
    import type {IPieButtonExecuteMessage} from '$lib/data/piebuttonTypes.ts';
    import {type Button, ButtonType} from "$lib/data/piebuttonTypes.ts";
    import Fuse from "fuse.js";
    // --- NATS Integration ---
    import {publishMessage} from '$lib/natsAdapter.svelte.ts';
    import {goto} from "$app/navigation";

    // --- State
    let search = $state('');
    let results = $state<{ button: Button; menuId: number; pageId: number; buttonId: number }[]>([]);
    let allButtons = $state<{ button: Button; menuId: number; pageId: number; buttonId: number }[]>([]);
    let fuse: Fuse<{ button: Button; menuId: number; pageId: number; buttonId: number }>;
    let selectedIndex = $state(0);
    let lastSearch = $state('');
    let mouseMoved = $state(false);
    let inputEl: HTMLInputElement;

    function onLostFocus() {
        console.log('Window lost focus');
        const window = getCurrentWindow();
        window.hide();  // Hide first
        goto('/');
    }

    function extractButtons() {
        const config = getMenuConfiguration();
        const arr: { button: Button; menuId: number; pageId: number; buttonId: number }[] = [];
        for (const [menuId, pages] of config) {
            for (const [pageId, buttons] of pages) {
                for (const [buttonId, button] of buttons) {
                    if (
                        (button.button_type === ButtonType.ShowProgramWindow ||
                            button.button_type === ButtonType.ShowAnyWindow) &&
                        button.properties.button_text_upper.trim() !== ''
                    ) {
                        arr.push({button, menuId, pageId, buttonId});
                    }
                }
            }
        }
        return arr;
    }

    function handleKeyDown(event: KeyboardEvent) {
        const current = selectedIndex;
        console.log('KEY:', event.key, 'selectedIndex:', current);
        if (event.key === "Escape") {
            onLostFocus();
        } else if (event.key === "ArrowDown") {
            if (results.length > 0) {
                selectedIndex = (current + 1) % results.length;
                event.preventDefault();
            }
        } else if (event.key === "ArrowUp") {
            if (results.length > 0) {
                selectedIndex = (current - 1 + results.length) % results.length;
                event.preventDefault();
            }
        } else if (event.key === "Enter") {
            if (results.length > 0 && results[current]) {
                const {pageId, buttonId, button} = results[current];
                publishButtonClick(pageId, buttonId, button.button_type, button.properties);
                event.preventDefault();
            }
        }
    }

    onMount(() => {
        const initialize = async () => {
            const window = getCurrentWindow();
            await window.setFocus();
            await window.setSize(new LogicalSize(Number(PUBLIC_QUICKMENU_SIZE_X), Number(PUBLIC_QUICKMENU_SIZE_Y)));
            await ensureWindowWithinMonitorBounds();
        };
        initialize();

        window.addEventListener('blur', onLostFocus);

        return () => {
            window.removeEventListener('blur', onLostFocus);
        };
    });

    // --- Reactivity
    $effect(() => {
        if (search !== lastSearch) {
            selectedIndex = 0;
            lastSearch = search;
            mouseMoved = false;
            // Add a one-time mousemove listener to detect real mouse movement
            const onFirstMove = () => {
                mouseMoved = true;
                window.removeEventListener('mousemove', onFirstMove);
            };
            window.addEventListener('mousemove', onFirstMove);
        }
    });

    $effect(() => {
        allButtons = extractButtons();
        fuse = new Fuse(allButtons, {
            keys: ["button.properties.button_text_upper", "button.properties.button_text_lower"],
            threshold: 0.4,
        });
        results = search.trim().length > 0 ? fuse.search(search).map(r => r.item) : [];
    });

    // Always keep input focused when results change
    $effect(() => {
        if (inputEl) inputEl.focus();
    });

    function publishButtonClick(pageID: number, buttonID: number, taskType: ButtonType, properties: any) {
        if (!properties || !taskType) return;
        const message: IPieButtonExecuteMessage = {
            page_index: pageID,
            button_index: buttonID,
            button_type: taskType,
            properties: properties,
            click_type: "left_up"
        };
        publishMessage<IPieButtonExecuteMessage>(PUBLIC_NATSSUBJECT_PIEBUTTON_EXECUTE, message);
    }
</script>

<div class="w-full min-h-screen flex flex-col items-center justify-center bg-white dark:bg-zinc-900 rounded-2xl shadow-lg relative">
    <input
            bind:this={inputEl}
            bind:value={search}
            class="w-96 px-4 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800 text-lg text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-400 mb-4"
            onkeydown={handleKeyDown}
            placeholder="Search..."
            type="text"
    />
    {#if search.trim().length > 0}
        <div class="w-96 bg-zinc-100 dark:bg-zinc-800 rounded shadow p-2 max-h-80 overflow-y-auto horizontal-scrollbar"
             role="listbox">
            {#if results.length === 0}
                <div class="text-zinc-400 text-center py-2">No results found.</div>
            {:else}
                {#each results as {button, menuId, pageId, buttonId}, i}
                    <div
                            class="w-full text-left py-2 px-3 rounded cursor-pointer flex flex-col transition-colors duration-75 {selectedIndex === i ? 'bg-blue-100 dark:bg-blue-900 ring-1 ring-blue-400' : ''}"
                            onmouseenter={() => { if (mouseMoved) selectedIndex = i; }}
                            onmousedown={e => { e.preventDefault(); publishButtonClick(pageId, buttonId, button.button_type, button.properties); }}
                            aria-selected={selectedIndex === i}
                            role="option"
                            tabindex="-1"
                            style="user-select: none;"
                    >
                        <span class="font-semibold dark:text-white">{button.properties.button_text_upper}</span>
                        {#if button.properties.button_text_lower}
                            <span class="text-xs text-zinc-400 mt-0.5">{button.properties.button_text_lower}</span>
                        {/if}
                        <span class="text-xs text-zinc-500">Menu {menuId + 1} / Page {pageId + 1}
                            / Slot {buttonId + 1}</span>
                    </div>
                {/each}
            {/if}
        </div>

    {/if}
</div>
