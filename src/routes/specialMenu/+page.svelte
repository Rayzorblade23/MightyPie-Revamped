<script lang="ts">
    import {goto} from "$app/navigation";
    import {onMount} from "svelte";
    import {getCurrentWindow, LogicalSize} from "@tauri-apps/api/window";
    import {PUBLIC_PIEMENU_SIZE_X, PUBLIC_PIEMENU_SIZE_Y} from "$env/static/public";

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

    function onLostFocus() {
        console.log('Window lost focus');
        const window = getCurrentWindow();
        window.hide();  // Hide first
        goto('/');
    }

    onMount(() => {
        isDark = document.documentElement.classList.contains('dark');
        const initialize = async () => {
            const window = getCurrentWindow();
            await window.setFocus();
            await window.setSize(new LogicalSize(Number(PUBLIC_PIEMENU_SIZE_X), Number(PUBLIC_PIEMENU_SIZE_Y)));
        };

        initialize();

        window.addEventListener('blur', onLostFocus);
        return () => {
            window.removeEventListener('blur', onLostFocus);
        };
    });
</script>

<div class="w-full min-h-screen flex flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
    <div class="w-full max-w-md p-6 rounded-2xl border-2 border-gray-200 dark:border-gray-800 shadow-lg bg-white dark:bg-gray-800">
        <h1 class="text-center mb-6 font-semibold text-lg text-gray-900 dark:text-white">Special Menu</h1>
        <div class="flex gap-4 justify-center items-center py-4">
            <button class="px-4 py-2 bg-slate-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg shadow-sm text-gray-700 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600 transition" onclick={navigateToSettings}>Go to Settings</button>
            <button class="px-4 py-2 bg-slate-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg shadow-sm text-gray-700 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600 transition" onclick={navigateToPieMenuConfig}>Go to PieMenu Config</button>
        </div>
        <!-- Dark mode toggle button -->
        <button
            class="mt-8 px-4 py-2 rounded bg-blue-500 text-white hover:bg-blue-600 dark:bg-blue-700 dark:hover:bg-blue-800 transition text-base focus:outline-none"
            onclick={toggleDark}
            aria-label="Toggle dark mode"
        >
            {isDark ? '🌙 Dark' : '☀️ Light'}
        </button>
    </div>
</div>

<style>
/* Remove all previous styles, since Tailwind classes are now used for color, layout, etc. */
</style>