<!-- src/lib/components/piemenuConfig/ButtonTypeSelector.svelte -->
<script lang="ts">
    import type {ButtonType} from '$lib/data/piebuttonTypes.ts';

    let {
        currentType,
        buttonTypeKeys,
        buttonTypeFriendlyNames,
        disabled = false,
        onChange
    } = $props<{
        currentType: ButtonType | undefined;
        buttonTypeKeys: ButtonType[];
        buttonTypeFriendlyNames: Record<ButtonType, string>;
        disabled?: boolean;
        onChange: (newType: ButtonType) => void;
    }>();

    function handleChange(event: Event) {
        const target = event.target as HTMLSelectElement;
        onChange(target.value as ButtonType);
    }
</script>

<div class="mt-1">
    <label for="buttonTypeSelect" class="block text-sm font-medium text-gray-700 mb-1">
        Button Type:
    </label>
    <select
            id="buttonTypeSelect"
            class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md shadow-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
            value={currentType ?? ''}
            onchange={handleChange}
            {disabled}
    >
        {#each buttonTypeKeys as typeValue (typeValue)}
            <option value={typeValue}>{buttonTypeFriendlyNames[typeValue]}</option>
        {/each}
    </select>
</div>