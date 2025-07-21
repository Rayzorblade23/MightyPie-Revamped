<!-- src/lib/components/piemenuConfig/ApplicationSelector.svelte -->
<script lang="ts">
    import type {InstalledAppsMap} from '$lib/data/installedAppsInfoManager.svelte.ts';
    import SelectorSvgIcon from '../SelectorSvgIcon.svelte';
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
        <div class="flex-shrink-0 h-[40px] w-[40px] flex items-center justify-center border border-zinc-200 dark:border-zinc-600 rounded bg-zinc-50 dark:bg-zinc-700">
            <SelectorSvgIcon iconPath={currentAppInfo?.iconPath} svgClasses="h-6 w-6 text-zinc-700 dark:text-zinc-200"
                             titleText={currentAppInfo?.iconPath || 'No icon'}/>
        </div>
        <select
                id="appNameSelect"
                class="block w-full pl-3 pr-10 py-2 text-base border-zinc-300 dark:border-zinc-600 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 dark:focus:border-indigo-400 sm:text-sm rounded-md shadow-sm bg-zinc-50 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-600 transition-colors"
                value={selectedAppName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedAppName !== ''}
                    class="bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100">-- Select an application --
            </option>
            {#each appSelectionKeys as appName (appName)}
                <option value={appName}
                        class="bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100">{appName}</option>
            {/each}
        </select>
    </div>
    {#if currentAppInfo?.exePath}
        <p class="text-xs text-zinc-500 dark:text-zinc-400 mt-1 pl-1 italic w-full flex items-center">
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