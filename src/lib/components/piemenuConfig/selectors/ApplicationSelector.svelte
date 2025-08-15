<!-- src/lib/components/piemenuConfig/ApplicationSelector.svelte -->
<script lang="ts">
    import type {InstalledAppsMap} from '$lib/data/installedAppsInfoManager.svelte.ts';
    import IconRenderer from '../IconRenderer.svelte';
    import {middleEllipsis} from '../middleEllipsisAction.ts';

    let {
        selectedAppName,
        installedAppsMap,
        onSelect
    } = $props<{
        selectedAppName: string;
        installedAppsMap: InstalledAppsMap;
        onSelect: (appName: string) => void;
    }>();

    const appSelectionKeys = $derived(Array.from(installedAppsMap.keys()).sort());
    const currentAppInfo = $derived(selectedAppName ? installedAppsMap.get(selectedAppName) : undefined);

    function handleChange(event: Event) {
        const target = event.target as HTMLSelectElement;
        onSelect(target.value);
    }
</script>

<div class="mt-3 space-y-1">
    <label for="appNameSelect" class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1">
        Select Application:
    </label>
    <div class="flex items-stretch space-x-2">
        <div class="flex-shrink-0 h-[40px] w-[40px] flex items-center justify-center border border-none shadow-sm rounded-lg bg-zinc-200 dark:bg-zinc-800">
            <IconRenderer iconPath={currentAppInfo?.iconPath} svgClasses="h-6 w-6 text-zinc-700 dark:text-zinc-200"
                          titleText={currentAppInfo?.iconPath || 'No icon'}/>
        </div>
        <select
                id="appNameSelect"
                class="custom-select block w-full pl-3 py-2 text-base border-none focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-700 transition-colors custom-select"
                value={selectedAppName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedAppName !== ''}
                    class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">-- Select an application --
            </option>
            {#each appSelectionKeys as appName (appName)}
                <option value={appName}
                        class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">{appName}</option>
            {/each}
        </select>
    </div>
    {#if currentAppInfo?.exePath}
        <p class="text-xs text-zinc-800 dark:text-zinc-400 mt-3 pl-1 italic w-full flex items-center">
            Path:
            <span
                    use:middleEllipsis={currentAppInfo.exePath}
                    class="ml-1 flex-1 min-w-0 overflow-hidden whitespace-nowrap text-ellipsis cursor-pointer middle-ellipsis"
                    title={currentAppInfo.exePath}
            >
                {currentAppInfo.exePath}
            </span>
        </p>
    {/if}
</div>