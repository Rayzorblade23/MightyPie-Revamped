<!-- MenuTabs.svelte -->
<script lang="ts">
    import RemovePageButton from "$lib/components/piemenuConfig/buttons/RemovePageButton.svelte";
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

<div class="tabs flex items-center space-x-1 pt-1 border-none bg-zinc-100 dark:bg-zinc-900 rounded-t-lg">
    <div class="flex-1 overflow-x-auto whitespace-nowrap flex flex-nowrap items-center horizontal-scrollbar"
         bind:this={scrollDiv} use:horizontalScroll>
        {#each menuIndices as menuIndex (menuIndex)}
            <button
                    type="button"
                    class="flex items-center gap-2 px-4 py-2 font-semibold text-base transition-colors cursor-pointer focus:outline-none
                {currentSelectedMenuID === menuIndex
                    ? 'rounded-t-lg text-rose-400  border-b-2 border-amber-400 bg-white dark:bg-zinc-800'
                    : 'rounded-t-lg  border-t-1 border-r-1 border-zinc-300 dark:border-zinc-700  bg-zinc-100 dark:bg-zinc-900 text-zinc-800 dark:text-zinc-300 hover:text-amber-400 hover:bg-white hover:border-zinc-300  dark:hover:bg-zinc-800 dark:hover:border-zinc-600 '}"
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
                        buttonClass={`ml-2 p-0.5 bg-zinc-700 hover:bg-rose-500 text-white rounded-full focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 transition-colors${(menuIndices.length <= 1 || disableRemove) ? ' pointer-events-none opacity-50' : ''}`}
                        svgClass="w-3 h-3"
                />
            </button>
        {/each}
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
    <span class="px-4 py-2 text-sm text-zinc-400 dark:text-zinc-500">No Menus Configured</span>
{/if}