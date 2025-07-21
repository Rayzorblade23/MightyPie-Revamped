<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<script lang="ts">
    import {onDestroy, onMount} from 'svelte';
    import {getSettings, publishSettings, type SettingsMap} from '$lib/data/settingsManager.svelte.ts';
    import {goto} from "$app/navigation";
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils.ts";
    import {PUBLIC_SETTINGS_SIZE_X, PUBLIC_SETTINGS_SIZE_Y, PUBLIC_DIR_BUTTONFUNCTIONS, PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER} from "$env/static/public";
    import ConfirmationDialog from '$lib/components/ui/ConfirmationDialog.svelte';
    import {publishMessage} from '$lib/natsAdapter.svelte.ts';

    // Canonical deep clone for settings, matching piemenuConfig approach
    function cloneSettings(settings: SettingsMap): SettingsMap {
        return JSON.parse(JSON.stringify(settings));
    }

    let settings = $state<SettingsMap>(cloneSettings(getSettings()));
    let currentWindow: Window | null = null;

    // --- Undo/Discard State ---
    let undoHistory = $state<SettingsMap[]>([]);
    let showDiscardConfirmDialog = $state(false);
    let initialSettingsSnapshot: SettingsMap = cloneSettings(getSettings());

    // --- Deadzone Function Options State ---
    let deadzoneFunctionOptions = $state<string[]>([]);

    onMount(() => {
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
                console.error(e);
            }
            try {
                const response = await fetch(PUBLIC_DIR_BUTTONFUNCTIONS);
                if (response.ok) {
                    const defs = await response.json();
                    deadzoneFunctionOptions = Object.keys(defs).sort();
                }
            } catch (e) {
                console.error("Failed to load buttonFunctions.json for deadzone options:", e);
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
        handleStringChange(key, '#' + target.value.replace(/[^0-9a-fA-F]/g, '').slice(0,6));
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

    onDestroy(() => {
        publishSettings(settings);
    });
</script>

<div class="w-full h-screen flex flex-col bg-zinc-100 dark:bg-zinc-900 rounded-lg border-b border-zinc-200 dark:border-zinc-700">
    <!-- Title Bar -->
    <div class="title-bar relative flex items-center py-1 bg-zinc-300 dark:bg-zinc-800 rounded-t-lg border-b border-zinc-200 dark:border-zinc-700 h-8 flex-shrink-0">
        <div class="w-0.5 min-w-[2px] h-full" data-tauri-drag-region="none"></div>
        <div class="absolute left-0 right-0 top-0 bottom-0 flex items-center justify-center pointer-events-none select-none">
            <span class="font-semibold text-sm lg:text-base text-zinc-900 dark:text-zinc-400">Settings</span>
        </div>
        <div class="flex-1 h-full" data-tauri-drag-region></div>
    </div>
    <div class="flex-1 w-full p-4 space-y-6 relative overflow-y-auto horizontal-scrollbar">
        {#if Object.keys(settings).length === 0}
            <p class="text-zinc-500 dark:text-zinc-400 bg-white/80 dark:bg-zinc-800/70 rounded-lg shadow p-6 text-center text-lg font-medium">
                No settings available.</p>
        {:else}
            <div class="w-full">
                {#each Object.entries(settings)
                    .sort((a, b) => (a[1].index ?? 0) - (b[1].index ?? 0))
                    as [key, entry]}
                    {#if entry.isExposed}
                        <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg mb-2 shadow-sm border border-zinc-200 dark:border-zinc-700">
                            <label class="w-1/2 md:w-1/3 text-zinc-900 dark:text-zinc-200 pr-4 pl-4 text-base" for={key}>{entry.label}</label>
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
                                            class="block w-10 h-6 rounded-full transition-colors duration-200 relative"
                                            style="background-color: {entry.value ? '#2563eb' : (document.documentElement.classList.contains('dark') ? '#374151' : '#d1d5db')};"
                                        >
                                            <span
                                                class="absolute left-0.5 top-0.5 w-5 h-5 bg-white dark:bg-zinc-200 rounded-full shadow transition-transform duration-200"
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
                                                   class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg pl-6 pr-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all shadow-sm text-zinc-900 dark:text-zinc-100 w-full"
                                                   value={entry.value ? entry.value.replace(/^#/, '') : ''}
                                                   oninput={e => handleHexInput(key, e)}
                                                   maxlength="6"
                                                   placeholder="RRGGBB" />
                                        </div>
                                    </div>
                                {:else if entry.type === 'number' || entry.type === 'float'}
                                    <input type="number" id={key}
                                           class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)} />
                                {:else if entry.type === 'int' || entry.type === 'integer'}
                                    <input type="number" id={key}
                                           step="1"
                                           inputmode="numeric"
                                           pattern="^-?\\d+$"
                                           class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)}
                                           onkeydown={handleIntKeydown} />
                                {:else if entry.type === 'string'}
                                    <input type="text" id={key}
                                           class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                           value={entry.value}
                                           onchange={e => handleStringInput(key, e)}/>
                                {:else if entry.type === 'enum' && key === 'pieMenuDeadzoneFunction'}
                                    <select id={key}
                                            class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                            value={entry.value}
                                            onchange={e => handleEnumChange(e, key)}>
                                        {#each deadzoneFunctionOptions as opt}
                                            <option value={opt} selected={entry.value === opt}>{opt}</option>
                                        {/each}
                                    </select>
                                {:else if entry.type === 'enum' && entry.options}
                                    <select id={key}
                                            class="bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-zinc-900 dark:text-zinc-100"
                                            value={entry.value}
                                            onchange={e => handleEnumChange(e, key)}>
                                        {#each entry.options as opt}
                                            <option value={opt} selected={entry.value === opt}>{opt}</option>
                                        {/each}
                                    </select>
                                {:else}
                                    <span>{entry.value}</span>
                                {/if}
                                <div class="flex-1"></div>
                                <button
                                    class="w-8 h-8 flex items-center justify-center p-0 rounded bg-white/80 dark:bg-zinc-700/80 border border-zinc-300 dark:border-zinc-600 shadow hover:bg-zinc-200 dark:hover:bg-zinc-600 transition-colors flex-shrink-0"
                                    title="Reset to Default"
                                    aria-label="Reset to Default"
                                    onclick={() => handleResetToDefault(key)}
                                    tabindex="0"
                                    type="button"
                                >
                                    <img src="tabler_icons/restore.svg" alt="Reset to Default" class="w-5 h-5 opacity-90 dark:invert" />
                                </button>
                            </div>
                        </div>
                    {/if}
                {/each}

                <!-- Folder Navigation Buttons -->
                <div class="flex flex-row gap-2 pt-4">
                    <button
                        class="px-4 py-2 bg-zinc-200 dark:bg-zinc-700 hover:bg-zinc-300 dark:hover:bg-zinc-600 text-zinc-800 dark:text-zinc-200 rounded-lg transition-colors text-sm font-medium shadow-sm border border-zinc-300 dark:border-zinc-600"
                        onclick={() => publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER, 'appdata')}
                    >
                        Open App Config Folder
                    </button>
                    <button
                        class="px-4 py-2 bg-zinc-200 dark:bg-zinc-700 hover:bg-zinc-300 dark:hover:bg-zinc-600 text-zinc-800 dark:text-zinc-200 rounded-lg transition-colors text-sm font-medium shadow-sm border border-zinc-300 dark:border-zinc-600"
                        onclick={() => publishMessage(PUBLIC_NATSSUBJECT_PIEBUTTON_OPENFOLDER, 'appfolder')}
                    >
                        Open App Folder
                    </button>
                </div>
            </div>
        {/if}
    </div>

    <!-- Action Buttons Footer -->
    <div class="flex-shrink-0 w-full p-2 bg-zinc-200/50 dark:bg-zinc-800/50 border-t border-zinc-200 dark:border-zinc-700">
        <div class="w-full flex flex-row justify-end items-center gap-2 px-6">
            <button
                aria-label="Undo"
                class="px-4 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer disabled:opacity-60 disabled:text-zinc-400 disabled:dark:text-zinc-500 hover:bg-zinc-300 dark:hover:bg-zinc-600 disabled:hover:bg-zinc-200 disabled:dark:hover:bg-zinc-700"
                onclick={handleUndo}
                type="button"
                disabled={undoHistory.length === 0}>
                Undo
            </button>
            <button
                aria-label="Discard Changes"
                class="px-4 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer disabled:opacity-60 disabled:text-zinc-400 disabled:dark:text-zinc-500 hover:bg-zinc-300 dark:hover:bg-zinc-600 disabled:hover:bg-zinc-200 disabled:dark:hover:bg-zinc-700"
                onclick={() => showDiscardConfirmDialog = true}
                type="button"
                disabled={undoHistory.length === 0}>
                Discard Changes
            </button>
            <button
                aria-label="Done"
                class="px-4 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer hover:bg-zinc-300 dark:hover:bg-zinc-600"
                onclick={() => goto('/')} type="button">
                Done
            </button>
        </div>
        <ConfirmationDialog
            cancelText="Save Changes"
            confirmText="Discard Changes"
            isOpen={showDiscardConfirmDialog}
            message="You have unsaved changes. What would you like to do?"
            onCancel={() => { showDiscardConfirmDialog = false; goto('/'); }}
            onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); goto('/'); }}
            title="Unsaved Changes"
        />
    </div>
</div>
