<svelte:window on:contextmenu={(e) => e.preventDefault()}/>

<script lang="ts">
    import {onDestroy, onMount} from 'svelte';
    import {getSettings, publishSettings, type SettingsMap} from '$lib/data/settingsHandler.svelte.ts';
    import {goto} from "$app/navigation";
    import {getCurrentWindow, type Window} from "@tauri-apps/api/window";
    import {centerAndSizeWindowOnMonitor} from "$lib/windowUtils.ts";
    import {PUBLIC_SETTINGS_SIZE_X, PUBLIC_SETTINGS_SIZE_Y} from "$env/static/public";
    import ConfirmationDialog from '$lib/components/ui/ConfirmationDialog.svelte';

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

    // Update accent color CSS variables whenever settings change
    $effect(() => {
        if (!settings) return;
        const map = {
            colorAccentAnywin: '--color-accent-anywin',
            colorAccentProgramwin: '--color-accent-programwin',
            colorAccentLaunch: '--color-accent-launch',
            colorAccentFunction: '--color-accent-function'
        };
        for (const [key, cssVar] of Object.entries(map)) {
            if (settings[key]?.value) {
                document.documentElement.style.setProperty(cssVar, settings[key].value);
            }
        }
    });

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

    onMount(async () => {
        try {
            currentWindow = getCurrentWindow();
            await centerAndSizeWindowOnMonitor(
                currentWindow,
                Number(PUBLIC_SETTINGS_SIZE_X),
                Number(PUBLIC_SETTINGS_SIZE_Y)
            );
        } catch (error) {
            console.error("Failed to get/resize window onMount:", error);
        }
    });

    onDestroy(() => {
        publishSettings(settings);
    });
</script>

<div class="w-full min-h-screen flex flex-col bg-gray-100 dark:bg-gray-900 rounded-lg border-b border-gray-200 dark:border-gray-700">
    <!-- Title Bar -->
    <div class="title-bar relative flex items-center py-1 bg-slate-300 dark:bg-gray-800 rounded-t-lg border-b border-gray-200 dark:border-gray-700 h-8">
        <div class="w-0.5 min-w-[2px] h-full" data-tauri-drag-region="none"></div>
        <div class="absolute left-0 right-0 top-0 bottom-0 flex items-center justify-center pointer-events-none select-none">
            <span class="font-semibold text-sm lg:text-base text-gray-900 dark:text-gray-400">Settings</span>
        </div>
        <div class="flex-1 h-full" data-tauri-drag-region></div>
    </div>
    <div class="flex-1 w-full p-4 space-y-6 relative">
        {#if Object.keys(settings).length === 0}
            <p class="text-gray-500 dark:text-gray-400 bg-white/80 dark:bg-gray-800/70 rounded-lg shadow p-6 text-center text-lg font-medium">
                No settings available.</p>
        {:else}
            <div class="w-full">
                {#each Object.entries(settings) as [key, entry]}
                    {#if entry.isExposed}
                        <div class="flex flex-row items-center h-12 py-0 px-1 md:px-4 bg-gray-100 dark:bg-gray-800 rounded-lg mb-2 shadow-sm border border-gray-200 dark:border-gray-700">
                            <label class="w-1/2 md:w-1/3 text-gray-900 dark:text-gray-200 pr-4 pl-4 text-base" for={key}>{entry.label}</label>
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
                                                class="absolute left-0.5 top-0.5 w-5 h-5 bg-white dark:bg-gray-200 rounded-full shadow transition-transform duration-200"
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
                                            <span class="absolute left-2 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none select-none">#</span>
                                            <input type="text"
                                                   class="bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-700 rounded-lg pl-6 pr-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all shadow-sm text-gray-900 dark:text-gray-100 w-full"
                                                   value={entry.value ? entry.value.replace(/^#/, '') : ''}
                                                   oninput={e => handleHexInput(key, e)}
                                                   maxlength="6"
                                                   placeholder="RRGGBB" />
                                        </div>
                                    </div>
                                {:else if entry.type === 'number' || entry.type === 'float'}
                                    <input type="number" id={key}
                                           class="bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-gray-900 dark:text-gray-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)} />
                                {:else if entry.type === 'int' || entry.type === 'integer'}
                                    <input type="number" id={key}
                                           step="1"
                                           inputmode="numeric"
                                           pattern="^-?\\d+$"
                                           class="bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-gray-900 dark:text-gray-100"
                                           value={entry.value}
                                           onchange={e => handleNumberChange(e, key)}
                                           onkeydown={handleIntKeydown} />
                                {:else if entry.type === 'string'}
                                    <input type="text" id={key}
                                           class="bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-gray-900 dark:text-gray-100"
                                           value={entry.value}
                                           onchange={e => handleStringInput(key, e)}/>
                                {:else if entry.type === 'enum' && entry.options}
                                    <select id={key}
                                            class="bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all w-full shadow-sm text-gray-900 dark:text-gray-100"
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
                                    class="w-8 h-8 flex items-center justify-center p-0 rounded bg-white/80 dark:bg-gray-700/80 border border-gray-300 dark:border-gray-600 shadow hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors flex-shrink-0"
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
            </div>
        {/if}
        <!-- Done Button, floating at bottom right -->
        <div class="fixed bottom-6 right-8 z-50 flex flex-row gap-2">
            <button
                aria-label="Undo"
                class="px-4 py-2 rounded border border-gray-300 dark:border-gray-700 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer disabled:opacity-60 disabled:text-gray-400 disabled:dark:text-gray-500 hover:bg-gray-300 dark:hover:bg-gray-600 disabled:hover:bg-gray-200 disabled:dark:hover:bg-gray-700"
                onclick={handleUndo}
                type="button"
                disabled={undoHistory.length === 0}
            >
                Undo
            </button>
            <button
                aria-label="Discard Changes"
                class="px-4 py-2 rounded border border-gray-300 dark:border-gray-700 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer disabled:opacity-60 disabled:text-gray-400 disabled:dark:text-gray-500 hover:bg-gray-300 dark:hover:bg-gray-600 disabled:hover:bg-gray-200 disabled:dark:hover:bg-gray-700"
                onclick={() => showDiscardConfirmDialog = true}
                type="button"
                disabled={undoHistory.length === 0}
            >
                Discard Changes
            </button>
            <button
                aria-label="Done"
                class="px-4 py-2 rounded border border-gray-300 dark:border-gray-700 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 font-semibold text-lg transition-colors focus:outline-none cursor-pointer hover:bg-gray-300 dark:hover:bg-gray-600"
                onclick={() => goto('/')} type="button">
                Done
            </button>
        </div>
        <ConfirmationDialog
            isOpen={showDiscardConfirmDialog}
            title="Discard All Changes?"
            message="This will reset all settings since you opened this window. Are you sure?"
            confirmText="Discard"
            cancelText="Cancel"
            onConfirm={() => { showDiscardConfirmDialog = false; discardChanges(); }}
            onCancel={() => showDiscardConfirmDialog = false}
        />
    </div>
</div>
