<!-- src/lib/components/piemenuConfig/FunctionSelector.svelte -->
<script lang="ts">
    import IconRenderer from '../IconRenderer.svelte';

    interface FunctionDefinition {
        icon_path: string;
        description?: string;
    }

    type AvailableFunctionsMap = Record<string, FunctionDefinition>;

    let {
        selectedFunctionName,
        functionDefinitions,
        onSelect
    } = $props<{
        selectedFunctionName: string; // The key of the selected function, e.g., "Maximize"
        functionDefinitions: AvailableFunctionsMap;
        onSelect: (functionKey: string) => void;
    }>();

    const functionSelectionKeys = $derived(Object.keys(functionDefinitions));
    const currentFunctionDef = $derived(selectedFunctionName ? functionDefinitions[selectedFunctionName] : undefined);

    function handleChange(event: Event) {
        const target = event.target as HTMLSelectElement;
        onSelect(target.value);
    }
</script>

<div class="mt-3 space-y-1">
    <label for="functionNameSelect" class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1">
        Select Function:
    </label>
    <div class="flex items-stretch space-x-2">
        <div class="flex-shrink-0 h-[40px] w-[40px] flex items-center justify-center border border-zinc-200 dark:border-zinc-600 rounded bg-zinc-50 dark:bg-zinc-700">
            <IconRenderer iconPath={currentFunctionDef?.icon_path}
                          svgClasses="h-6 w-6 text-zinc-700 dark:text-zinc-200"
                          titleText={currentFunctionDef?.icon_path || 'No icon'}/>
        </div>
        <select
                id="functionNameSelect"
                class="block w-full pl-3 pr-10 py-2 text-base border-zinc-300 dark:border-zinc-600 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 dark:focus:border-indigo-400 sm:text-sm rounded-md shadow-sm bg-zinc-50 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-600 transition-colors"
                value={selectedFunctionName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedFunctionName !== ''}
                    class="bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100">-- Select a function --
            </option>
            {#each functionSelectionKeys as funcKey (funcKey)}
                <option value={funcKey}
                        class="bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100">{funcKey}</option>
            {/each}
        </select>
    </div>
    {#if currentFunctionDef?.description}
        <p class="text-xs text-zinc-600 dark:text-zinc-400 mt-1 pl-1 italic">
            {currentFunctionDef.description}
        </p>
    {/if}
</div>