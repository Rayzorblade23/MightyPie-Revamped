<!-- MenuTabs.svelte -->
<script lang="ts">
    import RemovePageButton from "$lib/components/piemenuConfig/elements/RemovePageButton.svelte";
    import {horizontalScroll} from "$lib/generalUtil.ts";

    // Define prop types
    export interface MenuTabsProps {
        menuIndices: number[];
        onSelectMenu: (menuIndex: number) => void;
        currentSelectedMenuID: number | undefined;
        onRemoveMenu: (menuIndex: number) => void;
        disableRemove?: boolean;
        onAddMenu: () => void;
    }

    // Use the prop types
    const {
        menuIndices,
        onSelectMenu,
        currentSelectedMenuID,
        onRemoveMenu,
        disableRemove = false,
        onAddMenu
    }: MenuTabsProps = $props();

    let scrollDiv = $state(null) as HTMLDivElement | null;
    let isOverflowing = $state(false);

    function checkOverflow() {
        if (!scrollDiv) return;
        isOverflowing = scrollDiv.scrollWidth > scrollDiv.clientWidth + 1;
    }

    $effect(() => {
        queueMicrotask(() => {
            checkOverflow();
            if (!scrollDiv) return;
            const mutationObs = new MutationObserver(checkOverflow);
            mutationObs.observe(scrollDiv, {childList: true, subtree: true});

            return () => {
                mutationObs.disconnect();
            };
        });
    });
</script>

<div class="tabs flex items-center space-x-1 border-b border-gray-300 dark:border-gray-700 mb-4 bg-gray-100 dark:bg-gray-900 rounded-t-lg">
    <div class="flex-1 overflow-x-auto whitespace-nowrap flex flex-nowrap items-center horizontal-scrollbar"
         bind:this={scrollDiv} use:horizontalScroll>
        {#each menuIndices as menuIndex (menuIndex)}
            <button
                    type="button"
                    class="flex items-center gap-2 px-4 py-2 font-semibold text-base border-b-2 border-transparent bg-transparent transition-colors cursor-pointer focus:outline-none
                {currentSelectedMenuID === menuIndex
                    ? 'text-rose-400 border-amber-400 dark:text-rose-400 dark:border-amber-400'
                    : 'rounded-t-lg text-gray-700 dark:text-gray-300 hover:text-amber-400 hover:bg-gray-200 dark:hover:text-amber-400 dark:hover:bg-gray-800'}"
                    onclick={() => onSelectMenu(menuIndex)}
            >
                <span>Menu {menuIndex + 1}</span>
                <RemovePageButton
                        title="Remove Menu"
                        onClick={(e) => {
                        if (menuIndices.length <= 1 || disableRemove) return;
                        e.stopPropagation();
                        onRemoveMenu(menuIndex);
                    }}
                        buttonClass={`ml-2 p-0.5 bg-slate-700 hover:bg-rose-500 text-white rounded-full focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 transition-colors${(menuIndices.length <= 1 || disableRemove) ? ' pointer-events-none opacity-50' : ''}`}
                        svgClass="w-3 h-3"
                />
            </button>
        {/each}
        <div class="mx-2 h-6 border-l border-gray-300 dark:border-gray-700"></div>
        {#if !isOverflowing}
            <button
                    type="button"
                    aria-label="Add Menu"
                    class="m-2 p-1 rounded bg-green-500 hover:bg-green-600 text-white flex items-center justify-center flex-shrink-0"
                    onclick={onAddMenu}
            >
                <img src="/tabler_icons/plus.svg" alt="Add" class="h-4 w-4 filter invert"/>
            </button>
        {/if}
    </div>
    {#if isOverflowing}
        <button
                type="button"
                aria-label="Add Menu"
                class="m-2 p-1 rounded bg-green-500 hover:bg-green-600 text-white flex items-center justify-center flex-shrink-0"
                onclick={onAddMenu}
        >
            <img src="/tabler_icons/plus.svg" alt="Add" class="h-4 w-4 filter invert"/>
        </button>
    {/if}
</div>
{#if menuIndices.length === 0}
    <span class="px-4 py-2 text-sm text-gray-400 dark:text-gray-500">No Menus Configured</span>
{/if}