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
        <div class="flex-shrink-0 h-[40px] w-[40px] flex items-center justify-center border border-none rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800">
            <IconRenderer iconPath={currentFunctionDef?.icon_path}
                          svgClasses="h-6 w-6 text-zinc-700 dark:text-zinc-200"
                          titleText={currentFunctionDef?.icon_path || 'No icon'}/>
        </div>
        <select
                id="functionNameSelect"
                class="custom-select block w-full pl-3 py-2 text-base border-none focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-700 transition-colors"
                value={selectedFunctionName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedFunctionName !== ''}
                    class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">-- Select a function --
            </option>
            {#each functionSelectionKeys as funcKey (funcKey)}
                <option value={funcKey}
                        class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">{funcKey}</option>
            {/each}
        </select>
    </div>
    {#if currentFunctionDef?.description}
        <p class="text-xs text-zinc-700 mt-3 dark:text-zinc-400 pl-1 italic">
            {currentFunctionDef.description}
        </p>
    {/if}
</div>