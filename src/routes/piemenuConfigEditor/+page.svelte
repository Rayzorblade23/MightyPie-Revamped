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
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_ABORT,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_ABORT,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_CAPTURE,
        PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_UPDATE
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
    import ApplicationSelector from "$lib/components/piemenuConfig/selectors/ApplicationSelector.svelte";
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

    // Also listen for button-abort at page level to force-close the dialog state
    const subscription_button_abort = useNatsSubscription(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_ABORT, (_msg: string) => {
        logger.debug('Button shortcut ABORT received at page level');
        isButtonShortcutDialogOpen = false;
        buttonShortcutErrorMessage = null;
    });

    // Track page-level button abort subscription status/errors
    $effect(() => {
        if (subscription_button_abort.status === "subscribed") {
            logger.debug("NATS subscription_button_abort ready.");
        }
        if (subscription_button_abort.error) {
            logger.error("NATS subscription_button_abort error:", subscription_button_abort.error);
        }
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
    let isButtonShortcutDialogOpen = $state(false);
    let buttonShortcutErrorMessage = $state<string | null>(null);
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
    const subscription_shortcutsetter_update = useNatsSubscription(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_UPDATE, (msg: string) => {
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
    // Context menu state
    let contextMenuVisible = $state(false);
    let contextMenuX = $state(0);
    let contextMenuY = $state(0);
    let contextMenuTargetMenuID = $state<number | undefined>(undefined);
    let contextMenuTargetPageID = $state<number | undefined>(undefined);
    // Page removal confirmation dialog state
    let showRemovePageDialog = $state(false);
    let pendingRemovePageMenuID = $state<number | undefined>(undefined);
    let pendingRemovePageID = $state<number | undefined>(undefined);
    // Clipboard state for copy/paste
    let copiedButton = $state<Button | null>(null);
    let copiedPageButtons = $state<Map<number, Button> | null>(null);

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

    // Detect conflict type for the selected menu's shortcut
    let shortcutConflictType = $derived.by(() => {
        if (selectedMenuID === undefined) return null;
        const currentShortcut = shortcutLabels[selectedMenuID];
        if (!currentShortcut) return null;

        const menusWithSameShortcut = shortcutUsage[currentShortcut];
        if (!menusWithSameShortcut || menusWithSameShortcut.length <= 1) return null;

        // Get current menu's target app
        const currentEntry = editorPieMenuConfig.shortcuts?.[String(selectedMenuID)];
        const currentTargetApp = currentEntry?.targetApp;

        // Check conflict types with other menus
        let hasTrueConflict = false;
        let hasAppSpecificOnly = false;

        for (const menuID of menusWithSameShortcut) {
            if (menuID === selectedMenuID) continue;

            const otherEntry = editorPieMenuConfig.shortcuts?.[String(menuID)];
            const otherTargetApp = otherEntry?.targetApp;

            // Both have target apps
            if (currentTargetApp && otherTargetApp) {
                // Same app = true conflict
                if (currentTargetApp === otherTargetApp) {
                    hasTrueConflict = true;
                }
                // Different apps = safe, app-specific
                else {
                    hasAppSpecificOnly = true;
                }
            }
            // One or both have no target app - only a conflict if they're both global
            else if (!currentTargetApp && !otherTargetApp) {
                hasTrueConflict = true;
            }
            // One is global, one is app-specific = safe (they won't conflict)
            else {
                hasAppSpecificOnly = true;
            }
        }

        // True conflicts take precedence
        if (hasTrueConflict) return 'global';
        if (hasAppSpecificOnly) return 'app-specific';
        return null;
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
        window.addEventListener('click', handleClickOutsideTargetAppTooltip);
        window.addEventListener('click', closeContextMenu);
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                if (event.defaultPrevented) return;
                const active = document.activeElement;
                // If an input, textarea, select, or contenteditable is focused, first Escape should blur it, second Escape should trigger the normal logic
                if (active && (["INPUT", "TEXTAREA", "SELECT"].includes(active.tagName) || active.getAttribute("contenteditable") === "true")) {
                    (active as HTMLElement).blur();
                    return;
                }
                if (contextMenuVisible) {
                    closeContextMenu();
                    return;
                }
                if (showRemoveMenuDialog || showDiscardConfirmDialog || showResetAllConfirmDialog || isShortcutDialogOpen || isButtonShortcutDialogOpen || showBackupCreatedDialog) return;
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
            window.removeEventListener('click', handleClickOutsideTargetAppTooltip);
            window.removeEventListener('click', closeContextMenu);
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
        publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_ABORT, {});
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
            
            // Select the newly created page
            const newPageButtons = editorButtonsConfig.get(selectedMenuID)?.get(result.newPageID);
            if (newPageButtons) {
                const firstButton = newPageButtons.get(0) ?? getDefaultButton(ButtonType.ShowAnyWindow);
                selectedButtonDetails = {
                    menuID: selectedMenuID,
                    pageID: result.newPageID,
                    buttonID: 0,
                    slotIndex: 0,
                    button: firstButton
                };
            }
            
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

    /** Check if a page has non-simple buttons */
    function hasNonSimpleButtons(buttonsOnPage: ButtonsOnPageMap): boolean {
        for (const button of buttonsOnPage.values()) {
            if (button &&
                button.button_type !== ButtonType.ShowAnyWindow &&
                button.button_type !== ButtonType.Disabled) {
                return true;
            }
        }
        return false;
    }

    /** Remove a page from the selected menu and update selection if needed. */
    function handleRemovePage(menuIDToRemoveFrom: number, pageIDToRemove: number) {
        // Check if page has non-simple buttons and show confirmation if needed
        const buttonsOnPage = editorButtonsConfig.get(menuIDToRemoveFrom)?.get(pageIDToRemove);
        if (buttonsOnPage && hasNonSimpleButtons(buttonsOnPage)) {
            pendingRemovePageMenuID = menuIDToRemoveFrom;
            pendingRemovePageID = pageIDToRemove;
            showRemovePageDialog = true;
            return;
        }
        
        // Proceed with removal
        executeRemovePage(menuIDToRemoveFrom, pageIDToRemove);
    }

    /** Actually execute the page removal */
    function executeRemovePage(menuIDToRemoveFrom: number, pageIDToRemove: number) {
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

    function confirmRemovePage() {
        if (pendingRemovePageMenuID !== undefined && pendingRemovePageID !== undefined) {
            executeRemovePage(pendingRemovePageMenuID, pendingRemovePageID);
        }
        showRemovePageDialog = false;
        pendingRemovePageMenuID = undefined;
        pendingRemovePageID = undefined;
    }

    function cancelRemovePage() {
        showRemovePageDialog = false;
        pendingRemovePageMenuID = undefined;
        pendingRemovePageID = undefined;
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
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_MENU_CAPTURE, {index: selectedMenuID});
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

    function handleTargetAppSelect(appName: string) {
        if (selectedMenuID === undefined) return;
        pushUndoState();
        const key = String(selectedMenuID);
        const currentShortcuts = editorPieMenuConfig.shortcuts || {};
        const currentEntry = currentShortcuts[key];

        if (currentEntry) {
            const updatedEntry: ShortcutEntry = {
                codes: currentEntry.codes,
                label: currentEntry.label
            };
            if (appName) {
                updatedEntry.targetApp = appName;
            }
            const newShortcuts = {
                ...currentShortcuts,
                [key]: updatedEntry
            };
            editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: newShortcuts};
        } else if (appName) {
            const newShortcuts = {
                ...currentShortcuts,
                [key]: {
                    codes: [],
                    label: '',
                    targetApp: appName
                }
            };
            editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: newShortcuts};
        }
    }

    function handleClearTargetApp() {
        if (selectedMenuID === undefined) return;
        pushUndoState();
        const key = String(selectedMenuID);
        const currentShortcuts = editorPieMenuConfig.shortcuts || {};
        const currentEntry = currentShortcuts[key];

        if (currentEntry) {
            const updatedEntry: ShortcutEntry = {
                codes: currentEntry.codes,
                label: currentEntry.label
            };
            const newShortcuts = {
                ...currentShortcuts,
                [key]: updatedEntry
            };
            editorPieMenuConfig = {...editorPieMenuConfig, shortcuts: newShortcuts};
        }
    }

    const buttonTypeFriendlyNames: Record<ButtonType, string> = {
        [ButtonType.ShowProgramWindow]: "Show Program Window",
        [ButtonType.ShowAnyWindow]: "Show Any Window",
        [ButtonType.CallFunction]: "Call Function",
        [ButtonType.LaunchProgram]: "Launch Program",
        [ButtonType.OpenSpecificPieMenuPage]: "Open Page",
        [ButtonType.OpenResource]: "Open Resource",
        [ButtonType.KeyboardShortcut]: "Keyboard Shortcut",
        [ButtonType.Disabled]: "Disabled",
    };
    const buttonTypeKeys = Object.keys(buttonTypeFriendlyNames) as ButtonType[];

    let resetType = $state<ButtonType>(ButtonType.ShowAnyWindow);

    const installedAppsMap = $derived(getInstalledAppsInfo());
    const selectedMenuTargetApp = $derived.by(() => {
        if (selectedMenuID === undefined) return '';
        const shortcut = editorPieMenuConfig.shortcuts?.[String(selectedMenuID)];
        return shortcut?.targetApp || '';
    });

    const selectedMenuAlias = $derived.by(() => {
        if (selectedMenuID === undefined) return '';
        return editorPieMenuConfig.menuAliases?.[String(selectedMenuID)] || '';
    });

    function handleMenuAliasChange(event: Event) {
        if (selectedMenuID === undefined) return;
        const input = event.target as HTMLInputElement;
        const newAlias = input.value.trim();
        pushUndoState();
        const currentAliases = editorPieMenuConfig.menuAliases || {};
        const newAliases = {...currentAliases};
        if (newAlias) {
            newAliases[String(selectedMenuID)] = newAlias;
        } else {
            delete newAliases[String(selectedMenuID)];
        }
        editorPieMenuConfig = {...editorPieMenuConfig, menuAliases: newAliases};
    }

    function handleResetMenuAlias() {
        if (selectedMenuID === undefined) return;
        pushUndoState();
        const currentAliases = editorPieMenuConfig.menuAliases || {};
        const newAliases = {...currentAliases};
        delete newAliases[String(selectedMenuID)];
        editorPieMenuConfig = {...editorPieMenuConfig, menuAliases: newAliases};
    }

    // State for target app tooltip
    let showTargetAppTooltip = $state(false);
    let targetAppQuestionMarkButton = $state<HTMLElement | null>(null);

    function toggleTargetAppTooltip(event: MouseEvent) {
        event.stopPropagation();
        showTargetAppTooltip = !showTargetAppTooltip;
    }

    function handleClickOutsideTargetAppTooltip(event: MouseEvent) {
        if (showTargetAppTooltip && 
            targetAppQuestionMarkButton && 
            event.target instanceof Node && 
            !targetAppQuestionMarkButton.contains(event.target)) {
            showTargetAppTooltip = false;
        }
    }

    function handlePageContextMenu(event: MouseEvent, menuID: number, pageID: number) {
        // Estimate context menu dimensions (adjust if needed)
        const menuWidth = 180;
        const menuHeight = 280; // Approximate height based on number of items
        
        // Get window dimensions
        const windowWidth = window.innerWidth;
        const windowHeight = window.innerHeight;
        
        // Calculate position, adjusting if too close to edges
        let x = event.clientX;
        let y = event.clientY;
        
        // Adjust horizontal position if menu would overflow right edge
        if (x + menuWidth > windowWidth) {
            x = Math.max(10, windowWidth - menuWidth - 10);
        }
        
        // Adjust vertical position if menu would overflow bottom edge
        if (y + menuHeight > windowHeight) {
            y = Math.max(10, windowHeight - menuHeight - 10);
        }
        
        contextMenuX = x;
        contextMenuY = y;
        contextMenuTargetMenuID = menuID;
        contextMenuTargetPageID = pageID;
        contextMenuVisible = true;
    }

    function closeContextMenu() {
        contextMenuVisible = false;
    }

    function handleContextMenuCopyButton() {
        if (selectedButtonDetails) {
            const button = editorButtonsConfig.get(selectedButtonDetails.menuID)?.get(selectedButtonDetails.pageID)?.get(selectedButtonDetails.slotIndex);
            if (button) {
                copiedButton = JSON.parse(JSON.stringify(button));
                logger.log(`Copied button from slot ${selectedButtonDetails.slotIndex}`);
            }
        }
        closeContextMenu();
    }

    function handleContextMenuCopyPage() {
        if (contextMenuTargetMenuID !== undefined && contextMenuTargetPageID !== undefined) {
            const pageButtons = editorButtonsConfig.get(contextMenuTargetMenuID)?.get(contextMenuTargetPageID);
            if (pageButtons) {
                copiedPageButtons = new Map(JSON.parse(JSON.stringify(Array.from(pageButtons.entries()))));
                logger.log(`Copied page ${contextMenuTargetPageID} from menu ${contextMenuTargetMenuID}`);
            }
        }
        closeContextMenu();
    }

    function handleContextMenuPasteButton() {
        if (copiedButton && selectedButtonDetails) {
            pushUndoState();
            const newConfig = new Map(editorButtonsConfig);
            const menuPages = new Map(newConfig.get(selectedButtonDetails.menuID));
            const pageButtons = new Map(menuPages.get(selectedButtonDetails.pageID));
            pageButtons.set(selectedButtonDetails.slotIndex, JSON.parse(JSON.stringify(copiedButton)));
            menuPages.set(selectedButtonDetails.pageID, pageButtons);
            newConfig.set(selectedButtonDetails.menuID, menuPages);
            editorButtonsConfig = newConfig;
            logger.log(`Pasted button to slot ${selectedButtonDetails.slotIndex}`);
        }
        closeContextMenu();
    }

    function handleContextMenuPastePage() {
        if (copiedPageButtons && contextMenuTargetMenuID !== undefined && contextMenuTargetPageID !== undefined) {
            pushUndoState();
            const newConfig = new Map(editorButtonsConfig);
            const menuPages = new Map(newConfig.get(contextMenuTargetMenuID));
            menuPages.set(contextMenuTargetPageID, new Map(JSON.parse(JSON.stringify(Array.from(copiedPageButtons.entries())))));
            newConfig.set(contextMenuTargetMenuID, menuPages);
            editorButtonsConfig = newConfig;
            logger.log(`Pasted page to page ${contextMenuTargetPageID} in menu ${contextMenuTargetMenuID}`);
        }
        closeContextMenu();
    }

    function handleContextMenuResetButton() {
        if (selectedButtonDetails) {
            pushUndoState();
            const defaultButton = getDefaultButton(ButtonType.ShowAnyWindow);
            const newConfig = new Map(editorButtonsConfig);
            const menuPages = new Map(newConfig.get(selectedButtonDetails.menuID));
            const pageButtons = new Map(menuPages.get(selectedButtonDetails.pageID));
            pageButtons.set(selectedButtonDetails.slotIndex, defaultButton);
            menuPages.set(selectedButtonDetails.pageID, pageButtons);
            newConfig.set(selectedButtonDetails.menuID, menuPages);
            editorButtonsConfig = newConfig;
            // Update selected button details
            selectedButtonDetails = {
                ...selectedButtonDetails,
                button: defaultButton
            };
            logger.log(`Reset button at slot ${selectedButtonDetails.slotIndex}`);
        }
        closeContextMenu();
    }

    function handleContextMenuSetQuickMenu() {
        if (contextMenuTargetMenuID !== undefined && contextMenuTargetPageID !== undefined) {
            pushUndoState();
            editorPieMenuConfig = {
                ...editorPieMenuConfig,
                starred: {menuID: contextMenuTargetMenuID, pageID: contextMenuTargetPageID}
            };
        }
        closeContextMenu();
    }

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

    // Compose and publish the full editorPieMenuConfig (buttons + shortcuts + starred + menuAliases)
    function savePieMenuConfig() {
        // Unparse buttons from staged editorButtonsConfig
        const buttons = unparseMenuConfiguration(editorButtonsConfig);
        const newFull: PieMenuConfig = {
            buttons,
            shortcuts: {...(editorPieMenuConfig.shortcuts || {})},
            starred: editorPieMenuConfig.starred ?? null,
            menuAliases: editorPieMenuConfig.menuAliases ? {...editorPieMenuConfig.menuAliases} : undefined,
        };
        // Publish to backend and update global authoritative store
        publishPieMenuConfig(newFull);
    }

    // Editor config is owned by this page and should not be continuously reloaded or fall back to live config.
</script>

<!-- Page-level overlay dialogs (avoid parent transparency) -->
<SetShortcutDialog
        errorMessage={buttonShortcutErrorMessage}
        isOpen={isButtonShortcutDialogOpen}
        onCancel={() => {
            isButtonShortcutDialogOpen = false;
            publishMessage(PUBLIC_NATSSUBJECT_SHORTCUTSETTER_BUTTON_ABORT, {});
        }}
/>

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
        <div class="flex-1 w-full overflow-y-auto horizontal-scrollbar relative flex flex-col min-h-0 px-4 pt-4 pb-0"
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
                            menuAliases={editorPieMenuConfig.menuAliases}
                    />
                </section>
                <!-- --- UI: Main Content Area --- -->
                {#if selectedMenuID !== undefined}
                    <div class="main-content-area flex flex-col space-y-6 flex-1 min-h-0">
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
                                                        const button: Button = buttonsOnPage.get(0) ?? getDefaultButton(ButtonType.ShowAnyWindow);
                                                        selectedButtonDetails = {
                                                            menuID: currentMenuIDForCallback,
                                                            pageID: pageIDOfLoop,
                                                            buttonID: 0,
                                                            slotIndex: 0,
                                                            button: button
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
                                                            onContextMenu={handlePageContextMenu}
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
                        <div class="w-full flex flex-wrap items-stretch gap-4 flex-1">
                            <div class="break-words flex flex-col min-w-0 w-[calc((100%-2rem)/3)] max-[1000px]:w-[calc((100%-1rem)/2)] max-[800px]:w-full transition-all duration-300">
                                {#if selectedButtonDetails}
                                    <div class="flex-1">
                                        <ButtonInfoDisplay
                                                selectedButtonDetails={selectedButtonDetails}
                                                onConfigChange={handleButtonConfigUpdate}
                                                menuConfig={editorButtonsConfig}
                                                bind:isButtonShortcutDialogOpen={isButtonShortcutDialogOpen}
                                                bind:buttonShortcutErrorMessage={buttonShortcutErrorMessage}
                                        />
                                    </div>
                                {:else}
                                    <div class="flex-1 p-4 border border-zinc-300 dark:border-zinc-700 rounded-lg shadow text-center text-zinc-500 dark:text-zinc-400">
                                        Select a button from a pie menu preview to see its details, or add a page
                                        if the menu is
                                        empty.
                                    </div>
                                {/if}
                            </div>
                            <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 min-w-0 w-[calc((100%-2rem)/3)] max-[1000px]:w-[calc((100%-1rem)/2)] max-[800px]:w-full transition-all duration-300">
                                <div class="flex items-center justify-between mb-2">
                                    <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 w-full text-left">
                                        Page Settings
                                    </h3>
                                    {#if selectedButtonDetails}
                                        <p class="text-right text-zinc-600 dark:text-zinc-400 whitespace-nowrap">Page: {selectedButtonDetails.pageID + 1}</p>
                                    {/if}
                                </div>
                                <span class="text-sm font-medium mt-2 mb-1 text-zinc-700 dark:text-zinc-400">Reset every button on this page to:</span>
                                <div class="flex flex-row justify-between items-center w-full">
                                    <div class="flex-1 mr-[0.5rem]">
                                        <ButtonTypeSelector
                                                currentType={resetType}
                                                buttonTypeKeys={buttonTypeKeys}
                                                buttonTypeFriendlyNames={buttonTypeFriendlyNames}
                                                onChange={handleResetTypeChange}
                                        />
                                    </div>
                                    <StandardButton
                                            label="Reset Page"
                                            onClick={handleResetPageToDefault}
                                            disabled={selectedMenuID === undefined || (selectedButtonDetails && selectedButtonDetails.pageID === undefined)}
                                            variant="primary"
                                    />
                                </div>
                                <span class="text-sm mt-3 font-medium text-zinc-700 dark:text-zinc-400">Assign this Page to Quick Menu:</span>
                                <button
                                        aria-label="Use for Quick Menu"
                                        class="mt-1 px-4 py-1 bg-zinc-900/30 dark:bg-white/5 rounded-lg border border-white dark:border-zinc-400 text-white dark:text-white transition-colors focus:outline-none cursor-pointer disabled:cursor-not-allowed disabled:bg-zinc-900/20 disabled:text-white/60 disabled:dark:text-zinc-500 hover:bg-zinc-900/10 dark:hover:bg-white/10 disabled:hover:bg-white/0 disabled:dark:hover:bg-white/0 flex items-center w-full relative shadow-sm"
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
                            <div class="flex flex-col items-stretch bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md px-4 py-3 min-w-0 w-[calc((100%-2rem)/3)] max-[1000px]:w-full max-[800px]:w-full transition-all duration-300">
                                <div class="flex items-center justify-between mb-2">
                                    <h3 class="font-semibold text-lg text-zinc-900 dark:text-zinc-200 w-full text-left">
                                        Menu Settings
                                    </h3>
                                    {#if selectedMenuID !== undefined}
                                        <p class="text-right text-zinc-600 dark:text-zinc-400 whitespace-nowrap">Menu: {selectedMenuID + 1}</p>
                                    {/if}
                                </div>
                                <div class="flex flex-col mt-2">
                                    <span class="text-sm font-medium text-zinc-700 dark:text-zinc-400">Custom Name:</span>
                                    <div class="flex flex-row items-center gap-2 mt-1">
                                        <input
                                            type="text"
                                            value={selectedMenuAlias}
                                            oninput={handleMenuAliasChange}
                                            placeholder={selectedMenuID !== undefined ? `Menu ${selectedMenuID + 1}` : ''}
                                            disabled={selectedMenuID === undefined}
                                            class="flex-1 px-3 py-2 bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-600 rounded-lg text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 dark:placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-purple-500 disabled:opacity-50 disabled:cursor-not-allowed"
                                        />
                                        <StandardButton
                                            label="Reset"
                                            onClick={handleResetMenuAlias}
                                            disabled={selectedMenuID === undefined || !selectedMenuAlias}
                                            style="max-width: 100px;"
                                            variant="primary"
                                        />
                                    </div>
                                </div>
                                <div class="flex flex-col">
                                    <span class="text-sm font-medium mt-2 text-zinc-700 dark:text-zinc-400">Set Shortcut to open Menu:</span>
                                    {#if shortcutConflictType === 'global'}
                                        <span class="mt-1 text-xs text-red-500 font-semibold">Warning: Shortcut conflicts with another menu!</span>
                                    {:else if shortcutConflictType === 'app-specific'}
                                        <span class="mt-1 text-xs text-blue-500 dark:text-blue-400 font-semibold">Note: Same shortcut used in different apps (safe)</span>
                                    {/if}
                                </div>
                                <div class="flex flex-row justify-between items-center w-full my-1">
                                    <StandardButton
                                            variant="special"
                                            onClick={handlePublishShortcutSetterUpdate}
                                            disabled={selectedMenuID === undefined}
                                            style="max-width: 300px; min-width: 150px;"
                                            label={selectedMenuID !== undefined && shortcutLabels[selectedMenuID]
                                            ? shortcutLabels[selectedMenuID]
                                            : 'Set Shortcut'}
                                    />
                                    <div class="flex-1 mx-2 my-1 h-[10px] rounded-lg bg-black/10"></div>
                                    <StandardButton
                                            label="Clear"
                                            onClick={handleClearShortcut}
                                            disabled={selectedMenuID === undefined || !shortcutLabels[selectedMenuID]}
                                            style="max-width: 120px;"
                                            variant="primary"
                                    />
                                </div>
                                <div class="mt-1.5">
                                    <div class="flex justify-between items-center mb-1">
                                        <span class="text-sm font-medium text-zinc-700 dark:text-zinc-400">Show Menu only in:</span>
                                        <div class="flex">
                                            <button
                                                class="flex items-center justify-center w-4 h-4 mr-1 rounded-full bg-purple-800 dark:bg-purple-950 text-zinc-100 hover:bg-violet-800 dark:hover:bg-violet-950 active:bg-purple-700 dark:active:bg-indigo-950 transition-colors text-xs font-medium"
                                                onclick={toggleTargetAppTooltip}
                                                aria-label="Show target app explanation"
                                                bind:this={targetAppQuestionMarkButton}
                                            >
                                                ?
                                            </button>
                                        </div>
                                    </div>
                                    <div class="flex flex-row items-start gap-2 -mt-3">
                                        <div class="flex-1">
                                            <ApplicationSelector
                                                    selectedAppName={selectedMenuTargetApp}
                                                    installedAppsMap={installedAppsMap}
                                                    onSelect={handleTargetAppSelect}
                                                    labelText=""
                                            />
                                        </div>
                                        <div class="flex flex-col" style="margin-top: 0.75rem;">
                                            <StandardButton
                                                    label="Clear"
                                                    onClick={handleClearTargetApp}
                                                    disabled={selectedMenuID === undefined || !selectedMenuTargetApp}
                                                    style="max-width: 120px;"
                                                    variant="primary"
                                            />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <!-- Spacer to create gap at bottom when scrolling (flex-shrink-0 prevents it from being compressed) -->
                        <div class="h-px bg-black/0 flex-shrink-0"></div>
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
        </div>

        {#if showTargetAppTooltip && targetAppQuestionMarkButton}
            {@const buttonRect = targetAppQuestionMarkButton.getBoundingClientRect()}
            <div class="fixed inset-0 z-[100] pointer-events-none">
                <div 
                    class="absolute bg-white dark:bg-zinc-800 p-3 rounded-md shadow-lg w-80 text-sm text-zinc-800 dark:text-zinc-200 border border-zinc-200 dark:border-zinc-700 whitespace-pre-line pointer-events-auto"
                    style="left: {Math.max(10, buttonRect.right - 320)}px; top: {Math.max(10, buttonRect.top - 10)}px; transform: translateY(-100%);"
                >
                    The shortcut will only work within the application selected here.
                    Leave it empty for the shortcut to work globally (in all applications).
                </div>
            </div>
        {/if}

        {#if contextMenuVisible}
            <div 
                class="fixed z-[200] bg-white dark:bg-zinc-800 rounded-lg shadow-xl border border-zinc-300 dark:border-zinc-600 py-1 min-w-[180px]"
                style="left: {contextMenuX}px; top: {contextMenuY}px;"
                onclick={(e) => e.stopPropagation()}
                onkeydown={(e) => { if (e.key === 'Escape') closeContextMenu(); }}
                role="menu"
                aria-label="Context menu"
                tabindex="-1"
            >
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2"
                    onclick={handleContextMenuCopyButton}
                    type="button"
                    disabled={!selectedButtonDetails}
                >
                    <img src="/tabler_icons/copy.svg" alt="" class="w-4 h-4 dark:invert" />
                    Copy Button
                </button>
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2"
                    onclick={handleContextMenuCopyPage}
                    type="button"
                >
                    <img src="/tabler_icons/copy.svg" alt="" class="w-4 h-4 dark:invert" />
                    Copy Page
                </button>
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                    onclick={handleContextMenuPasteButton}
                    type="button"
                    disabled={!copiedButton || !selectedButtonDetails}
                >
                    <img src="/tabler_icons/clipboard.svg" alt="" class="w-4 h-4 dark:invert" />
                    Paste into Button
                </button>
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                    onclick={handleContextMenuPastePage}
                    type="button"
                    disabled={!copiedPageButtons}
                >
                    <img src="/tabler_icons/clipboard.svg" alt="" class="w-4 h-4 dark:invert" />
                    Paste into Page
                </button>
                <div class="h-px bg-zinc-300 dark:bg-zinc-600 my-1"></div>
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                    onclick={handleContextMenuResetButton}
                    type="button"
                    disabled={!selectedButtonDetails}
                >
                    <img src="/tabler_icons/restore.svg" alt="" class="w-4 h-4 dark:invert" />
                    Reset Button
                </button>
                <button
                    class="w-full px-4 py-2 text-left text-sm text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors flex items-center gap-2"
                    onclick={handleContextMenuSetQuickMenu}
                    type="button"
                >
                    <img src="/tabler_icons/star.svg" alt="" class="w-4 h-4 dark:invert" />
                    Use for Quick Menu
                </button>
            </div>
        {/if}

        <div class="action-bar relative flex items-center py-1 bg-zinc-200 dark:bg-neutral-800 rounded-b-[0.875rem] border-t border-none flex-shrink-0">
            <div class="w-full flex flex-row justify-between items-center gap-2 px-4 py-2 max-[450px]:flex-col max-[450px]:items-start">
                <div class="w-auto flex flex-row justify-start items-center gap-2 max-[400px]:w-full">
                    <StandardButton
                            ariaLabel="Save Config"
                            label=""
                            onClick={handleSaveConfigViaDialog}
                            variant="primary"
                            iconSrc="/tabler_icons/device-floppy.svg"
                            iconImgClasses="w-6 h-6 invert"
                            iconSlotClasses="w-5 h-5"
                            tooltipText="Save Config"
                    />
                    <StandardButton
                            ariaLabel="Load Config"
                            label=""
                            onClick={openFileDialog}
                            variant="primary"
                            iconSrc="/tabler_icons/upload.svg"
                            iconImgClasses="w-5 h-5 invert"
                            iconSlotClasses="w-5 h-5"
                            tooltipText="Load Config"
                    />
                    <StandardButton
                            ariaLabel="Reset it all!"
                            label=""
                            onClick={() => showResetAllConfirmDialog = true}
                            variant="warning"
                            iconSrc="/tabler_icons/trash-x.svg"
                            iconImgClasses="w-6 h-6 invert"
                            iconSlotClasses="w-5 h-5"
                            tooltipText="Reset it all!"
                    />
                </div>
                <div class="w-auto flex flex-row justify-end items-center gap-2 max-[400px]:w-full max-[400px]:justify-start">
                    <StandardButton
                            ariaLabel="Undo"
                            bold={true}
                            disabled={undoHistory.length === 0}
                            label=""
                            onClick={handleUndo}
                            variant="primary"
                            iconSrc="/tabler_icons/player-skip-back.svg"
                            iconImgClasses="w-5 h-5 invert"
                            iconSlotClasses="w-5 h-5"
                            tooltipText="Undo"
                    />
                    <StandardButton
                            ariaLabel="Discard Changes"
                            bold={true}
                            disabled={undoHistory.length === 0}
                            label="Discard Changes"
                            onClick={() => showDiscardConfirmDialog = true}
                            variant="primary"
                            tooltipText="Discard Changes and Exit"
                    />
                    <StandardButton
                            ariaLabel="Done"
                            bold={true}
                            label=""
                            onClick={() => { savePieMenuConfig(); goto('/'); }}
                            variant="primary"
                            iconSrc="/tabler_icons/check.svg"
                            iconImgClasses="w-6 h-6 invert"
                            iconSlotClasses="w-5 h-5"
                            tooltipText="Apply and Exit"
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
        <ConfirmationDialog
                cancelText="Cancel"
                confirmText="Remove Page"
                isOpen={showRemovePageDialog}
                message="This page contains buttons that are not simple buttons. Are you sure you want to remove it?"
                onCancel={cancelRemovePage}
                onConfirm={confirmRemovePage}
                title="Remove Page"
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
