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

<div>
    <select
            id="buttonTypeSelect"
            class="custom-select block w-full px-4 py-2 border-none focus:outline-none focus:ring-2 focus:ring-amber-400 text-sm rounded-lg shadow-md disabled:bg-zinc-100 dark:disabled:bg-zinc-900 disabled:cursor-not-allowed bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-700 transition-colors"
            value={currentType ?? ''}
            onchange={handleChange}
            {disabled}
    >
        {#each buttonTypeKeys as typeValue (typeValue)}
            <option value={typeValue}
                    class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">{buttonTypeFriendlyNames[typeValue]}</option>
        {/each}
    </select>
</div>