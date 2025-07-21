<!-- src/lib/components/piemenuConfig/ButtonTypeSelector.svelte -->
<script lang="ts">
    import type {ButtonType} from '$lib/data/types/pieButtonTypes.ts';

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
    <label for="buttonTypeSelect" class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1">
        Button Type:
    </label>
    <select
            id="buttonTypeSelect"
            class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-zinc-300 dark:border-zinc-600 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 dark:focus:border-indigo-400 sm:text-sm rounded-md shadow-sm disabled:bg-zinc-100 dark:disabled:bg-zinc-900 disabled:cursor-not-allowed bg-zinc-50 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-600 transition-colors"
            value={currentType ?? ''}
            onchange={handleChange}
            {disabled}
    >
        {#each buttonTypeKeys as typeValue (typeValue)}
            <option value={typeValue}
                    class="bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100">{buttonTypeFriendlyNames[typeValue]}</option>
        {/each}
    </select>
</div>