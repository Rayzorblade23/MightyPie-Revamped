<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<script lang="ts">
    import {onDestroy, onMount} from 'svelte';
    import {getSettings, publishSettings, type SettingsMap} from '$lib/data/settingsManager.svelte.ts';
    import {goto} from "$app/navigation";
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
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
        getSavedAutoStartPreference,
        syncAutoStartPreference
    } from '$lib/autostartUtils';
    import StandardButton from '$lib/components/StandardButton.svelte';

    // Create a logger for this component
    const logger = createLogger('Settings');

    // Canonical deep clone for settings, matching piemenuConfig approach
    function cloneSettings(settings: SettingsMap): SettingsMap {
        return JSON.parse(JSON.stringify(settings));
    }

    let settings = $state<SettingsMap>(cloneSettings(getSettings()));
    let currentWindow: Window | null = null;

    // Autostart state (managed separately from settings)
    let autoStartEnabled = $state<boolean | null>(null);
    let autoStartLoading = $state<boolean>(true);

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
                // Get the buttonFunctions.json parsed data using the utility function
                const buttonFunctions = await getButtonFunctions();

                // Extract the keys for deadzone function options
                deadzoneFunctionOptions = Object.keys(buttonFunctions).sort();
            } catch (e) {
                logger.error("Failed to load buttonFunctions.json for deadzone options:", e);
            }

            // Load autostart setting
            try {
                autoStartLoading = true;
                // Get the actual system autostart status
                autoStartEnabled = await syncAutoStartPreference();
            } catch (error) {
                logger.error('Failed to load autostart setting:', error);
                // Fall back to saved preference or default to false
                autoStartEnabled = getSavedAutoStartPreference() ?? false;
            } finally {
                autoStartLoading = false;
            }
        })();

        return () => {
            window.removeEventListener("keydown", handleKeyDown);
        };
    });

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
    async function handleAutoStartToggle(e: Event) {
        const target = e.target as HTMLInputElement;
        const newValue = target?.checked ?? false;

        try {
            autoStartLoading = true;
            if (newValue) {
                await enableAutoStart();
            } else {
                await disableAutoStart();
            }
            autoStartEnabled = newValue;
        } catch (error) {
            logger.error(`Failed to ${newValue ? 'enable' : 'disable'} autostart:`, error);
            // Revert the UI state on error
            autoStartEnabled = !newValue;
        } finally {
            autoStartLoading = false;
        }
    }

    onDestroy(() => {
        publishSettings(settings);
    });
</script>

<div class="w-full h-screen flex flex-col bg-gradient-to-br from-amber-500 to-purple-700 rounded-2xl shadow-lg">
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
                        <label class="relative inline-flex items-center cursor-pointer select-none">
                            <input
                                    type="checkbox"
                                    id="autoStartWithSystem"
                                    checked={autoStartEnabled ?? false}
                                    disabled={autoStartLoading}
                                    class="sr-only"
                                    onchange={handleAutoStartToggle}
                            />
                            <span
                                    class="block w-10 h-6 rounded-full transition-colors duration-200 relative bg-zinc-200 dark:bg-neutral-800"
                                    style="opacity: {autoStartLoading ? '0.7' : '1'};"
                            >
                                <span
                                        class="absolute left-0.5 top-0.5 w-5 h-5 rounded-full shadow transition-transform duration-200"
                                        class:bg-amber-400={autoStartEnabled}
                                        class:bg-zinc-500={!autoStartEnabled}
                                        class:dark:bg-amber-400={autoStartEnabled}
                                        class:dark:bg-zinc-200={!autoStartEnabled}
                                        style="transform: translateX({autoStartEnabled ? '1.0rem' : '0'});"
                                ></span>
                            </span>
                        </label>
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
                                    <label class="relative inline-flex items-center cursor-pointer select-none">
                                        <input
                                                type="checkbox"
                                                id={key}
                                                checked={entry.value}
                                                class="sr-only"
                                                onchange={e => handleBooleanChange(e, key)}
                                        />
                                        <span
                                                class="block w-10 h-6 rounded-full transition-colors duration-200 relative bg-zinc-200 dark:bg-neutral-800"
                                        >
                                            <span
                                                    class="absolute left-0.5 top-0.5 w-5 h-5 rounded-full shadow transition-transform duration-200"
                                                    class:bg-amber-400={entry.value}
                                                    class:bg-zinc-500={!entry.value}
                                                    class:dark:bg-amber-400={entry.value}
                                                    class:dark:bg-zinc-200={!entry.value}
                                                    style="transform: translateX({entry.value ? '1.0rem' : '0'});"
                                            ></span>
                                        </span>
                                    </label>
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
                                           class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)}/>
                                {:else if entry.type === 'int' || entry.type === 'integer'}
                                    <input type="number" id={key}
                                           step="1"
                                           inputmode="numeric"
                                           pattern="^-?\\d+$"
                                           class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)}
                                           onkeydown={handleIntKeydown}/>
                                {:else if entry.type === 'string'}
                                    <input type="text" id={key}
                                           class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleStringInput(key, e)}/>
                                {:else if entry.type === 'enum'}
                                    <select id={key}
                                            class="bg-zinc-200 dark:bg-neutral-800 border border-none rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-amber-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
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
                <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl mb-2 shadow-md border border-none">
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
                            Open App Folder
                        </button>
                    </div>
                </div>
            </div>
        {/if}
    </div>

    <!-- Action Buttons Footer -->
    <div class="flex-shrink-0 w-full px-2 py-3 bg-zinc-200 dark:bg-neutral-900 opacity-90 border-t border-none rounded-b-2xl">
        <div class="w-full flex flex-row justify-end items-center gap-2 px-6">
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

<!-- Dialog moved outside the footer div to prevent inheriting opacity -->
<ConfirmationDialog
        cancelText="Save Changes"
        confirmText="Discard Changes"
        isOpen={showDiscardConfirmDialog}
        message="You have unsaved changes. What would you like to do?"
        onCancel={() => { showDiscardConfirmDialog = false; goto('/'); }}
        onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); goto('/'); }}
        title="Unsaved Changes"
/>
