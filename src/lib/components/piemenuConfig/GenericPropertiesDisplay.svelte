<!-- src/lib/components/piemenuConfig/GenericPropertiesDisplay.svelte -->
<script lang="ts">
    import {ButtonType} from '$lib/data/piebuttonTypes.ts';

    let {
        displayableProperties,
        buttonType,
        getPropertyFriendlyNameFn, // This prop IS STRONGLY TYPED via $props
        dropdownPropertyKeys = [],
        friendlyButtonTypeName = "This button type"
    } = $props<{
        displayableProperties: Record<string, any>;
        buttonType: ButtonType;
        getPropertyFriendlyNameFn: (key: string, type: ButtonType) => string; // Correct type
        dropdownPropertyKeys?: string[];
        friendlyButtonTypeName?: string;
    }>();

    let friendlyDropdownKeysString = $derived(() => {
        return dropdownPropertyKeys.map((key: string) => getPropertyFriendlyNameFn(key, buttonType)).join(', ');
    });
</script>

{#if Object.keys(displayableProperties).length > 0}
    <p class="font-semibold mt-3 mb-1">Configurable Properties:</p>
    <dl class="space-y-2 mt-1 pl-2">
        {#each Object.entries(displayableProperties) as [key, val]}
            <div>
                <dt class="font-medium text-gray-700">
                    {getPropertyFriendlyNameFn(key, buttonType)}:
                </dt>
                <dd class="ml-4 text-gray-600 bg-white border border-gray-200 px-2 py-1 rounded text-xs inline-block shadow-sm">
                    {#if val === "" && key === 'button_text_lower' && buttonType === ButtonType.CallFunction}
                        <span class="italic text-gray-400">(empty - as intended)</span>
                    {:else if val === ""}
                        <span class="italic text-gray-400">(empty)</span>
                    {:else}
                        {String(val)}
                    {/if}
                </dd>
            </div>
        {/each}
    </dl>
{:else if dropdownPropertyKeys.length > 0}
    <p class="text-gray-600 mt-2">
        {friendlyButtonTypeName} allows configuration for:
        <span class="font-medium">
            {friendlyDropdownKeysString}
        </span>.
        <br/>These properties are not currently set with values or are empty.
    </p>
{:else}
    <p class="text-gray-600 mt-2">
        {friendlyButtonTypeName} has no other specific properties to configure here.
    </p>
{/if}