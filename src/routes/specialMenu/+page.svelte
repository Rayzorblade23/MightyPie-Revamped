<script lang="ts">
    import {goto} from "$app/navigation";
    import {onMount} from "svelte";
    import {getCurrentWindow} from "@tauri-apps/api/window";

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
        const initialize = async () => {
            const window = getCurrentWindow();
            await window.setFocus();
        };

        initialize();

        window.addEventListener('blur', onLostFocus);
        return () => {
            window.removeEventListener('blur', onLostFocus);
        };
    });
</script>

<div class="container">
    <h1>Special Menu</h1>
    <div class="button-grid">
        <button on:click={navigateToSettings}>Go to Settings</button>
        <button on:click={navigateToPieMenuConfig}>Go to PieMenu Config</button>
    </div>
</div>

<style>
    .container {
        min-height: 100vh;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background-color: rgba(51, 51, 51, 0.9);
        border-radius: 20px;
        border: 2px solid #ccc;
        padding: 20px;
        box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
    }

    h1 {
        text-align: center;
        margin-bottom: 20px;
    }

    .button-grid {
        display: flex;
        gap: 10px;
        justify-content: center;
        align-items: center;
        padding: 20px;
    }

    button {
        padding: 10px 20px;
        font-size: 16px;
        cursor: pointer;
        border: 1px solid #ccc;
        border-radius: 5px;
        background-color: #f0f0f0;
        color: #333;
        transition: background-color 0.3s;
    }

    button:hover {
        background-color: #ddd;
    }
</style>