<!-- MenuTabs.svelte -->
<script lang="ts">
    export let menuIndices: number[];
    export let onSelectMenu: (menuIndex: number) => void;
    export let currentSelectedMenuID: number | undefined;
</script>

<div class="tabs flex space-x-1 border-b border-gray-300 mb-4">
    {#each menuIndices as menuIndex (menuIndex)}
        <button
                on:click={() => onSelectMenu(menuIndex)}
                class="px-4 py-2 -mb-px font-medium text-sm focus:outline-none border-b-2"
                class:border-blue-500={currentSelectedMenuID === menuIndex}
                class:text-blue-600={currentSelectedMenuID === menuIndex}
                class:border-transparent={currentSelectedMenuID !== menuIndex}
                class:text-gray-500={currentSelectedMenuID !== menuIndex}
                class:hover:text-gray-700={currentSelectedMenuID !== menuIndex}
                class:hover:border-gray-400={currentSelectedMenuID !== menuIndex}
        >
            Menu {menuIndex + 1}
        </button>
    {/each}
    {#if menuIndices.length === 0}
        <span class="px-4 py-2 text-sm text-gray-400">No Menus Configured</span>
    {/if}
    <!-- You might want an "Add Menu" button here -->
</div>