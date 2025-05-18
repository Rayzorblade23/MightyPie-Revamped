<!-- src/routes/piemenuConfig/+page.svelte -->
<script lang="ts">
    // --- Svelte and Third-Party Imports ---
    import {onDestroy, onMount} from 'svelte';
    import {getCurrentWindow, LogicalSize, type Window} from "@tauri-apps/api/window";
    import {type PhysicalSize} from "@tauri-apps/api/dpi";

    // --- Internal Library Imports ---
    import {
        addMenuToMenuConfiguration,
        addPageToMenuConfiguration,
        getBaseMenuConfiguration,
        publishBaseMenuConfiguration,
        removeMenuFromMenuConfiguration,
        removePageFromMenuConfiguration,
        updateBaseMenuConfiguration,
        updateButtonInMenuConfig
    } from '$lib/data/configHandler.svelte.ts';
    import type {Button, ButtonsOnPageMap, MenuConfiguration, PagesInMenuMap} from '$lib/data/piebuttonTypes.ts';
    import {horizontalScroll} from "$lib/generalUtil.ts";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    import {
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_REQUEST_RECORD
    } from "$env/static/public";
    import {getShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';

    // --- Component Imports ---
    import MenuTabs from "$lib/components/piemenuConfig/MenuTabs.svelte";
    import SettingsPieMenu from "$lib/components/piemenuConfig/SettingsPieMenu.svelte";
    import ButtonInfoDisplay from "$lib/components/piemenuConfig/ButtonInfoDisplay.svelte";
    import AddMenuButton from "$lib/components/piemenuConfig/elements/AddMenuButton.svelte";
    import AddPageButton from "$lib/components/piemenuConfig/elements/AddPageButton.svelte";
    import RemoveMenuButton from "$lib/components/piemenuConfig/elements/RemoveMenuButton.svelte";
    import ConfirmationDialog from "$lib/components/ui/ConfirmationDialog.svelte";
    import SetShortcutDialogue from "$lib/components/SetShortcutDialogue.svelte";

    // --- State ---
    let currentWindow: Window | null = null;
    let initialSize: PhysicalSize | null = null;

    let baseMenuConfig = $state<MenuConfiguration>(new Map());
    let selectedMenuID = $state<number | undefined>(undefined);
    let selectedButtonDetails = $state<
        { menuID: number; pageID: number; buttonID: number; slotIndex: number; button: Button } | undefined
    >(undefined);
    let sortedPagesForSelectedMenu = $state<[number, ButtonsOnPageMap][]>([]);
    let showRemoveMenuDialog = $state(false);
    let isShortcutDialogOpen = $state(false);
    let shortcutLabels = $derived(getShortcutLabels());
    let prevShortcutLabels: Record<number, string> | undefined;
    let hasMounted = false;
    let pagesContainer = $state<HTMLDivElement | null>(null);
    let lastTrackedMenuID: number | undefined = undefined;

    // --- Effects ---
    // Close shortcut dialog if shortcut labels change while open
    $effect(() => {
        if (
            isShortcutDialogOpen &&
            prevShortcutLabels &&
            JSON.stringify(shortcutLabels) !== JSON.stringify(prevShortcutLabels)
        ) {
            isShortcutDialogOpen = false;
        }
        prevShortcutLabels = {...shortcutLabels};
    });

    // Populate sortedPagesForSelectedMenu reactively
    $effect(() => {
        const pagesMapForSelectedMenu = pagesForSelectedMenu;
        if (pagesMapForSelectedMenu && pagesMapForSelectedMenu.size > 0) {
            sortedPagesForSelectedMenu = Array.from(pagesMapForSelectedMenu.entries()).sort(
                ([pageID_A], [pageID_B]) => pageID_A - pageID_B
            );
        } else {
            sortedPagesForSelectedMenu = [];
        }
    });

    // Effect: scroll to selected page
    $effect(() => {
        if (!pagesContainer || !selectedButtonDetails) return;
        // Prevent scroll on initial mount
        if (!hasMounted) {
            hasMounted = true;
            return;
        }
        const pageID = selectedButtonDetails.pageID;
        setTimeout(() => {
            if (pagesContainer && (pagesContainer as any).lockMomentum) {
                (pagesContainer as any).lockMomentum();
            }
            requestAnimationFrame(() => {
                const pageEl = pagesContainer!.querySelector(`[data-page-id='${pageID}']`);
                if (pageEl) {
                    const rect = (pageEl as HTMLElement).getBoundingClientRect();
                    const containerRect = pagesContainer!.getBoundingClientRect();
                    if (rect.left < containerRect.left || rect.right > containerRect.right) {
                        (pageEl as HTMLDivElement).scrollIntoView({
                            behavior: 'smooth',
                            block: 'nearest',
                            inline: 'center'
                        });
                    }
                }
            });
        }, 0);
    });

    // Reset horizontal scroll position to the left when switching menu tabs
    $effect(() => {
        if (!pagesContainer) return;
        if (selectedMenuID !== lastTrackedMenuID) {
            pagesContainer.scrollLeft = 0;
            lastTrackedMenuID = selectedMenuID;
        }
    });

    // Auto-select first button of first page when menu changes or on mount (IDs = 0)
    $effect(() => {
        if (selectedMenuID === undefined) return;
        if (selectedButtonDetails && selectedButtonDetails.menuID === selectedMenuID) return;
        const pagesMap = baseMenuConfig.get(selectedMenuID);
        if (!pagesMap || !pagesMap.has(0)) return;
        const buttonsOnPage = pagesMap.get(0);
        if (!buttonsOnPage) return;
        const button = buttonsOnPage.get(0);
        if (!button) return;
        selectedButtonDetails = {
            menuID: selectedMenuID,
            pageID: 0,
            buttonID: 0,
            slotIndex: 0,
            button
        };
    });

    // --- Derived State ---
    const menuIndices = $derived(Array.from(baseMenuConfig.keys()));
    const pagesForSelectedMenu = $derived<PagesInMenuMap | undefined>(
        selectedMenuID !== undefined ? baseMenuConfig.get(selectedMenuID) : undefined
    );

    // --- Lifecycle ---
    onMount(async () => {
        baseMenuConfig = getBaseMenuConfiguration();
        const initialMenuIndices = Array.from(baseMenuConfig.keys());
        if (initialMenuIndices.length > 0 && selectedMenuID === undefined) {
            selectedMenuID = initialMenuIndices[0];
        }
        try {
            currentWindow = getCurrentWindow();
            initialSize = await currentWindow.innerSize();
            const settingsWidth = 1200;
            const settingsHeight = 1000;
            await currentWindow.setSize(new LogicalSize(settingsWidth, settingsHeight));
        } catch (error) {
            console.error("Failed to get/resize window onMount:", error);
        }
    });

    onDestroy(async () => {
        publishBaseMenuConfiguration(baseMenuConfig);
        if (currentWindow && initialSize) {
            try {
                await currentWindow.setSize(initialSize);
            } catch (error) {
                console.error("Failed to restore window size onDestroy:", error);
            }
        }
    });

    // --- Event Handlers ---
    /** Close the shortcut dialog and abort shortcut recording. */
    function closeShortcutDialog() {
        isShortcutDialogOpen = false;
        publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT, {});
    }

    /** Handle menu tab selection. */
    function handleMenuSelect(menuID: number) {
        selectedMenuID = menuID;
        selectedButtonDetails = undefined;
    }

    /** Handle click on a pie button in the preview. */
    function handlePieButtonClick(
        detail: { menuID: number; pageID: number; buttonID: number; slotIndex: number; button: Button }
    ) {
        selectedButtonDetails = detail;
    }

    /** Apply changes to a button's configuration. */
    function handleButtonConfigUpdate(
        payload: { menuID: number; pageID: number; buttonID: number; newButton: Button }
    ) {
        const {menuID, pageID, buttonID, newButton} = payload;
        baseMenuConfig = updateButtonInMenuConfig(baseMenuConfig, menuID, pageID, buttonID, newButton);
        updateBaseMenuConfiguration(baseMenuConfig);
        if (
            selectedButtonDetails &&
            selectedButtonDetails.menuID === menuID &&
            selectedButtonDetails.pageID === pageID &&
            selectedButtonDetails.buttonID === buttonID
        ) {
            selectedButtonDetails = {
                ...selectedButtonDetails,
                button: newButton
            };
        }
    }

    /** Add a new page to the selected menu and scroll the plus button into view. */
    function handleAddPage() {
        if (selectedMenuID === undefined) {
            console.warn("No menu selected to add a page to.");
            return;
        }
        const result = addPageToMenuConfiguration(baseMenuConfig, selectedMenuID);
        if (result) {
            baseMenuConfig = result.newConfig;
            updateBaseMenuConfiguration(baseMenuConfig);
            console.log(`Locally added new page ${result.newPageID} to menu ${selectedMenuID}. UI should reflect this change.`);
            setTimeout(() => {
                if (pagesContainer && (pagesContainer as any).lockMomentum) {
                    (pagesContainer as any).lockMomentum();
                }
                requestAnimationFrame(() => {
                    const plusBtn = pagesContainer?.querySelector('button[data-plus-button]');
                    if (plusBtn) {
                        const rect = (plusBtn as HTMLElement).getBoundingClientRect();
                        const containerRect = pagesContainer!.getBoundingClientRect();
                        if (rect.left < containerRect.left || rect.right > containerRect.right) {
                            (plusBtn as HTMLButtonElement).scrollIntoView({
                                behavior: 'smooth',
                                block: 'nearest',
                                inline: 'center'
                            });
                        }
                    }
                });
            }, 0);
        } else {
            console.error(`Failed to add page to menu ${selectedMenuID}.`);
        }
    }

    /** Remove a page from the selected menu and update selection if needed. */
    function handleRemovePage(menuIDToRemoveFrom: number, pageIDToRemove: number) {
        if (selectedMenuID === undefined || selectedMenuID !== menuIDToRemoveFrom) {
            console.warn("Attempting to remove page from a menu that is not currently selected or invalid state.");
            return;
        }
        const result = removePageFromMenuConfiguration(baseMenuConfig, menuIDToRemoveFrom, pageIDToRemove);
        if (result) {
            baseMenuConfig = result;
            updateBaseMenuConfiguration(result);
            // Adjust selectedButtonDetails if affected by removal and re-indexing
            if (selectedButtonDetails && selectedButtonDetails.menuID === menuIDToRemoveFrom) {
                if (selectedButtonDetails.pageID === pageIDToRemove) {
                    selectedButtonDetails = undefined; // Removed page was selected
                } else if (selectedButtonDetails.pageID > pageIDToRemove) {
                    selectedButtonDetails = {
                        ...selectedButtonDetails,
                        pageID: selectedButtonDetails.pageID - 1
                    };
                }
            }
            console.log(`Locally removed page ${pageIDToRemove} from menu ${menuIDToRemoveFrom}. UI should reflect this change.`);
        } else {
            console.error(`Failed to remove page ${pageIDToRemove} from menu ${menuIDToRemoveFrom}.`);
        }
    }

    /** Add a new menu and select it. */
    function handleAddMenu() {
        const result = addMenuToMenuConfiguration(baseMenuConfig);
        baseMenuConfig = result.newConfig;
        selectedMenuID = result.newMenuID;
    }

    /** Show the remove menu confirmation dialog. */
    function handleRemoveMenu() {
        if (menuIndices.length <= 1) return;
        showRemoveMenuDialog = true;
    }

    /** Confirm removal of the selected menu. */
    function confirmRemoveMenu() {
        if (selectedMenuID === undefined) return;
        const newConfig = removeMenuFromMenuConfiguration(baseMenuConfig, selectedMenuID);
        if (newConfig) {
            baseMenuConfig = newConfig;
            const indices = Array.from(baseMenuConfig.keys());
            selectedMenuID = indices.length > 0 ? indices[0] : undefined;
        }
        showRemoveMenuDialog = false;
    }

    /** Cancel menu removal dialog. */
    function cancelRemoveMenu() {
        showRemoveMenuDialog = false;
    }

    /** Publish a shortcut setter update for the selected menu. */
    function handlePublishShortcutSetterUpdate() {
        if (selectedMenuID !== undefined) {
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_REQUEST_RECORD, {index: selectedMenuID});
            console.log("Published shortcut setter update for menu index:", selectedMenuID);
            isShortcutDialogOpen = true;
        }
    }
</script>

<!-- --- Markup --- -->
<div class="container mx-auto p-4 space-y-6" style="background-color: #f0f0f0;">
    {#if menuIndices.length > 0}
        <!-- --- UI: Menu Tabs --- -->
        <section>
            <MenuTabs
                    menuIndices={menuIndices}
                    currentSelectedMenuID={selectedMenuID}
                    onSelectMenu={handleMenuSelect}
            />
        </section>
        <!-- --- UI: Main Content Area --- -->
        {#if selectedMenuID !== undefined}
            <div class="main-content-area flex flex-col space-y-8">
                <!-- --- UI: Pie Menus Section --- -->
                <section class="pie-menus-section">
                    {#if sortedPagesForSelectedMenu.length > 0}
                        <div
                                class="flex space-x-4 overflow-x-auto py-2 px-1"
                                bind:this={pagesContainer}
                                use:horizontalScroll
                        >
                            <div class="flex flex-row gap-x-6 pb-2">
                                {#each sortedPagesForSelectedMenu as [pageIDOfLoop, buttonsOnPage] (pageIDOfLoop)}
                                    {@const currentMenuIDForCallback = selectedMenuID}
                                    <div class="page-container flex-shrink-0 p-1 border border-gray-200 rounded-lg shadow-sm bg-slate-50"
                                         data-page-id={pageIDOfLoop}>
                                        <SettingsPieMenu
                                                menuID={currentMenuIDForCallback}
                                                pageID={pageIDOfLoop}
                                                buttonsOnPage={buttonsOnPage}
                                                onButtonClick={handlePieButtonClick}
                                                onRemovePage={(removedPageID) => handleRemovePage(currentMenuIDForCallback, removedPageID)}
                                        />
                                    </div>
                                {/each}
                                <AddPageButton onClick={handleAddPage}/>
                            </div>
                        </div>
                    {:else if pagesForSelectedMenu && pagesForSelectedMenu.size === 0 && selectedMenuID !== undefined}
                        <div class="flex flex-col items-center justify-center py-10 text-center">
                            <p class="text-gray-500 mb-4">Menu {selectedMenuID + 1} has no pages.</p>
                            <AddPageButton onClick={handleAddPage}/>
                        </div>
                    {:else if !pagesForSelectedMenu && selectedMenuID !== undefined}
                        <p class="text-gray-500">Loading page data for Menu {selectedMenuID + 1}...</p>
                    {/if}
                </section>
                <!-- --- UI: Button Details & Actions --- -->
                <section class="button-details-section w-full lg:w-3/4 xl:w-1/2 mx-auto flex flex-row items-start gap-4">
                    <div class="flex-1">
                        {#if selectedButtonDetails}
                            <ButtonInfoDisplay
                                    selectedButtonDetails={selectedButtonDetails}
                                    onConfigChange={handleButtonConfigUpdate}
                            />
                        {:else}
                            <div class="p-4 border border-gray-300 rounded-lg shadow text-center text-gray-500">
                                Select a button from a pie menu preview to see its details, or add a page
                                if the menu is
                                empty.
                            </div>
                        {/if}
                    </div>
                    <div class="flex flex-col gap-2 items-end bg-white border border-gray-200 rounded-lg shadow px-4 py-3 min-w-[160px]">
                        <AddMenuButton onClick={handleAddMenu}/>
                        <RemoveMenuButton onClick={handleRemoveMenu} disabled={menuIndices.length <= 1}/>
                        <button
                                class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
                                onclick={handlePublishShortcutSetterUpdate}
                                disabled={selectedMenuID === undefined}
                        >
                            {selectedMenuID !== undefined && shortcutLabels[selectedMenuID]
                                ? shortcutLabels[selectedMenuID]
                                : 'Set Shortcut'}
                        </button>
                    </div>
                </section>
            </div>
            <!-- --- UI: End Main Content Area --- -->
        {:else if selectedMenuID !== undefined}
            <p class="text-gray-500">Loading pages for Menu {selectedMenuID + 1} or menu is empty.</p>
        {/if}
        <!-- --- UI: Empty State --- -->
    {:else}
        <div class="flex flex-col items-center justify-center py-10 text-center">
            <p class="text-gray-500 mb-4">No menus found. Configuration might be loading or empty.</p>
            <!-- Optionally, allow creating the first menu here -->
            <!-- <button on:click={() => handleAddMenu()} class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Add First Menu</button> -->
        </div>
    {/if}
    <!-- --- UI: Dialogs --- -->
    <ConfirmationDialog
        isOpen={showRemoveMenuDialog}
        onConfirm={confirmRemoveMenu}
        onCancel={cancelRemoveMenu}
        message="Are you sure you want to remove this menu? This action cannot be undone."
    />
    <SetShortcutDialogue isOpen={isShortcutDialogOpen} onCancel={closeShortcutDialog}/>
</div>