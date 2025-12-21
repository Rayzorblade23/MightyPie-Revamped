<script lang="ts">
    import {goto} from "$app/navigation";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {WebviewWindow} from "@tauri-apps/api/webviewWindow";
    import {
        PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS,
        PUBLIC_PIEBUTTON_HEIGHT,
        PUBLIC_PIEBUTTON_WIDTH,
        PUBLIC_QUICKMENU_SIZE_X,
        PUBLIC_QUICKMENU_SIZE_Y
    } from "$env/static/public";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import QuickMenuPieButton from '$lib/components/quickMenu/QuickMenuPieButton.svelte';
    import {getLiveButtonConfig, getPieMenuConfig} from '$lib/data/configManager.svelte.ts';
    import type {Button, ButtonsOnPageMap} from '$lib/data/types/pieButtonTypes.ts';
    import {ensureWindowWithinMonitorBounds} from "$lib/components/piemenu/piemenuUtils.ts";
    import {createLogger} from "$lib/logger";
    import {exitApp} from "$lib/generalUtil.ts";

    // Create a logger for this component
    const logger = createLogger('QuickMenu');

    // --- THEME TOGGLE LOGIC ---
    let isDark = $state(false);

    function toggleDark() {
        isDark = !isDark;
        document.documentElement.classList.toggle('dark', isDark);
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
    }

    function navigateToSettings() {
        goto('/settings');
    }

    function navigateToPieMenuConfig() {
        goto('/piemenuConfigEditor');
    }

    function navigateToFuzzySearch() {
        goto('/fuzzySearch');
    }

    function onLostFocus() {
        logger.debug('Window lost focus');
        const window = getCurrentWindow();
        window.hide();  // Hide first
        goto('/');
    }

    async function togglePauseIndicatorWindow() {
        const indicator = await WebviewWindow.getByLabel('shortcut_pause_indicator');
        if (!indicator) return;
        if (await indicator.isVisible()) {
            await indicator.hide();
        } else {
            await indicator.show();
        }
    }

    let menuID = $state(0);
    let pageID = $state(0);

    const buttonWidth = Number(PUBLIC_PIEBUTTON_WIDTH);
    const buttonHeight = Number(PUBLIC_PIEBUTTON_HEIGHT);
    const liveButtonsConfig = getLiveButtonConfig();
    let buttonsOnPage: ButtonsOnPageMap | undefined = $state(undefined);
    let buttonList: [number, Button][] = $state([]);

    $effect(() => {
        buttonsOnPage = liveButtonsConfig.get(menuID)?.get(pageID);
        buttonList = buttonsOnPage ? Array.from(buttonsOnPage.entries()) : [];
    });

    onMount(() => {
        logger.info('Quick Menu Mounted');

        const piemenuConfig = getPieMenuConfig();
        const starred = piemenuConfig?.starred;
        if (starred) {
            menuID = starred.menuID;
            pageID = starred.pageID;
        }

        isDark = document.documentElement.classList.contains('dark');
        const initialize = async () => {
            const window = getCurrentWindow();
            await window.setFocus();
            setTimeout(() => {
                window.setFocus();
            }, 50);
            setTimeout(() => {
                window.setFocus();
            }, 200);
            await window.setSize(new LogicalSize(Number(PUBLIC_QUICKMENU_SIZE_X), Number(PUBLIC_QUICKMENU_SIZE_Y)));
            await ensureWindowWithinMonitorBounds();
        };

        initialize();

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                onLostFocus();
            }
        };
        window.addEventListener("keydown", handleKeyDown);

        window.addEventListener('blur', onLostFocus);
        return () => {
            window.removeEventListener('blur', onLostFocus);
            window.removeEventListener("keydown", handleKeyDown);
        };
    });
</script>

<div class="w-full h-screen p-1">
    <div class="w-full h-full flex flex-col items-center justify-start pt-8 bg-gradient-to-br from-amber-500 to-purple-700 rounded-2xl shadow-[0px_1px_4px_rgba(0,0,0,0.5)] relative">
        <button
                aria-label="Toggle dark mode"
                class="absolute top-4 right-4 py-2 px-2 rounded-lg bg-amber-500 text-white hover:bg-orange-400 dark:bg-purple-900 dark:hover:bg-purple-800 transition text-base focus:outline-none z-10"
                onclick={toggleDark}
        >
            <img
                    alt="theme icon"
                    class="w-5 h-5 invert"
                    src={isDark ? "/tabler_icons/moon.svg" : "/tabler_icons/sun.svg"}
            />
        </button>
        <h1 class="text-center mb-8 font-semibold text-2xl text-white">Quick Menu</h1>
        <div class="rounded-xl bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 p-2 flex flex-col items-center w-auto h-auto">
            <div class="grid grid-cols-3" style="column-gap: 1.1rem; row-gap: 1.1rem;">
                {#each buttonList.slice(0, 4) as [buttonID, button]}
                    <QuickMenuPieButton
                            width={buttonWidth}
                            height={buttonHeight}
                            buttonID={buttonID}
                            buttonTextLower={button?.properties?.button_text_lower ?? ''}
                            buttonTextUpper={button?.properties?.button_text_upper ?? ''}
                            pageID={pageID}
                            properties={button?.properties}
                            taskType={button?.button_type ?? 'empty'}
                    />
                {/each}
                <div class="flex items-center justify-center" style="width:100%;">
                    <img alt="star icon" class="w-full h-full rounded-lg p-2 object-contain bg-transparent dark:invert"
                         src="/tabler_icons/star.svg"
                         style="min-width:{buttonWidth}rem; min-height:{buttonHeight}rem; max-width:{buttonWidth}rem; max-height:{buttonHeight}rem;"/>
                </div>
                {#each buttonList.slice(4, 8) as [buttonID, button]}
                    <QuickMenuPieButton
                            width={buttonWidth}
                            height={buttonHeight}
                            buttonID={buttonID}
                            buttonTextLower={button?.properties?.button_text_lower ?? ''}
                            buttonTextUpper={button?.properties?.button_text_upper ?? ''}
                            pageID={pageID}
                            properties={button?.properties}
                            taskType={button?.button_type ?? 'empty'}
                    />
                {/each}
            </div>
        </div>

        <div class="flex flex-col items-center px-3 py-3 w-auto h-auto bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mt-6">
            <div class="grid grid-cols-4" style="column-gap: 1.1rem; row-gap: 1.1rem;">
                <button class="aspect-square w-full bg-purple-800 dark:bg-purple-950 border border-none rounded-xl flex flex-col items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 group p-2 shadow-md"
                        onclick={() => publishMessage(PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS, {})}>
                    <img alt="icon" class="w-8 h-8 mb-1 opacity-90 invert" src="/tabler_icons/sort-descending-2.svg"/>
                    <span class="text-xs text-zinc-100 opacity-90">Fill gaps</span>
                </button>
                <button class="aspect-square w-full bg-purple-800 dark:bg-purple-950 border border-none rounded-xl flex flex-col items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 group p-2 shadow-md"
                        onclick={navigateToFuzzySearch}>
                    <img alt="icon" class="w-8 h-8 mb-1 opacity-90 invert" src="/tabler_icons/search.svg"/>
                    <span class="text-xs text-zinc-100 opacity-90">Fuzzy Search</span>
                </button>
                <button class="aspect-square w-full bg-purple-800 dark:bg-purple-950 border border-none rounded-xl flex flex-col items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 group p-2 shadow-md"
                        onclick={navigateToPieMenuConfig}>
                    <img alt="icon" class="w-8 h-8 mb-1 opacity-90 invert" src="/tabler_icons/custom_pie-menu.svg"/>
                    <span class="text-xs text-zinc-100 opacity-90">Pie Menu <br>Config</span>
                </button>
                <button class="aspect-square w-full bg-purple-800 dark:bg-purple-950 border border-none rounded-xl flex flex-col items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 group p-2 shadow-md"
                        onclick={navigateToSettings}>
                    <img alt="icon" class="w-8 h-8 mb-1 opacity-90 invert" src="/tabler_icons/settings.svg"/>
                    <span class="text-xs text-zinc-100 opacity-90">Settings</span>
                </button>
            </div>
        </div>
    </div>
    <div class="fixed bottom-6 right-6 z-50">
        <div class="flex items-center gap-2">
            <button class="px-4 py-2 bg-purple-800 border border-none rounded-lg text-zinc-200 hover:bg-violet-800 transition active:bg-violet-900  flex items-center gap-2"
                    onclick={togglePauseIndicatorWindow}>
                Toggle Pause Indicator
            </button>
            <button class="px-4 py-2 bg-purple-800 border border-none rounded-lg text-zinc-200 hover:bg-violet-800 transition active:bg-violet-900  flex items-center gap-2"
                    onclick={async () => { await exitApp(); }}>
                Exit
            </button>
        </div>
    </div>
</div>
