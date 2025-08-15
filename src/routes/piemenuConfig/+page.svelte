<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<!-- src/routes/piemenuConfig/+page.svelte -->
<script lang="ts">
    // --- Svelte and Third-Party Imports ---
    import {onDestroy, onMount} from 'svelte';
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
    import {open} from '@tauri-apps/plugin-dialog';

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
    } from '$lib/data/configManager.svelte.ts';
    import type {Button, ButtonsOnPageMap, MenuConfiguration, PagesInMenuMap} from '$lib/data/types/pieButtonTypes.ts';
    import {ButtonType} from '$lib/data/types/pieButtonTypes.ts';
    import {horizontalScroll} from "$lib/generalUtil.ts";
    import {publishMessage} from "$lib/natsAdapter.svelte.ts";
    // --- Function Definitions for CallFunction Buttons ---
    import {
        PUBLIC_CONFIG_SIZE_X,
        PUBLIC_CONFIG_SIZE_Y,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_DELETE
    } from "$env/static/public";
    import {getShortcutLabels} from '$lib/data/shortcutLabelsManager.svelte.ts';
    import {getDefaultButton} from '$lib/data/types/pieButtonDefaults.ts';
    import {getInstalledAppsInfo} from "$lib/data/installedAppsInfoManager.svelte.ts";
    import {createLogger} from "$lib/logger";
    import StandardButton from '$lib/components/StandardButton.svelte';

    // --- Component Imports ---
    import MenuTabs from "$lib/components/piemenuConfig/MenuTabs.svelte";
    import ConfigPieMenuPage from "$lib/components/piemenuConfig/configPieMenuElements/ConfigPieMenuPage.svelte";
    import ButtonInfoDisplay from "$lib/components/piemenuConfig/ButtonInfoDisplay.svelte";
    import AddPageButton from "$lib/components/piemenuConfig/buttons/AddPageButton.svelte";
    import ConfirmationDialog from "$lib/components/ui/ConfirmationDialog.svelte";
    import SetShortcutDialog from "$lib/components/ui/SetShortcutDialog.svelte";
    import ButtonTypeSelector from "$lib/components/piemenuConfig/selectors/ButtonTypeSelector.svelte";
    import {goto} from "$app/navigation";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils.ts";
    import {getButtonFunctions} from "$lib/fileAccessUtils.ts";
    import {invoke} from "@tauri-apps/api/core";
    import NotificationDialog from "$lib/components/ui/NotificationDialog.svelte";

    // Create a logger for this component
    const logger = createLogger('PieMenuConfig');

    interface FunctionDefinition {
        icon_path: string;
        description?: string;
    }

    type AvailableFunctionsMap = Record<string, FunctionDefinition>;

    let availableFunctionsData = $state<AvailableFunctionsMap>({});

    onMount(() => {
        logger.info('Config Mounted');

        getCurrentWindow().show();
    });

    $effect(() => {
        (async () => {
            try {
                // Get the buttonFunctions.json parsed data using the utility function
                availableFunctionsData = await getButtonFunctions<AvailableFunctionsMap>();
            } catch (error) {
                logger.error('Error loading buttonFunctions.json:', error);
            }
        })();
    });

    // --- State ---
    let currentWindow: Window | null = null;

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

    let initialMenuConfigSnapshot: MenuConfiguration | undefined = undefined;

    // --- Undo State ---
    type UndoState = {
        config: MenuConfiguration;
        selectedMenuID: number | undefined;
        selectedButtonDetails: typeof selectedButtonDetails;
    };
    const undoHistory = $state<UndoState[]>([]);
    const UNDO_LIMIT = 20;

    let showDiscardConfirmDialog = $state(false);
    let showResetAllConfirmDialog = $state(false);
    let showBackupCreatedDialog = $state(false);

    // --- Duplicate Shortcut Detection ---
    let shortcutUsage = $derived.by(() => {
        const usage: Record<string, number[]> = {};
        Object.entries(shortcutLabels).forEach(([menuID, shortcut]) => {
            if (!shortcut) return;
            if (!usage[shortcut]) usage[shortcut] = [];
            usage[shortcut].push(Number(menuID));
        });
        return usage;
    });

    function cloneMenuConfig(config: MenuConfiguration): MenuConfiguration {
        return new Map(
            Array.from(config.entries()).map(([menuId, pagesMap]) => [
                menuId,
                new Map(
                    Array.from(pagesMap.entries()).map(([pageId, buttonsMap]) => [
                        pageId,
                        new Map(buttonsMap.entries())
                    ])
                )
            ])
        );
    }

    function pushUndoState() {
        undoHistory.push({
            config: cloneMenuConfig(baseMenuConfig),
            selectedMenuID,
            selectedButtonDetails: selectedButtonDetails ? {...selectedButtonDetails} : undefined
        });
        if (undoHistory.length > UNDO_LIMIT) undoHistory.shift();
    }

    function handleUndo() {
        if (undoHistory.length === 0) return;
        const prev = undoHistory.pop();
        if (!prev) return;
        baseMenuConfig = prev.config;
        selectedMenuID = prev.selectedMenuID;
        selectedButtonDetails = prev.selectedButtonDetails;
        updateBaseMenuConfiguration(baseMenuConfig);
    }

    function discardChanges() {
        if (!initialMenuConfigSnapshot) return;
        baseMenuConfig = cloneMenuConfig(initialMenuConfigSnapshot);
        updateBaseMenuConfiguration(cloneMenuConfig(initialMenuConfigSnapshot));
        // Select first menu if available
        const initialMenuIndices = Array.from(baseMenuConfig.keys());
        if (initialMenuIndices.length > 0) {
            selectedMenuID = initialMenuIndices[0];
        } else {
            selectedMenuID = undefined;
        }
        selectedButtonDetails = undefined; // Let the auto-select $effect handle button selection
        undoHistory.length = 0;
    }

    // --- Reset All Menus ---
    function handleResetAllMenus() {
        pushUndoState();
        const menuKeys = Array.from(baseMenuConfig.keys()).sort((a, b) => a - b);
        if (menuKeys.length === 0) return;
        const firstMenuID = menuKeys[0];
        // Remove shortcuts for all menus except the first
        for (const menuID of menuKeys) {
            if (menuID !== firstMenuID) {
                publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_DELETE, {index: menuID});
            }
        }
        // Reset menu config to only the first menu with one default page
        const newConfig = new Map();
        const defaultButtons = new Map();
        for (let i = 0; i < 8; i++) {
            defaultButtons.set(i, getDefaultButton(ButtonType.ShowAnyWindow));
        }
        const pagesInMenu = new Map();
        pagesInMenu.set(0, defaultButtons);
        newConfig.set(firstMenuID, pagesInMenu);
        baseMenuConfig = newConfig;
        selectedMenuID = firstMenuID;
        selectedButtonDetails = undefined;
        updateBaseMenuConfiguration(baseMenuConfig);
    }

    function confirmResetAllMenus() {
        showResetAllConfirmDialog = false;
        handleResetAllMenus();
    }

    function cancelResetAllMenus() {
        showResetAllConfirmDialog = false;
    }

    // --- Backup handler for Menu Settings
    function handleBackupConfig() {
        // Convert baseMenuConfig (MenuConfiguration) to ConfigData
        const configData: Record<string, any> = {};
        baseMenuConfig.forEach((pagesInMenu, menuId) => {
            const menuKey = menuId.toString();
            configData[menuKey] = {};
            pagesInMenu.forEach((buttonsOnPage, pageId) => {
                const pageKey = pageId.toString();
                configData[menuKey][pageKey] = {};
                buttonsOnPage.forEach((button, buttonId) => {
                    const buttonKey = buttonId.toString();
                    configData[menuKey][pageKey][buttonKey] = {
                        button_type: button.button_type,
                        properties: button.properties,
                    };
                });
            });
        });
        publishMessage(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP, configData);
    }

    function handleBackupWithConfirmation() {
        handleBackupConfig();
        showBackupCreatedDialog = true;
    }

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
                if (!pagesContainer) return;
                const pageEl = pagesContainer.querySelector(`[data-page-id='${pageID}']`);
                if (pageEl) {
                    const rect = (pageEl as HTMLElement).getBoundingClientRect();
                    const containerRect = pagesContainer.getBoundingClientRect();
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
        initialMenuConfigSnapshot = cloneMenuConfig(baseMenuConfig);
        const initialMenuIndices = Array.from(baseMenuConfig.keys());
        if (initialMenuIndices.length > 0 && selectedMenuID === undefined) {
            selectedMenuID = initialMenuIndices[0];
        }
        try {
            currentWindow = getCurrentWindow();
            await centerAndSizeWindowOnMonitor(currentWindow, Number(PUBLIC_CONFIG_SIZE_X), Number(PUBLIC_CONFIG_SIZE_Y));
        } catch (error) {
            logger.error("Failed to get/resize window onMount:", error);
        }
    });

    onMount(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                if (event.defaultPrevented) return;
                const active = document.activeElement;
                // If an input, textarea, select, or contenteditable is focused, first Escape should blur it, second Escape should trigger the normal logic
                if (active && (["INPUT", "TEXTAREA", "SELECT"].includes(active.tagName) || active.getAttribute("contenteditable") === "true")) {
                    (active as HTMLElement).blur();
                    return;
                }
                if (showRemoveMenuDialog || showDiscardConfirmDialog || showResetAllConfirmDialog || isShortcutDialogOpen || showBackupCreatedDialog) return;
                // Use the same logic as Discard Changes button for unsaved changes
                if (undoHistory.length === 0) {
                    goto('/');
                } else {
                    showDiscardConfirmDialog = true;
                }
            }
        };
        window.addEventListener("keydown", handleKeyDown);
        return () => {
            window.removeEventListener("keydown", handleKeyDown);
        };
    });

    onDestroy(async () => {
        publishBaseMenuConfiguration(baseMenuConfig);
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
        pushUndoState();
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

    async function openFileDialog() {
        try {
            logger.debug("Opening file dialog with app data directory path");

            // Get app data directory from Tauri backend
            const configPath = await invoke<string>('get_app_data_dir');
            const selected = await open({multiple: false, defaultPath: configPath});

            if (selected) {
                if (selected.includes('buttonConfig_BACKUP')) {
                    pushUndoState();
                    publishMessage(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP, selected);
                    logger.log('Published selected file path:', selected);
                } else {
                    alert('Please select a file with "buttonConfig_BACKUP" in the name.');
                }
            }
        } catch (error) {
            logger.error(`Error opening file dialog: ${error}`);
        }
    }

    /** Add a new page to the selected menu and scroll the plus button into view. */
    function handleAddPage() {
        pushUndoState();
        if (selectedMenuID === undefined) {
            logger.warn("No menu selected to add a page to.");
            return;
        }
        const result = addPageToMenuConfiguration(baseMenuConfig, selectedMenuID);
        if (result) {
            baseMenuConfig = result.newConfig;
            updateBaseMenuConfiguration(baseMenuConfig);
            logger.log(`Locally added new page ${result.newPageID} to menu ${selectedMenuID}.`);
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
            logger.error(`Failed to add page to menu ${selectedMenuID}.`);
        }
    }

    /** Remove a page from the selected menu and update selection if needed. */
    function handleRemovePage(menuIDToRemoveFrom: number, pageIDToRemove: number) {
        pushUndoState();
        if (selectedMenuID === undefined || selectedMenuID !== menuIDToRemoveFrom) {
            logger.warn("Attempting to remove page from a menu that is not currently selected or invalid state.");
            return;
        }
        const result = removePageFromMenuConfiguration(baseMenuConfig, menuIDToRemoveFrom, pageIDToRemove);
        if (result) {
            baseMenuConfig = result;
            updateBaseMenuConfiguration(result);
            if (selectedButtonDetails && selectedButtonDetails.menuID === menuIDToRemoveFrom) {
                if (selectedButtonDetails.pageID === pageIDToRemove) {
                    selectedButtonDetails = undefined;
                } else if (selectedButtonDetails.pageID > pageIDToRemove) {
                    selectedButtonDetails = {
                        ...selectedButtonDetails,
                        pageID: selectedButtonDetails.pageID - 1
                    };
                }
            }
            logger.log(`Locally removed page ${pageIDToRemove} from menu ${menuIDToRemoveFrom}.`);
        } else {
            logger.error(`Failed to remove page ${pageIDToRemove} from menu ${menuIDToRemoveFrom}.`);
        }
    }

    /** Add a new menu and select it. */
    function handleAddMenu() {
        pushUndoState();
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
        pushUndoState();
        publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_DELETE, {index: selectedMenuID});
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
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE, {index: selectedMenuID});
            logger.log("Published shortcut setter update for menu index:", selectedMenuID);
            isShortcutDialogOpen = true;
        }
    }

    function handleClearShortcut() {
        if (selectedMenuID !== undefined && shortcutLabels[selectedMenuID]) {
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_DELETE, {index: selectedMenuID});
        }
    }

    const buttonTypeFriendlyNames: Record<ButtonType, string> = {
        [ButtonType.ShowProgramWindow]: "Show Program Window",
        [ButtonType.ShowAnyWindow]: "Show Any Window",
        [ButtonType.CallFunction]: "Call Function",
        [ButtonType.LaunchProgram]: "Launch Program",
        [ButtonType.OpenSpecificPieMenuPage]: "Open Page",
        [ButtonType.OpenResource]: "Open Resource",
        [ButtonType.Disabled]: "Disabled",
    };
    const buttonTypeKeys = Object.keys(buttonTypeFriendlyNames) as ButtonType[];

    let resetType = $state<ButtonType>(ButtonType.ShowAnyWindow);

    function handleResetTypeChange(newType: ButtonType) {
        resetType = newType;
    }

    // Handler to reset the active page's buttons to selected type
    function handleResetPageToDefault() {
        pushUndoState();
        if (selectedMenuID === undefined || !sortedPagesForSelectedMenu.length) return;
        let pageID = selectedButtonDetails ? selectedButtonDetails.pageID : sortedPagesForSelectedMenu[0][0];
        if (pageID === undefined) return;
        const numberOfSlots = 8;
        const newButtonsOnPage = new Map();
        const installedAppsMap = getInstalledAppsInfo();
        // Ensure typedAvailableFunctions is available (from ButtonInfoDisplay logic)
        // If not, fallback to empty object
        const typedAvailableFunctions = typeof availableFunctionsData !== 'undefined' ? availableFunctionsData : {};
        for (let i = 0; i < numberOfSlots; i++) {
            let newButton = getDefaultButton(resetType);
            // Match Button Details logic for icon assignment
            if (
                (resetType === ButtonType.ShowProgramWindow || resetType === ButtonType.LaunchProgram) &&
                'properties' in newButton && newButton.properties
            ) {
                const appName = resetType === ButtonType.ShowProgramWindow
                    ? newButton.properties.button_text_lower
                    : newButton.properties.button_text_upper;
                const appInfo = installedAppsMap.get(appName || "");
                if (appInfo) {
                    newButton.properties.icon_path = appInfo.iconPath || "";
                }
            } else if (
                resetType === ButtonType.CallFunction &&
                'properties' in newButton && newButton.properties
            ) {
                const functionName = newButton.properties.button_text_upper;
                const functionInfo = typedAvailableFunctions[functionName || ""];
                if (functionInfo) {
                    newButton.properties.icon_path = functionInfo.icon_path || "";
                }
            }
            newButtonsOnPage.set(i, newButton);
        }
        const newConfig = new Map(baseMenuConfig);
        const menuPages = new Map(newConfig.get(selectedMenuID));
        menuPages.set(pageID, newButtonsOnPage);
        newConfig.set(selectedMenuID, menuPages);
        baseMenuConfig = newConfig;
        updateBaseMenuConfiguration(newConfig);
        publishBaseMenuConfiguration(newConfig);
    }

    // --- Quick Menu Favorite Logic ---
    const QUICK_MENU_FAVORITE_KEY = 'quickMenuFavorite';
    let quickMenuFavoriteVersion = $state(0);

    function getQuickMenuFavorite() {
        try {
            const raw = localStorage.getItem(QUICK_MENU_FAVORITE_KEY);
            if (!raw) return null;
            return JSON.parse(raw);
        } catch {
            return null;
        }
    }

    function setQuickMenuFavorite(menuID: number, pageID: number) {
        localStorage.setItem(QUICK_MENU_FAVORITE_KEY, JSON.stringify({menuID, pageID}));
        quickMenuFavoriteVersion++;
    }

    let isQuickMenuFavorite = $state(false);

    $effect(() => {
        if (selectedMenuID !== undefined && selectedButtonDetails && selectedButtonDetails.pageID !== undefined) {
            const fav = getQuickMenuFavorite();
            isQuickMenuFavorite = !!fav && fav.menuID === selectedMenuID && fav.pageID === selectedButtonDetails.pageID;
        } else {
            isQuickMenuFavorite = false;
        }
    });

    function handleUseForQuickMenu() {
        if (selectedMenuID !== undefined && selectedButtonDetails && selectedButtonDetails.pageID !== undefined) {
            setQuickMenuFavorite(selectedMenuID, selectedButtonDetails.pageID);
            isQuickMenuFavorite = true;
        }
    }

    // Reload the base menu configuration when it changes
    $effect(() => {
        baseMenuConfig = getBaseMenuConfiguration();
    });
</script>

<div class="w-full h-screen flex flex-col rounded-2xl shadow-lg overflow-hidden">
    <!-- --- Title Bar --- -->
    <div class="title-bar relative flex items-center py-1 bg-zinc-200 dark:bg-neutral-800 rounded-t-2xl border-b border-none h-8 flex-shrink-0">
        <div class="w-0.5 min-w-[2px] h-full" data-tauri-drag-region="none"></div>
        <div class="absolute left-0 right-0 top-0 bottom-0 flex items-center justify-center pointer-events-none select-none">
            <span class="font-semibold text-sm lg:text-base text-zinc-900 dark:text-zinc-400">Pie Menu Config</span>
        </div>
        <div class="flex-1 h-full" data-tauri-drag-region></div>
    </div>
    <!-- --- Main Content --- -->
    <div class="flex-1 w-full p-4 overflow-y-auto horizontal-scrollbar relative bg-gradient-to-br from-amber-500 to-purple-700">
        {#if menuIndices.length > 0}
            <!-- --- UI: Menu Tabs --- -->
            <section>
                <MenuTabs
                        menuIndices={menuIndices}
                        currentSelectedMenuID={selectedMenuID}
                        onSelectMenu={handleMenuSelect}
                        onRemoveMenu={(menuIndex) => {
                            selectedMenuID = menuIndex;
                            handleRemoveMenu();
                        }}
                        disableRemove={menuIndices.length <= 1}
                        onAddMenu={handleAddMenu}
                />
            </section>
            <!-- --- UI: Main Content Area --- -->
            {#if selectedMenuID !== undefined}
                <div class="main-content-area flex flex-col space-y-8">
                    <!-- --- UI: Pie Menus Section --- -->
                    <section class="pie-menus-section">
                        {#if sortedPagesForSelectedMenu.length > 0}
                            <div
                                    class="flex rounded-b-lg space-x-4 overflow-x-auto py-3 px-3 horizontal-scrollbar bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 shadow-md"
                                    bind:this={pagesContainer}
                                    use:horizontalScroll
                            >
                                <div class="flex flex-row gap-x-6 pb-0">
                                    {#key quickMenuFavoriteVersion}
                                        {#each sortedPagesForSelectedMenu as [pageIDOfLoop, buttonsOnPage] (pageIDOfLoop)}
                                            {@const currentMenuIDForCallback = selectedMenuID}
                                            <button type="button"
                                                    class="page-container flex-shrink-0 rounded-lg shadow-sm bg-zinc-800 overflow-hidden border-2"
                                                    data-page-id={pageIDOfLoop}
                                                    class:dark:border-purple-500={selectedButtonDetails && selectedButtonDetails.menuID === currentMenuIDForCallback && selectedButtonDetails.pageID === pageIDOfLoop}
                                                    class:dark:border-zinc-700={!selectedButtonDetails || selectedButtonDetails.menuID !== currentMenuIDForCallback || selectedButtonDetails.pageID !== pageIDOfLoop}
                                                    class:border-purple-500={selectedButtonDetails && selectedButtonDetails.menuID === currentMenuIDForCallback && selectedButtonDetails.pageID === pageIDOfLoop}
                                                    class:border-zinc-300={!selectedButtonDetails || selectedButtonDetails.menuID !== currentMenuIDForCallback || selectedButtonDetails.pageID !== pageIDOfLoop}
                                                    onclick={() => {
                                                    if (!selectedButtonDetails || selectedButtonDetails.menuID !== currentMenuIDForCallback || selectedButtonDetails.pageID !== pageIDOfLoop) {
                                                        selectedButtonDetails = {
                                                            menuID: currentMenuIDForCallback,
                                                            pageID: pageIDOfLoop,
                                                            buttonID: 0,
                                                            slotIndex: 0,
                                                            button: buttonsOnPage.get(0) ?? getDefaultButton(ButtonType.ShowAnyWindow)
                                                        };
                                                    }
                                                 }}
                                                    style="cursor:pointer"
                                                    aria-label={`Select page ${pageIDOfLoop + 1}`}
                                            >
                                                <ConfigPieMenuPage
                                                        menuID={currentMenuIDForCallback}
                                                        pageID={pageIDOfLoop}
                                                        buttonsOnPage={buttonsOnPage}
                                                        onButtonClick={handlePieButtonClick}
                                                        onRemovePage={(removedPageID) => handleRemovePage(currentMenuIDForCallback, removedPageID)}
                                                        activeSlotIndex={selectedButtonDetails && selectedButtonDetails.menuID === currentMenuIDForCallback && selectedButtonDetails.pageID === pageIDOfLoop
                                                            ? selectedButtonDetails.slotIndex
                                                            : -1
                                                        }
                                                        isQuickMenuFavorite={(() => { const fav = getQuickMenuFavorite(); return !!fav && fav.menuID === currentMenuIDForCallback && fav.pageID === pageIDOfLoop; })()}
                                                />
                                            </button>
                                        {/each}
                                    {/key}
                                    <AddPageButton onClick={handleAddPage}/>
                                </div>
                            </div>
                        {:else if pagesForSelectedMenu && pagesForSelectedMenu.size === 0 && selectedMenuID !== undefined}
                            <div class="flex flex-col items-center justify-center py-10 text-center bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md">
                                <p class="text-zinc-900 dark:text-zinc-200">Menu {selectedMenuID + 1} has no
                                    pages.</p>
                                <AddPageButton onClick={handleAddPage}/>
                            </div>
                        {:else if !pagesForSelectedMenu && selectedMenuID !== undefined}
                            <p class="text-zinc-900 dark:text-zinc-200">Loading page data for
                                Menu {selectedMenuID + 1}
                                ...</p>
                        {/if}
                    </section>
                    <!-- --- UI: Button Details & Actions --- -->
                    <div class="w-full flex flex-row items-start gap-4">
                        <div class="min-w-[396px] max-w-[480px] w-full break-words">
                            {#if selectedButtonDetails}
                                <ButtonInfoDisplay
                                        selectedButtonDetails={selectedButtonDetails}
                                        onConfigChange={handleButtonConfigUpdate}
                                        menuConfig={baseMenuConfig}
                                />
                            {:else}
                                <div class="p-4 border border-zinc-300 dark:border-zinc-700 rounded-lg shadow text-center text-zinc-500 dark:text-zinc-400">
                                    Select a button from a pie menu preview to see its details, or add a page
                                    if the menu is
                                    empty.
                                </div>
                            {/if}
                        </div>
                        <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 min-w-[280px] max-w-[360px] self-start">
                            <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 mb-2 w-full text-left">
                                Page Settings
                            </h3>
                            <ButtonTypeSelector
                                    currentType={resetType}
                                    buttonTypeKeys={buttonTypeKeys}
                                    buttonTypeFriendlyNames={buttonTypeFriendlyNames}
                                    onChange={handleResetTypeChange}
                            />
                            <StandardButton
                                    label="Reset Page with Type"
                                    onClick={handleResetPageToDefault}
                                    disabled={selectedMenuID === undefined || (selectedButtonDetails && selectedButtonDetails.pageID === undefined)}
                                    style="margin-top: 0.5rem; margin-bottom: 0.5rem;"
                                    variant="primary"
                            />
                            <button
                                    aria-label="Use for Quick Menu"
                                    class="mt-2 px-4 py-2 bg-zinc-900/30 dark:bg-white/5 rounded-lg border border-white dark:border-zinc-400 text-white dark:text-white transition-colors focus:outline-none cursor-pointer disabled:bg-zinc-900/20 disabled:text-white/60 disabled:dark:text-zinc-500 hover:bg-zinc-900/10 dark:hover:bg-white/10 disabled:hover:bg-white/0 disabled:dark:hover:bg-white/0 flex items-center w-full relative shadow-sm"
                                    onclick={handleUseForQuickMenu}
                                    disabled={isQuickMenuFavorite || selectedMenuID === undefined || (selectedButtonDetails && selectedButtonDetails.pageID === undefined)}
                            >
                                <span class="absolute left-4 top-1/2 -translate-y-1/2 flex items-center justify-center min-w-[1.25rem]">
                                    <img src="/tabler_icons/star.svg" alt="star icon"
                                         class="inline w-5 h-5 align-text-bottom invert"/>
                                </span>
                                <span class="mx-auto w-full text-center block">
                                    {#if isQuickMenuFavorite}Used for Quick Menu{:else}Use for Quick Menu{/if}
                                </span>
                            </button>
                        </div>
                        <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 min-w-[396px] max-w-[480px] self-start">
                            <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 mb-2 w-full text-left">
                                Menu Settings</h3>
                            <div class="flex flex-row justify-between items-center w-full mt-2">
                                <div class="flex flex-col">
                                    <span class="text-zinc-700 dark:text-zinc-200">Set Shortcut to open Menu:</span>
                                    {#if selectedMenuID !== undefined && shortcutLabels[selectedMenuID] && shortcutUsage[shortcutLabels[selectedMenuID]] && shortcutUsage[shortcutLabels[selectedMenuID]].length > 1}
                                        <span class="mt-1 text-xs text-red-500 font-semibold">Warning: Shortcut is used multiple times!</span>
                                    {/if}
                                </div>
                                <StandardButton
                                        variant="special"
                                        onClick={handlePublishShortcutSetterUpdate}
                                        disabled={selectedMenuID === undefined}
                                        label={selectedMenuID !== undefined && shortcutLabels[selectedMenuID]
                                        ? shortcutLabels[selectedMenuID]
                                        : 'Set Shortcut'}
                                />
                            </div>
                            <div class="flex flex-row justify-between items-center w-full mt-2">
                                <span class="text-zinc-700 dark:text-zinc-200">Clear Shortcut:</span>
                                <StandardButton
                                        label="Clear"
                                        onClick={handleClearShortcut}
                                        disabled={selectedMenuID === undefined || !shortcutLabels[selectedMenuID]}
                                        style="max-width: 120px;"
                                        variant="primary"
                                />
                            </div>
                            <div class="flex flex-row justify-between mt-2 items-center w-full">
                                <span class="text-zinc-700 dark:text-zinc-200">Reset the whole Config:</span>
                                <StandardButton
                                        label="Reset"
                                        variant="warning"
                                        onClick={() => showResetAllConfirmDialog = true}
                                        style="max-width: 120px;"
                                />
                            </div>
                        </div>
                        <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 w-auto self-start">
                            <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 mb-3 w-full text-left">
                                Config Backup
                            </h3>
                            <div class="flex flex-col items-start gap-2 w-full">
                                <StandardButton
                                        label="Create Backup"
                                        onClick={handleBackupWithConfirmation}
                                        style={`width: 100%;`}
                                        variant="primary"
                                />
                                <StandardButton
                                        label="Load Backup"
                                        onClick={openFileDialog}
                                        style={`width: 100%;`}
                                        variant="primary"
                                />
                            </div>
                        </div>
                    </div>
                </div>
            {:else if selectedMenuID !== undefined}
                <p class="text-zinc-900 dark:text-zinc-200">Loading pages for Menu {selectedMenuID + 1} or menu is
                    empty.</p>
            {/if}
        {:else}
            <div class="flex flex-col items-center justify-center py-10 text-center bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md">
                <p class="text-zinc-900 dark:text-zinc-200 mb-4">No menus found. Configuration might be loading or
                    empty.</p>
                <button
                        class="px-4 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer disabled:opacity-60 disabled:text-zinc-400 disabled:dark:text-zinc-500 hover:bg-zinc-300 dark:hover:bg-zinc-600 disabled:hover:bg-zinc-200 disabled:dark:hover:bg-zinc-700 shadow-sm"
                        onclick={() => goto('/')}
                        type="button"
                >
                    Back to Home
                </button>
            </div>
        {/if}

        <!-- Compact Action Buttons -->
        <div class="absolute bottom-4 right-4">
            <div class="flex flex-row justify-end items-center gap-2 px-4 py-3 bg-zinc-200 dark:bg-neutral-900 opacity-90 rounded-xl shadow-md">
                <StandardButton
                        label="Undo"
                        ariaLabel="Undo"
                        onClick={handleUndo}
                        disabled={undoHistory.length === 0}
                        variant="primary"
                        bold={true}
                />
                <StandardButton
                        label="Discard Changes"
                        ariaLabel="Discard Changes"
                        onClick={() => showDiscardConfirmDialog = true}
                        disabled={undoHistory.length === 0}
                        variant="primary"
                        bold={true}
                />
                <StandardButton
                        label="Done"
                        ariaLabel="Done"
                        onClick={() => goto('/')}
                        variant="primary"
                        bold={true}
                />
            </div>
        </div>
    </div>
    <!-- --- UI: Dialogs --- -->
    <ConfirmationDialog
            isOpen={showRemoveMenuDialog}
            message="Are you sure you want to remove this menu?"
            onCancel={cancelRemoveMenu}
            onConfirm={confirmRemoveMenu}
    />
    <ConfirmationDialog
            cancelText="Save Changes"
            confirmText="Discard Changes"
            isOpen={showDiscardConfirmDialog}
            message="You have unsaved changes. What would you like to do?"
            onCancel={() => { showDiscardConfirmDialog = false; goto('/'); }}
            onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); goto('/'); }}
            onClose={() => { showDiscardConfirmDialog = false; }}
            title="Unsaved Changes"
    />
    <ConfirmationDialog
            cancelText="Cancel"
            confirmText="Reset All"
            isOpen={showResetAllConfirmDialog}
            message="This will remove all menus except the first menu and reset it to a single page with default buttons. Are you sure? (Undo will still work.)"
            onCancel={cancelResetAllMenus}
            onConfirm={confirmResetAllMenus}
            title="Reset All Menus?"
    />
    <SetShortcutDialog isOpen={isShortcutDialogOpen} onCancel={closeShortcutDialog}/>
    <NotificationDialog
            isOpen={showBackupCreatedDialog}
            message="A backup of the current Config has been created."
            onClose={() => showBackupCreatedDialog = false}
            title="Backup Created"
    />
</div>

<style>
    /* Add any additional styles needed for consistency */
    :global(.horizontal-scrollbar::-webkit-scrollbar) {
        height: 6px;
    }

    :global(.horizontal-scrollbar::-webkit-scrollbar-track) {
        background: rgba(0, 0, 0, 0.1);
        border-radius: 3px;
    }

    :global(.horizontal-scrollbar::-webkit-scrollbar-thumb) {
        background: rgba(0, 0, 0, 0.2);
        border-radius: 3px;
    }

    :global(.horizontal-scrollbar::-webkit-scrollbar-thumb:hover) {
        background: rgba(0, 0, 0, 0.3);
    }

    :global(.dark .horizontal-scrollbar::-webkit-scrollbar-track) {
        background: rgba(255, 255, 255, 0.1);
    }

    :global(.dark .horizontal-scrollbar::-webkit-scrollbar-thumb) {
        background: rgba(255, 255, 255, 0.2);
    }

    :global(.dark .horizontal-scrollbar::-webkit-scrollbar-thumb:hover) {
        background: rgba(255, 255, 255, 0.3);
    }
</style>
