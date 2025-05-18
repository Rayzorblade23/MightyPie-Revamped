<!-- src/lib/components/piemenuConfig/ApplicationSelector.svelte -->
<script lang="ts">
    import type {InstalledAppsMap} from '$lib/data/installedAppsInfoManager.svelte.ts';
    import SvgIcon from './SvgIcon.svelte';

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
    <label for="appNameSelect" class="block text-sm font-medium text-gray-700 mb-1">
        Select Application:
    </label>
    <div class="flex items-center space-x-2">
        <div class="flex-shrink-0 w-6 h-6 flex items-center justify-center border rounded bg-gray-50">
            <SvgIcon iconPath={currentAppInfo?.iconPath} svgClasses="h-4 w-4"
                     titleText={currentAppInfo?.iconPath || 'No icon'}/>
        </div>
        <select
                id="appNameSelect"
                class="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md shadow-sm"
                value={selectedAppName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedAppName !== ''}>-- Select an application --</option>
            {#each appSelectionKeys as appName (appName)}
                <option value={appName}>{appName}</option>
            {/each}
        </select>
    </div>
    {#if currentAppInfo?.exePath}
        <p class="text-xs text-gray-500 mt-1 pl-1 italic">Path: {currentAppInfo.exePath}</p>
    {/if}
</div>