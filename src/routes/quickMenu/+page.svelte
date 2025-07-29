<script lang="ts">
    import {goto} from "$app/navigation";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {
        PUBLIC_PIEBUTTON_HEIGHT,
        PUBLIC_PIEBUTTON_WIDTH,
        PUBLIC_QUICKMENU_SIZE_X,
        PUBLIC_QUICKMENU_SIZE_Y
    } from "$env/static/public";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import {PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS} from "$env/static/public";
    import QuickMenuPieButton from '$lib/components/quickMenu/QuickMenuPieButton.svelte';
    import {getMenuConfiguration} from '$lib/data/configManager.svelte.ts';
    import type {Button, ButtonsOnPageMap} from '$lib/data/types/pieButtonTypes.ts';
    import {ensureWindowWithinMonitorBounds} from "$lib/components/piemenu/piemenuUtils.ts";
    import {createLogger} from "$lib/logger";

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
        goto('/piemenuConfig');
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

    // --- Quick Menu Favorite Logic ---
    const QUICK_MENU_FAVORITE_KEY = 'quickMenuFavorite';

    function getQuickMenuFavorite() {
        try {
            const raw = localStorage.getItem(QUICK_MENU_FAVORITE_KEY);
            if (!raw) return null;
            return JSON.parse(raw);
        } catch {
            return null;
        }
    }

    let menuID = $state(0);
    let pageID = $state(0);
    const favorite = getQuickMenuFavorite();
    if (favorite && typeof favorite.menuID === 'number' && typeof favorite.pageID === 'number') {
        menuID = favorite.menuID;
        pageID = favorite.pageID;
    }

    const buttonWidth = Number(PUBLIC_PIEBUTTON_WIDTH);
    const buttonHeight = Number(PUBLIC_PIEBUTTON_HEIGHT);
    const menuConfig = getMenuConfiguration();
    let buttonsOnPage: ButtonsOnPageMap | undefined = $state(undefined);
    let buttonList: [number, Button][] = $state([]);

    $effect(() => {
        buttonsOnPage = menuConfig.get(menuID)?.get(pageID);
        buttonList = buttonsOnPage ? Array.from(buttonsOnPage.entries()) : [];
    });

    onMount(() => {
        logger.info('Quick Menu Mounted');

        isDark = document.documentElement.classList.contains('dark');
        const initialize = async () => {
            const window = getCurrentWindow();
            await window.setFocus();
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

<div class="w-full min-h-screen flex flex-col items-center justify-center bg-white dark:bg-zinc-900 rounded-2xl shadow-lg relative">
    <button
            aria-label="Toggle dark mode"
            class="absolute top-4 right-4 py-2 px-2 rounded bg-purple-500 text-white hover:bg-purple-600 dark:bg-purple-700 dark:hover:bg-purple-800 transition text-base focus:outline-none z-10"
            onclick={toggleDark}
    >
        {isDark ? '🌙 Dark Theme' : '☀️ Light Theme'}
    </button>
    <h1 class="text-center mb-6 font-semibold text-lg text-zinc-900 dark:text-white">Quick Menu</h1>
    <div class="rounded-xl bg-zinc-100 dark:bg-zinc-700 p-2 flex flex-col items-center w-auto h-auto">
        <div class="grid grid-cols-3 gap-4 mx-auto">
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
    <div class="flex flex-col gap-4 items-center px-4 py-4 w-full max-w-xs mx-auto">
        <button class="w-full px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                onclick={navigateToFuzzySearch}>Fuzzy Search
        </button>
        <button class="w-full px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                onclick={() => publishMessage(PUBLIC_NATSSUBJECT_BUTTONMANAGER_FILL_GAPS, {})}>
            Fill unassigned Button gaps
        </button>
        <button class="w-full px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                onclick={navigateToPieMenuConfig}>Pie Menu Config
        </button>
        <button class="w-full px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                onclick={navigateToSettings}>Settings
        </button>
        <button class="w-full px-4 py-2  bg-zinc-200 dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 rounded-lg text-zinc-700 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-600 transition active:bg-zinc-400 active:dark:bg-zinc-500"
                onclick={async () => { await getCurrentWindow().close(); }}>
            Exit
        </button>
    </div>
</div>

<style>
    /* Remove all previous styles, since Tailwind classes are now used for color, layout, etc. */
</style>
