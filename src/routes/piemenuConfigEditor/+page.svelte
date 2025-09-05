<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<!-- src/routes/piemenuConfig/+page.svelte -->
<script lang="ts">
    // --- Svelte and Third-Party Imports ---
    import {onDestroy, onMount} from 'svelte';
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
    import {open, save} from '@tauri-apps/plugin-dialog';
    import {join} from '@tauri-apps/api/path';

    // --- Internal Library Imports ---
    import {
        addMenuToMenuConfiguration,
        addPageToMenuConfiguration,
        getPieMenuConfig,
        parseButtonConfig,
        publishPieMenuConfig,
        removeMenuFromMenuConfiguration,
        removePageFromMenuConfiguration,
        unparseMenuConfiguration,
        updateButtonInMenuConfig
    } from '$lib/data/configManager.svelte.ts';
    import type {Button, ButtonsConfig, ButtonsOnPageMap, PagesInMenuMap} from '$lib/data/types/pieButtonTypes.ts';
    import {ButtonType} from '$lib/data/types/pieButtonTypes.ts';
    import {horizontalScroll} from "$lib/generalUtil.ts";
    import {publishMessage, useNatsSubscription} from "$lib/natsAdapter.svelte.ts";
    // --- Function Definitions for CallFunction Buttons ---
    import {
        PUBLIC_CONFIG_SIZE_X,
        PUBLIC_CONFIG_SIZE_Y,
        PUBLIC_DIR_CONFIGBACKUPS,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_ERROR,
        PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_ABORT,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE
    } from "$env/static/public";
    import {getDefaultButton} from '$lib/data/types/pieButtonDefaults.ts';
    import type {PieMenuConfig, ShortcutEntry, ShortcutsMap} from '$lib/data/types/piemenuConfigTypes.ts';
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

    // Full local editor config (authoritative for the editor until Save)
    let editorPieMenuConfig = $state<PieMenuConfig>({buttons: {}, shortcuts: {}, starred: null});
    // Buttons-only map derived from editorPieMenuConfig.buttons for UI editing
    let editorButtonsConfig = $state<ButtonsConfig>(new Map());
    let selectedMenuID = $state<number | undefined>(undefined);
    let selectedButtonDetails = $state<
        { menuID: number; pageID: number; buttonID: number; slotIndex: number; button: Button } | undefined
    >(undefined);
    let sortedPagesForSelectedMenu = $state<[number, ButtonsOnPageMap][]>([]);
    let showRemoveMenuDialog = $state(false);
    let isShortcutDialogOpen = $state(false);
    let shortcutLabels = $derived.by(() => {
        const labels: Record<number, string> = {};
        const sc = editorPieMenuConfig?.shortcuts || {};
        Object.entries(sc).forEach(([k, entry]: [string, any]) => {
            const n = Number(k);
            if (!isNaN(n) && entry && typeof entry.label === 'string') {
                labels[n] = entry.label;
            }
        });
        return labels;
    });
    let prevShortcutLabels: Record<number, string> | undefined;
    let hasMounted = false;
    let pagesContainer = $state<HTMLDivElement | null>(null);
    let lastTrackedMenuID: number | undefined = undefined;

    // Listen for live shortcut capture updates from the backend capture adapter and stage them in local editor state
    const subscription_shortcutsetter_update = useNatsSubscription(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE, (msg: string) => {
        logger.debug('ShortcutSetter UPDATE raw message:', msg);
        try {
            const obj = JSON.parse(msg) as ShortcutsMap;
            if (!obj || typeof obj !== 'object') {
                logger.warn('ShortcutSetter UPDATE ignored: payload not an object');
                return;
            }
            const entries = Object.entries(obj).filter(([k, v]) => {
                const n = Number(k);
                return !isNaN(n) && v && typeof v === 'object' && Array.isArray((v as ShortcutEntry).codes);
            });
            if (entries.length === 0) {
                logger.warn('ShortcutSetter UPDATE ignored: no valid ShortcutEntry items found');
                return;
            }
            // Make incoming shortcut capture undoable
            pushUndoState();
            const current = editorPieMenuConfig.shortcuts || {};
            const next: ShortcutsMap = {...current} as ShortcutsMap;
            for (const [k, v] of entries) {
                next[k] = {codes: (v as ShortcutEntry).codes, label: (v as ShortcutEntry).label} as ShortcutEntry;
            }
            editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: next};
            logger.debug('ShortcutSetter UPDATE applied for indices:', entries.map(([k]) => k).join(','));
        } catch (e) {
            logger.error('ShortcutSetter UPDATE handler error:', e);
        }
        // Close dialog regardless to reflect the end of capture
        isShortcutDialogOpen = false;
    });

    // Track subscription status and errors, consistent with other pages
    $effect(() => {
        if (subscription_shortcutsetter_update.status === "subscribed") {
            logger.debug("NATS subscription_shortcutsetter_update ready.");
        }
        if (subscription_shortcutsetter_update.error) {
            logger.error("NATS subscription_shortcutsetter_update error:", subscription_shortcutsetter_update.error);
            // If we were capturing a shortcut, close the dialog on error to avoid a stuck UI
            if (isShortcutDialogOpen) {
                isShortcutDialogOpen = false;
            }
        }
    });

    let initialEditPieMenuConfigSnapshot: PieMenuConfig | undefined = undefined;

    // --- Undo State ---
    type UndoState = {
        editorButtonsConfigSnapshot: ButtonsConfig;
        editorPieMenuConfigSnapshot: PieMenuConfig;
        selectedMenuID: number | undefined;
        selectedButtonDetails: typeof selectedButtonDetails;
    };
    const undoHistory = $state<UndoState[]>([]);
    const UNDO_LIMIT = 20;

    let showDiscardConfirmDialog = $state(false);
    let showResetAllConfirmDialog = $state(false);
    let showBackupCreatedDialog = $state(false);
    // Load Config error notification state
    let showLoadFailedDialog = $state(false);
    let loadFailedMessage = $state<string>("Failed to load config.");

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

    function cloneMenuConfig(config: ButtonsConfig): ButtonsConfig {
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
        const editorPieMenuConfigSnapshot: PieMenuConfig = JSON.parse(JSON.stringify({
            ...editorPieMenuConfig,
            // ensure buttons in snapshot reflect current editorButtonsConfig
            buttons: unparseMenuConfiguration(editorButtonsConfig)
        }));
        undoHistory.push({
            editorButtonsConfigSnapshot: cloneMenuConfig(editorButtonsConfig),
            editorPieMenuConfigSnapshot: editorPieMenuConfigSnapshot,
            selectedMenuID,
            selectedButtonDetails: selectedButtonDetails ? {...selectedButtonDetails} : undefined
        });
        if (undoHistory.length > UNDO_LIMIT) undoHistory.shift();
    }

    function handleUndo() {
        if (undoHistory.length === 0) return;
        const prev = undoHistory.pop();
        if (!prev) return;
        editorButtonsConfig = prev.editorButtonsConfigSnapshot;
        selectedMenuID = prev.selectedMenuID;
        selectedButtonDetails = prev.selectedButtonDetails;
        if (prev.editorPieMenuConfigSnapshot) {
            editorPieMenuConfig = prev.editorPieMenuConfigSnapshot;
        }
    }

    function discardChanges() {
        if (!initialEditPieMenuConfigSnapshot) return;
        // Restore full baseline config captured at mount
        editorPieMenuConfig = JSON.parse(JSON.stringify(initialEditPieMenuConfigSnapshot));
        // Re-derive the editable buttons map from the restored baseline
        editorButtonsConfig = parseButtonConfig(editorPieMenuConfig.buttons);
        // Select first menu if available
        const initialMenuIndices = Array.from(editorButtonsConfig.keys());
        if (initialMenuIndices.length > 0) {
            selectedMenuID = initialMenuIndices[0];
        } else {
            selectedMenuID = undefined;
        }
        selectedButtonDetails = undefined; // Let the auto-select $effect handle button selection
        undoHistory.length = 0;
    }

    // --- Reset the whole Config ---
    function handleResetConfig() {
        pushUndoState();
        const menuKeys = Array.from(editorButtonsConfig.keys()).sort((a, b) => a - b);
        if (menuKeys.length === 0) return;
        const firstMenuID = menuKeys[0];
        // Reset menu config to only the first menu with one default page
        const newConfig = new Map();
        const defaultButtons = new Map();
        for (let i = 0; i < 8; i++) {
            defaultButtons.set(i, getDefaultButton(ButtonType.ShowAnyWindow));
        }
        const pagesInMenu = new Map();
        pagesInMenu.set(0, defaultButtons);
        newConfig.set(firstMenuID, pagesInMenu);
        editorButtonsConfig = newConfig;
        selectedMenuID = firstMenuID;
        selectedButtonDetails = undefined;
        // Also clear shortcuts and starred in local editor config
        editorPieMenuConfig = {
            ...editorPieMenuConfig,
            shortcuts: {},
            starred: null,
        };
    }

    function confirmResetConfig() {
        showResetAllConfirmDialog = false;
        handleResetConfig();
    }

    function cancelResetAllMenus() {
        showResetAllConfirmDialog = false;
    }

    // --- Backup handler: open Save dialog and persist full editor config via backend
    async function handleSaveConfigViaDialog() {
        try {
            // Resolve default backups directory and filename
            const configPath = await invoke<string>('get_app_data_dir');
            const backupsDir = await join(configPath, PUBLIC_DIR_CONFIGBACKUPS);
            const defaultFile = await join(backupsDir, 'piemenuConfig.json');

            const targetPath = await save({
                defaultPath: defaultFile,
                filters: [{name: 'JSON', extensions: ['json']}]
            });

            if (targetPath) {
                // Send the chosen path to backend; it will write the current authoritative config there
                publishMessage(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP, String(targetPath));
                showBackupCreatedDialog = true;
            }
        } catch (e) {
            logger.error('Save Backup dialog failed:', e);
        }
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

    // Re-seed local editor state from backend full-config updates (e.g., after loading a backup)
    const subscription_backend_update = useNatsSubscription(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE, (msg: string) => {
        try {
            const full = JSON.parse(msg);
            if (!full || !full.buttons) return;
            const prevSelected = selectedMenuID;
            // Make loading a backup undoable by capturing current state first
            pushUndoState();
            editorPieMenuConfig = full as PieMenuConfig;
            editorButtonsConfig = parseButtonConfig(full.buttons);
            // preserve selection if possible
            if (prevSelected !== undefined && editorButtonsConfig.has(prevSelected)) {
                selectedMenuID = prevSelected;
            } else {
                const keys = Array.from(editorButtonsConfig.keys());
                selectedMenuID = keys.length > 0 ? keys[0] : undefined;
                selectedButtonDetails = undefined;
            }
        } catch (e) {
            logger.error('Failed to handle BACKEND_UPDATE in editor:', e);
        }
    });

    // Listen for explicit backend load errors and notify user immediately
    const subscription_backend_load_error = useNatsSubscription(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_ERROR, (msg: string) => {
        try {
            const payload = JSON.parse(msg);
            const baseMsg = 'Failed to load config';
            const detail = typeof payload?.error === 'string' ? payload.error : '';
            loadFailedMessage = `${baseMsg}${detail ? `: ${detail}` : ''}`;
        } catch (_e) {
            loadFailedMessage = 'Failed to load config.';
        }
        showLoadFailedDialog = true;
    });

    // Track subscription status and errors, consistent with other pages
    $effect(() => {
        if (subscription_backend_update.status === "subscribed") {
            logger.debug("NATS subscription_backend_update ready.");
        }
        if (subscription_backend_update.error) {
            logger.error("NATS subscription_backend_update error:", subscription_backend_update.error);
        }
    });

    // Track load error subscription as well
    $effect(() => {
        if (subscription_backend_load_error.status === "subscribed") {
            logger.debug("NATS subscription_backend_load_error ready.");
        }
        if (subscription_backend_load_error.error) {
            logger.error("NATS subscription_backend_load_error error:", subscription_backend_load_error.error);
        }
    });

    // Auto-select first button of first page when menu changes or on mount (IDs = 0)
    $effect(() => {
        if (selectedMenuID === undefined) return;
        if (selectedButtonDetails && selectedButtonDetails.menuID === selectedMenuID) return;
        const pagesMap = editorButtonsConfig.get(selectedMenuID);
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
    const menuIndices = $derived(Array.from(editorButtonsConfig.keys()));
    const pagesForSelectedMenu = $derived<PagesInMenuMap | undefined>(
        selectedMenuID !== undefined ? editorButtonsConfig.get(selectedMenuID) : undefined
    );

    // --- Lifecycle ---
    onMount(async () => {
        // Seed local editor state from current full config
        editorPieMenuConfig = JSON.parse(JSON.stringify(getPieMenuConfig()));
        editorButtonsConfig = parseButtonConfig(editorPieMenuConfig.buttons);
        initialEditPieMenuConfigSnapshot = JSON.parse(JSON.stringify(editorPieMenuConfig));
        const initialMenuIndices = Array.from(editorButtonsConfig.keys());
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

    onDestroy(() => {
        // Auto-save only if there are staged in-memory changes
        if (undoHistory.length > 0) {
            savePieMenuConfig();
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
        pushUndoState();
        const {menuID, pageID, buttonID, newButton} = payload;
        editorButtonsConfig = updateButtonInMenuConfig(editorButtonsConfig, menuID, pageID, buttonID, newButton);
        // Local-only update
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
            const backupsDir = await join(configPath, PUBLIC_DIR_CONFIGBACKUPS);
            const selected = await open({
                multiple: false,
                defaultPath: backupsDir,
                filters: [{name: 'JSON', extensions: ['json']}]
            });

            if (selected) {
                pushUndoState();
                publishMessage(PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP, selected);
                logger.log('Published selected file path:', selected);
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
        const result = addPageToMenuConfiguration(editorButtonsConfig, selectedMenuID);
        if (result) {
            editorButtonsConfig = result.newConfig;
            // local-only
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
        const result = removePageFromMenuConfiguration(editorButtonsConfig, menuIDToRemoveFrom, pageIDToRemove);
        if (result) {
            editorButtonsConfig = result;
            // local-only
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
        const result = addMenuToMenuConfiguration(editorButtonsConfig);
        editorButtonsConfig = result.newConfig;
        // local-only
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
        if (selectedMenuID === undefined) return;
        // Clear local shortcut and starred if they reference this menu
        const key = String(selectedMenuID);
        const newShortcuts = {...(editorPieMenuConfig.shortcuts || {})};
        if (newShortcuts[key]) delete newShortcuts[key];
        const newStarred = editorPieMenuConfig.starred && editorPieMenuConfig.starred.menuID === selectedMenuID
            ? null
            : editorPieMenuConfig.starred;
        editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: newShortcuts, starred: newStarred};

        const newConfig = removeMenuFromMenuConfiguration(editorButtonsConfig, selectedMenuID);
        if (newConfig) {
            editorButtonsConfig = newConfig;
            // local-only
            const indices = Array.from(editorButtonsConfig.keys());
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
            // Make setting a shortcut undoable (actual staging occurs when the dialog/capture resolves)
            pushUndoState();
            // NOTE: Currently starts backend capture; if we fully move to in-memory-only staging,
            // we will replace this with a local capture mechanism and NOT publish.
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_CAPTURE, {index: selectedMenuID});
            logger.log("Published shortcut setter update for menu index:", selectedMenuID);
            isShortcutDialogOpen = true;
        }
    }

    function handleClearShortcut() {
        if (selectedMenuID === undefined) return;
        // Stage clear in full config and make it undoable; do NOT publish immediately
        pushUndoState();
        const newShortcuts: Record<string, {
            codes: number[];
            label: string
        }> = {...(editorPieMenuConfig.shortcuts || {})} as any;
        const key = String(selectedMenuID);
        if (newShortcuts[key]) {
            delete newShortcuts[key];
        }
        editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: newShortcuts};
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
        const newConfig = new Map(editorButtonsConfig);
        const menuPages = new Map(newConfig.get(selectedMenuID));
        menuPages.set(pageID, newButtonsOnPage);
        newConfig.set(selectedMenuID, menuPages);
        editorButtonsConfig = newConfig;
    }

    // --- Quick Menu Favorite (starred) Logic ---
    let isStarred = $state(false);

    $effect(() => {
        const starred = editorPieMenuConfig.starred;
        if (selectedMenuID !== undefined && selectedButtonDetails && selectedButtonDetails.pageID !== undefined && starred) {
            isStarred = starred.menuID === selectedMenuID && starred.pageID === selectedButtonDetails.pageID;
        } else {
            isStarred = false;
        }
    });

    function handleUseForQuickMenu() {
        if (selectedMenuID !== undefined && selectedButtonDetails && selectedButtonDetails.pageID !== undefined) {
            // If already Starred, do nothing (button will be disabled in UI)
            if (isStarred) return;
            // Make setting Starred undoable in local editor config
            pushUndoState();
            editorPieMenuConfig = {
                ...editorPieMenuConfig,
                starred: {menuID: selectedMenuID, pageID: selectedButtonDetails.pageID}
            };
            isStarred = true;
        }
    }

    // Compose and publish the full editorPieMenuConfig (buttons + shortcuts + starred)
    function savePieMenuConfig() {
        // Unparse buttons from staged editorButtonsConfig
        const buttons = unparseMenuConfiguration(editorButtonsConfig);
        const newFull: PieMenuConfig = {
            buttons,
            shortcuts: {...(editorPieMenuConfig.shortcuts || {})},
            starred: editorPieMenuConfig.starred ?? null,
        };
        // Publish to backend and update global authoritative store
        publishPieMenuConfig(newFull);
    }

    // Editor config is owned by this page and should not be continuously reloaded or fall back to live config.
</script>

<div class="w-full h-screen p-1">
    <div class="w-full h-full flex flex-col bg-gradient-to-br from-amber-500 to-purple-700 rounded-t-3xl rounded-b-2xl shadow-[0px_1px_4px_rgba(0,0,0,0.5)]">
        <!-- --- Title Bar --- -->
        <div class="title-bar relative flex items-center py-1 bg-zinc-200 dark:bg-neutral-800 rounded-t-2xl border-b border-none h-8 flex-shrink-0">
            <div class="w-0.5 min-w-[2px] h-full" data-tauri-drag-region="none"></div>
            <div class="absolute left-0 right-0 top-0 bottom-0 flex items-center justify-center pointer-events-none select-none">
                <span class="font-semibold text-sm lg:text-base text-zinc-900 dark:text-zinc-400">Pie Menu Config</span>
            </div>
            <div class="flex-1 h-full" data-tauri-drag-region></div>
        </div>
        <!-- --- Main Content --- -->
        <div class="flex-1 w-full p-4 overflow-y-auto horizontal-scrollbar relative"
             style="scrollbar-gutter: stable both-edges;">
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
                    <div class="main-content-area flex flex-col space-y-6">
                        <!-- --- UI: Pie Menus Section --- -->
                        <section class="pie-menus-section">
                            {#if sortedPagesForSelectedMenu.length > 0}
                                <div
                                        class="flex rounded-b-lg space-x-4 overflow-x-scroll py-3 px-3 horizontal-scrollbar bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 shadow-md"
                                        bind:this={pagesContainer}
                                        use:horizontalScroll
                                        style="scrollbar-gutter: stable both-edges;"
                                >
                                    <div class="flex flex-row gap-x-6 pb-0">
                                        {#key JSON.stringify(editorPieMenuConfig.starred)}
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
                                                            isStarred={(() => { const starred = editorPieMenuConfig.starred; return !!starred && starred.menuID === currentMenuIDForCallback && starred.pageID === pageIDOfLoop; })()}
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
                                            menuConfig={editorButtonsConfig}
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
                                        class="mt-2 px-4 py-2 bg-zinc-900/30 dark:bg-white/5 rounded-lg border border-white dark:border-zinc-400 text-white dark:text-white transition-colors focus:outline-none cursor-pointer disabled:cursor-not-allowed disabled:bg-zinc-900/20 disabled:text-white/60 disabled:dark:text-zinc-500 hover:bg-zinc-900/10 dark:hover:bg-white/10 disabled:hover:bg-white/0 disabled:dark:hover:bg-white/0 flex items-center w-full relative shadow-sm"
                                        onclick={handleUseForQuickMenu}
                                        disabled={isStarred || selectedMenuID === undefined || (selectedButtonDetails && selectedButtonDetails.pageID === undefined)}
                                >
                                <span class="absolute left-4 top-1/2 -translate-y-1/2 flex items-center justify-center min-w-[1.25rem]">
                                    <img src="/tabler_icons/star.svg" alt="star icon"
                                         class="inline w-5 h-5 align-text-bottom invert"/>
                                </span>
                                    <span class="mx-auto w-full text-center block">
                                    {#if isStarred}Used for Quick Menu{:else}Use for Quick Menu{/if}
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
                            </div>
                            <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 w-auto self-start">
                                <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 mb-3 w-full text-left">
                                    Pie Menu Config
                                </h3>
                                <div class="flex flex-col items-start gap-2 w-full">
                                    <StandardButton
                                            label="Save Config"
                                            onClick={handleSaveConfigViaDialog}
                                            style={`width: 100%;`}
                                            variant="primary"
                                    />
                                    <StandardButton
                                            label="Load Config"
                                            onClick={openFileDialog}
                                            style={`width: 100%;`}
                                            variant="primary"
                                    />
                                    <StandardButton
                                            label="Reset it all!"
                                            variant="warning"
                                            onClick={() => showResetAllConfirmDialog = true}
                                            style={`width: 100%;`}
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
                            ariaLabel="Undo"
                            bold={true}
                            disabled={undoHistory.length === 0}
                            label="Undo"
                            onClick={handleUndo}
                            variant="primary"
                    />
                    <StandardButton
                            ariaLabel="Discard Changes"
                            bold={true}
                            disabled={undoHistory.length === 0}
                            label="Discard Changes"
                            onClick={() => showDiscardConfirmDialog = true}
                            variant="primary"
                    />
                    <StandardButton
                            ariaLabel="Done"
                            bold={true}
                            label="Done"
                            onClick={() => { savePieMenuConfig(); goto('/'); }}
                            variant="primary"
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
                onCancel={() => { showDiscardConfirmDialog = false; savePieMenuConfig(); goto('/'); }}
                onClose={() => { showDiscardConfirmDialog = false; }}
                onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); goto('/'); }}
                title="Unsaved Changes"
        />
        <ConfirmationDialog
                cancelText="Cancel"
                confirmText="Reset All"
                isOpen={showResetAllConfirmDialog}
                message="This will remove all menus except the first menu and reset it to a single page with default buttons. Are you sure? (Undo will still work.)"
                onCancel={cancelResetAllMenus}
                onConfirm={confirmResetConfig}
                title="Reset All Menus?"
        />
        <SetShortcutDialog isOpen={isShortcutDialogOpen} onCancel={closeShortcutDialog}/>
        <NotificationDialog
                isOpen={showBackupCreatedDialog}
                message="Pie Menu Configuration has been saved."
                onClose={() => showBackupCreatedDialog = false}
                title="Config Saved"
        />
        <NotificationDialog
                isOpen={showLoadFailedDialog}
                message={loadFailedMessage}
                onClose={() => showLoadFailedDialog = false}
                title="Load Failed"
        />
    </div>
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
