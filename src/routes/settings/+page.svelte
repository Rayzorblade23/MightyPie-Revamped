<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<script lang="ts">
    import {onMount} from 'svelte';
    import {getSettings, publishSettings, type SettingsMap} from '$lib/data/settingsManager.svelte.ts';
    import {goto} from "$app/navigation";
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
    import {getVersion} from "@tauri-apps/api/app";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils.ts";
    import {
        PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER,
        PUBLIC_SETTINGS_SIZE_X,
        PUBLIC_SETTINGS_SIZE_Y
    } from "$env/static/public";
    import ConfirmationDialog from '$lib/components/ui/ConfirmationDialog.svelte';
    import {publishMessage} from '$lib/natsAdapter.svelte.ts';
    import {createLogger} from "$lib/logger";
    import {getButtonFunctions} from "$lib/fileAccessUtils.ts";
    import {
        disableAutoStart,
        enableAutoStart,
        getSavedAdminRightsPreference,
        getSavedAutoStartPreference,
        isRunningAsAdmin,
        restartWithAdminRights,
        syncAdminRightsPreference,
        syncAutoStartPreference
    } from "$lib/autostartUtils";
    import StandardButton from '$lib/components/StandardButton.svelte';
    import ElevationDialog from '$lib/components/ui/ElevationDialog.svelte';
    import Toggle from '$lib/components/Toggle.svelte';

    // Create a logger for this component
    const logger = createLogger('Settings');

    // Canonical deep clone for settings, matching piemenuConfig approach
    function cloneSettings(settings: SettingsMap): SettingsMap {
        return JSON.parse(JSON.stringify(settings));
    }

    let settings = $state<SettingsMap>(cloneSettings(getSettings()));
    let currentWindow: Window | null = null;
    let appVersion = $state<string>("");

    // Autostart state (managed separately from settings)
    let autoStartEnabled = $state<boolean | null>(null);
    let adminRightsEnabled = $state<boolean | null>(null);
    let autoStartLoading = $state<boolean>(false);
    let adminRightsLoading = $state<boolean>(false);

    // State for elevation dialog
    let showElevationDialog = $state<boolean>(false);
    let pendingElevationAction = $state<(() => Promise<void>) | null>(null);

    // --- Undo/Discard State ---
    let undoHistory = $state<SettingsMap[]>([]);
    let showDiscardConfirmDialog = $state(false);
    let initialSettingsSnapshot: SettingsMap = cloneSettings(getSettings());

    // --- Deadzone Function Options State ---
    let deadzoneFunctionOptions = $state<string[]>([]);

    onMount(() => {
        logger.info('Settings Mounted');

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                if (event.defaultPrevented) return;
                if (showDiscardConfirmDialog) return;
                const active = document.activeElement;
                // If an input or textarea is focused, first Escape should blur it, second Escape should trigger the normal logic
                if (active && (["INPUT", "TEXTAREA"].includes(active.tagName) || active.getAttribute("contenteditable") === "true")) {
                    (active as HTMLElement).blur();
                    return;
                }
                // Only block Escape if the user is editing a text input or textarea
                if (active && ["INPUT", "TEXTAREA"].includes(active.tagName) && !(active as HTMLInputElement).readOnly && !(active as HTMLInputElement).disabled && (active as HTMLInputElement).type === "text") return;
                // Use the same logic as Discard Changes button for unsaved changes
                if (undoHistory.length === 0) {
                    goto('/');
                } else {
                    showDiscardConfirmDialog = true;
                }
            }
        };
        window.addEventListener("keydown", handleKeyDown);

        // Async logic for window setup
        (async () => {
            try {
                currentWindow = getCurrentWindow();
                await centerAndSizeWindowOnMonitor(
                    currentWindow,
                    Number(PUBLIC_SETTINGS_SIZE_X),
                    Number(PUBLIC_SETTINGS_SIZE_Y)
                );
                await currentWindow.show();
            } catch (e) {
                logger.error('Error setting up window:', e);
            }
            try {
                appVersion = await getVersion();
            } catch (e) {
                logger.error('Failed to get app version:', e);
            }
            try {
                // Get the buttonFunctions.json parsed data using the utility function
                const buttonFunctions = await getButtonFunctions();

                // Extract the keys for deadzone function options
                deadzoneFunctionOptions = Object.keys(buttonFunctions).sort();
            } catch (e) {
                logger.error("Failed to load buttonFunctions.json for deadzone options:", e);
            }

            // Initialize settings
            initSettings();

            // Initialize autostart and admin rights state
            initAutoStartState();
        })();

        return () => {
            window.removeEventListener("keydown", handleKeyDown);
        };
    });

    // Initialize settings
    function initSettings() {
        // Load settings from storage
        settings = cloneSettings(getSettings());
    }

    // Initialize autostart and admin rights state
    async function initAutoStartState() {
        try {
            // First try to get saved preferences from local storage
            const savedAutoStart = getSavedAutoStartPreference();
            const savedAdminRights = getSavedAdminRightsPreference();

            // Use saved preferences initially to avoid UI flicker
            autoStartEnabled = savedAutoStart ?? false;
            adminRightsEnabled = savedAdminRights ?? false;

            // Then sync with actual system state
            autoStartEnabled = await syncAutoStartPreference();
            adminRightsEnabled = await syncAdminRightsPreference();
        } catch (error) {
            logger.error('Failed to initialize autostart state:', error);
            autoStartEnabled = false;
            adminRightsEnabled = false;
        }
    }

    function pushUndoState() {
        undoHistory = [...undoHistory, cloneSettings(settings)];
        if (undoHistory.length > 20) undoHistory = undoHistory.slice(1);
    }

    function handleValueChange(key: string, value: any) {
        pushUndoState();
        settings = {
            ...settings,
            [key]: {
                ...settings[key],
                value
            }
        };
    }

    function handleUndo() {
        if (undoHistory.length === 0) return;
        const prev = undoHistory[undoHistory.length - 1];
        settings = cloneSettings(prev);
        undoHistory = undoHistory.slice(0, -1);
    }

    function discardChanges() {
        pushUndoState();
        settings = cloneSettings(initialSettingsSnapshot);
    }

    function handleBooleanChange(e: Event, key: string) {
        const target = e.target as HTMLInputElement;
        handleValueChange(key, target?.checked ?? false);
    }

    function handleNumberChange(e: Event, key: string) {
        const target = e.target as HTMLInputElement;
        let val = target?.value;
        if (settings[key].type === 'int' || settings[key].type === 'integer') {
            // Only allow valid integers
            if (/^-?\d+$/.test(val)) {
                handleValueChange(key, Number(val));
            } else if (val === '') {
                handleValueChange(key, ''); // allow clearing
            } // else ignore floats and invalid input
        } else {
            handleValueChange(key, val !== undefined ? Number(val) : settings[key].value);
        }
    }

    function handleStringChange(key: string, value: string) {
        handleValueChange(key, value);
    }

    function handleStringInput(key: string, e: Event) {
        const target = e.target as HTMLInputElement;
        handleStringChange(key, target.value);
    }

    function handleColorInput(key: string, e: Event) {
        const target = e.target as HTMLInputElement;
        handleStringChange(key, target.value);
    }

    function handleHexInput(key: string, e: Event) {
        const target = e.target as HTMLInputElement;
        handleStringChange(key, '#' + target.value.replace(/[^0-9a-fA-F]/g, '').slice(0, 6));
    }

    function handleEnumChange(e: Event, key: string) {
        const target = e.target as HTMLSelectElement;
        handleValueChange(key, target?.value ?? settings[key].value);
    }

    function handleResetToDefault(key: string) {
        pushUndoState();
        settings = {
            ...settings,
            [key]: {
                ...settings[key],
                value: settings[key].defaultValue
            }
        };
    }

    function handleIntKeydown(e: KeyboardEvent) {
        if (e.key === '.' || e.key === ',') {
            e.preventDefault();
        }
    }

    // Handle autostart toggle
    async function handleAutoStartToggle(e: MouseEvent) {
        // Always prevent the default behavior first
        e.preventDefault();
        e.stopPropagation();

        try {
            // Check if we're running as admin
            const isAdmin = await isRunningAsAdmin();

            if (!isAdmin) {
                // If not admin, show elevation dialog without changing toggle state
                logger.info('Not running as admin, showing elevation dialog');

                // Set up the elevation action
                pendingElevationAction = async () => {
                    await restartWithAdminRights();
                };

                // Show the dialog
                showElevationDialog = true;
                return;
            }

            // If we are admin, proceed with the toggle
            try {
                autoStartLoading = true;

                // Toggle the state
                const newValue = !autoStartEnabled;

                if (newValue) {
                    await enableAutoStart(adminRightsEnabled ?? false);
                    autoStartEnabled = true;
                } else {
                    await disableAutoStart();
                    autoStartEnabled = false;
                }
            } catch (error) {
                logger.error(`Failed to toggle autostart:`, error);
            } finally {
                autoStartLoading = false;
            }
        } catch (error) {
            logger.error('Error in handleAutoStartToggle:', error);
        }
    }

    // Handle admin rights toggle
    async function handleAdminRightsToggle(e: MouseEvent) {
        // Always prevent the default behavior first
        e.preventDefault();
        e.stopPropagation();

        try {
            // If autostart is not enabled, don't do anything
            if (!autoStartEnabled) {
                return;
            }

            // Check if we're running as admin
            const isAdmin = await isRunningAsAdmin();

            if (!isAdmin) {
                // If not admin, show elevation dialog without changing toggle state
                logger.info('Not running as admin, showing elevation dialog');

                // Set up the elevation action
                pendingElevationAction = async () => {
                    await restartWithAdminRights();
                };

                // Show the dialog
                showElevationDialog = true;
                return;
            }

            // If we are admin, proceed with the toggle
            try {
                adminRightsLoading = true;

                // Toggle the state
                const newValue = !adminRightsEnabled;

                // If autostart is enabled, update the task with new admin rights setting
                await enableAutoStart(newValue);
                adminRightsEnabled = newValue;
            } catch (error) {
                logger.error(`Failed to toggle admin rights:`, error);
            } finally {
                adminRightsLoading = false;
            }
        } catch (error) {
            logger.error('Error in handleAdminRightsToggle:', error);
        }
    }

    // Handle elevation dialog confirm
    function handleElevationConfirm() {
        if (pendingElevationAction) {
            pendingElevationAction().catch(error => {
                logger.error('Failed to restart with admin rights:', error);
            });
        }
    }

    // Handle elevation dialog cancel
    function handleElevationCancel() {
        // Just close the dialog and do nothing else
        showElevationDialog = false;
        pendingElevationAction = null;
    }

    // Explicitly save before leaving the page to avoid timing issues
    async function saveAndExit() {
        try {
            // Ensure the latest local state is sent to backend
            publishSettings(settings);
        } catch (e) {
            logger.error('Failed to publish settings before exit:', e);
        } finally {
            await goto('/');
        }
    }
</script>

<div class="w-full h-screen p-2">
    <div class="w-full h-full flex flex-col bg-gradient-to-br from-amber-500 to-purple-700 rounded-2xl shadow-md">
        <!-- Title Bar -->
        <div class="title-bar relative flex items-center py-1 bg-zinc-200 dark:bg-neutral-800 rounded-t-lg border-b border-none h-8 flex-shrink-0">
            <div class="w-0.5 min-w-[2px] h-full" data-tauri-drag-region="none"></div>
            <div class="absolute left-0 right-0 top-0 bottom-0 flex items-center justify-center pointer-events-none select-none">
                <span class="font-semibold text-sm lg:text-base text-zinc-900 dark:text-white">Settings</span>
            </div>
            <div class="flex-1 h-full" data-tauri-drag-region></div>
        </div>
        <div class="flex-1 w-full p-4 space-y-6 relative overflow-y-auto horizontal-scrollbar">
            {#if Object.keys(settings).length === 0}
                <p class="text-zinc-500 dark:text-zinc-400 bg-white/80 dark:bg-zinc-800/70 rounded-lg shadow p-6 text-center text-lg font-medium">
                    No settings available.</p>
            {:else}
                <div class="w-full">
                    <!-- Autostart Setting (managed separately) -->
                    <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mb-2 shadow-md border border-none">
                        <label class="w-1/2 md:w-1/3 text-zinc-900 dark:text-zinc-200 pr-4 pl-4 text-base"
                               for="autoStartWithSystem">Start automatically with system</label>
                        <div class="flex-1 flex items-center gap-2 min-w-0">
                            <Toggle
                                    id="autoStartWithSystem"
                                    checked={autoStartEnabled ?? false}
                                    disabled={autoStartLoading}
                                    onClick={handleAutoStartToggle}
                            />
                        </div>
                    </div>

                    <!-- Admin Rights Setting (managed separately) -->
                    <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mb-2 shadow-md border border-none">
                        <label class="w-1/2 md:w-1/3 text-zinc-900 dark:text-zinc-200 pr-4 pl-4 text-base"
                               for="adminRightsWithSystem">Start with admin rights</label>
                        <div class="flex-1 flex items-center gap-2 min-w-0">
                            <Toggle
                                    id="adminRightsWithSystem"
                                    checked={(adminRightsEnabled ?? false) && (autoStartEnabled ?? false)}
                                    disabled={adminRightsLoading || !autoStartEnabled}
                                    dimWhenDisabled={!autoStartEnabled}
                                    onClick={handleAdminRightsToggle}
                            />
                        </div>
                    </div>

                    <!-- Regular Settings -->
                    {#each Object.entries(settings)
                        .sort((a, b) => (a[1].index ?? 0) - (b[1].index ?? 0))
                            as [key, entry]}
                        {#if entry.isExposed}
                            <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mb-2 shadow-md border border-none">
                                <label class="w-1/2 md:w-1/3 text-zinc-900 dark:text-zinc-200 pr-4 pl-4 text-base"
                                       for={key}>{entry.label}</label>
                                <div class="flex-1 flex items-center gap-2 min-w-0">
                                    {#if entry.type === 'boolean' || entry.type === 'bool'}
                                        <Toggle
                                                id={key}
                                                checked={entry.value}
                                                onChange={(e: Event) => handleBooleanChange(e, key)}
                                        />
                                    {:else if entry.type === 'color'}
                                        <div class="flex flex-row items-center gap-2 min-w-0">
                                            <input type="color" id={key}
                                                   class="w-10 h-10 bg-transparent cursor-pointer flex-shrink-0"
                                                   value={entry.value}
                                                   onchange={e => handleColorInput(key, e)}/>
                                            <div class="relative w-28 flex-shrink min-w-0">
                                                <span class="absolute left-2 top-1/2 -translate-y-1/2 text-zinc-400 pointer-events-none select-none">#</span>
                                                <input type="text"
                                                       class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg pl-6 pr-2 py-1 focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all shadow-sm text-zinc-900 dark:text-zinc-100 w-full"
                                                       value={entry.value ? entry.value.replace(/^#/, '') : ''}
                                                       oninput={e => handleHexInput(key, e)}
                                                       maxlength="6"
                                                       placeholder="RRGGBB"/>
                                            </div>
                                        </div>
                                    {:else if entry.type === 'number' || entry.type === 'float'}
                                        <input type="number" id={key}
                                               class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                               value={entry.value}
                                               onchange={e => handleNumberChange(e, key)}/>
                                    {:else if entry.type === 'int' || entry.type === 'integer'}
                                        <input type="number" id={key}
                                               step="1"
                                               inputmode="numeric"
                                               pattern="^-?\\d+$"
                                               class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                               value={entry.value}
                                               onchange={e => handleNumberChange(e, key)}
                                               onkeydown={handleIntKeydown}/>
                                    {:else if entry.type === 'string'}
                                        <input type="text" id={key}
                                               class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                               value={entry.value}
                                               onchange={e => handleStringInput(key, e)}/>
                                    {:else if entry.type === 'enum'}
                                        <select id={key}
                                                class="custom-select pl-3 py-1 bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                                value={entry.value}
                                                onchange={e => handleEnumChange(e, key)}>
                                            {#if key === 'pieMenuDeadzoneFunction'}
                                                {#each deadzoneFunctionOptions as opt}
                                                    <option value={opt} selected={entry.value === opt}>{opt}</option>
                                                {/each}
                                            {:else}
                                                {#each entry.options ?? [] as opt}
                                                    <option value={opt} selected={entry.value === opt}>{opt}</option>
                                                {/each}
                                            {/if}
                                        </select>
                                    {:else}
                                        <span>{entry.value}</span>
                                    {/if}
                                    <div class="flex-1"></div>
                                    <button
                                            class="w-8 h-8 flex items-center justify-center p-0 mx-1 rounded-lg bg-purple-800 dark:bg-purple-950 border-none shadow hover:bg-violet-800 dark:hover:bg-violet-950 active:bg-purple-700 dark:active:bg-indigo-950 transition-colors flex-shrink-0"
                                            title="Reset to Default"
                                            aria-label="Reset to Default"
                                            onclick={() => handleResetToDefault(key)}
                                            tabindex="0"
                                            type="button"
                                    >
                                        <img src="tabler_icons/restore.svg" alt="Reset to Default"
                                             class="w-5 h-5 opacity-90 invert"/>
                                    </button>
                                </div>
                            </div>
                        {/if}
                    {/each}

                    <!-- Folder Navigation Buttons -->
                    <div class="flex flex-row items-center h-12 py-0 px-2 md:px-4 bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mb-2 shadow-md border border-none">
                        <div class="flex flex-row gap-2 w-full">
                            <button
                                    class="w-full px-4 py-2 bg-purple-800 dark:bg-purple-950 border border-none rounded-lg flex items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 text-zinc-100 text-sm font-medium shadow-md"
                                    onclick={() => publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER, 'appdata')}
                            >
                                Open App Config Folder
                            </button>
                            <button
                                    class="w-full px-4 py-2 bg-purple-800 dark:bg-purple-950 border border-none rounded-lg flex items-center justify-center hover:bg-violet-800 dark:hover:bg-violet-950 transition active:bg-purple-700 dark:active:bg-indigo-950 text-zinc-100 text-sm font-medium shadow-md"
                                    onclick={() => publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER, 'appfolder')}
                            >
                                Open Installation Folder
                            </button>
                        </div>
                    </div>
                </div>
            {/if}
        </div>

        <!-- Action Buttons Footer -->
        <div class="flex-shrink-0 w-full px-2 py-3 bg-zinc-200 dark:bg-neutral-800 border-t border-none rounded-b-lg">
            <div class="w-full flex flex-row justify-between items-center gap-2 px-6">
                <div class="text-xs text-zinc-600 dark:text-zinc-400 select-none">
                    {#if appVersion}
                        v{appVersion}
                    {/if}
                </div>
                <div class="flex flex-row items-center gap-2">
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
                            onClick={saveAndExit}
                            variant="primary"
                    />
                </div>
            </div>
        </div>
    </div>
</div>

<!-- Dialog moved outside the footer div to prevent inheriting opacity -->
<ConfirmationDialog
        cancelText="Save Changes"
        confirmText="Discard Changes"
        isOpen={showDiscardConfirmDialog}
        message="You have unsaved changes. What would you like to do?"
        onCancel={() => { showDiscardConfirmDialog = false; saveAndExit(); }}
        onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); goto('/'); }}
        title="Unsaved Changes"
/>

<!-- Elevation Dialog -->
<ElevationDialog
        isOpen={showElevationDialog}
        message="This operation requires administrator privileges to modify the Windows Task Scheduler. Do you want to restart the application with elevated privileges?"
        onCancel={handleElevationCancel}
        onConfirm={handleElevationConfirm}
/>
