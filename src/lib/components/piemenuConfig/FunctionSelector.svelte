<!-- src/lib/components/piemenuConfig/FunctionSelector.svelte -->
<script lang="ts">
    import SvgIcon from './SvgIcon.svelte';

    interface FunctionDefinition {
        function_name: string; // Not used for display directly but part of the data
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
    <label for="functionNameSelect" class="block text-sm font-medium text-gray-700 mb-1">
        Select Function:
    </label>
    <div class="flex items-center space-x-2">
        <div class="flex-shrink-0 w-6 h-6 flex items-center justify-center border rounded bg-gray-50">
            <SvgIcon iconPath={currentFunctionDef?.icon_path} svgClasses="h-4 w-4"
                     titleText={currentFunctionDef?.icon_path || 'No icon'}/>
        </div>
        <select
                id="functionNameSelect"
                class="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md shadow-sm"
                value={selectedFunctionName}
                onchange={handleChange}
        >
            <option value="" disabled={selectedFunctionName !== ''}>-- Select a function --</option>
            {#each functionSelectionKeys as funcKey (funcKey)}
                <option value={funcKey}>{funcKey}</option>
            {/each}
        </select>
    </div>
    {#if currentFunctionDef?.description}
        <p class="text-xs text-gray-600 mt-1 pl-1 italic">
            {currentFunctionDef.description}
        </p>
    {/if}
</div>