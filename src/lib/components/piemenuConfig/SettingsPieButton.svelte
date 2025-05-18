<!-- SettingsPieButton.svelte -->
<script lang="ts">
    import {type ButtonType} from '$lib/data/piebuttonTypes.ts';

    // Helper type for properties (can be moved to piebuttonTypes.ts if used elsewhere)
    type ButtonPropertiesUnion =
        | import('$lib/data/piebuttonTypes.ts').ShowProgramWindowProperties
        | import('$lib/data/piebuttonTypes.ts').ShowAnyWindowProperties
        | import('$lib/data/piebuttonTypes.ts').CallFunctionProperties
        | import('$lib/data/piebuttonTypes.ts').LaunchProgramProperties;

    // Using $props rune for props declaration
    let {
        x,
        y,
        width,
        height,
        taskType,
        properties,
        buttonTextUpper = '',
        buttonTextLower = '',
        onclick
    } = $props<{
        // Layout Props
        x: number;
        y: number;
        width: number; // Width in rem
        height: number; // Height in rem

        // Display Props
        taskType: ButtonType | 'empty';
        properties: ButtonPropertiesUnion | undefined;
        buttonTextUpper?: string;
        buttonTextLower?: string;
        onclick?: (event: MouseEvent) => void;
    }>();

    let svgPromise = $state<Promise<string> | undefined>(undefined);

    $effect(() => {
        if (properties?.icon_path?.endsWith('.svg')) {
            // console.log(`Fetching SVG: ${properties.icon_path} for button ${buttonTextUpper}`);
            svgPromise = fetch(properties.icon_path)
                .then(r => {
                    if (!r.ok) throw new Error(`Failed to fetch SVG: ${r.status} ${r.statusText}`);
                    return r.text();
                })
                .then(text => text.replace(/<svg /, '<svg class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1" '))
                .catch(err => {
                    console.error(`Error fetching/processing SVG ${properties.icon_path}:`, err);
                    return `<svg class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1" viewBox="0 0 24 24"><path fill="red" d="M12 2 L2 22 L22 22 Z M11 10 L13 10 L13 16 L11 16 Z M11 18 L13 18 L13 20 L11 20 Z"></path></svg>`; // Error SVG
                });
        } else {
            svgPromise = undefined; // Reset if not an SVG or no icon_path
        }
    });

</script>

<!-- The outer div handles absolute positioning using x, y passed as props -->
<!-- These x, y are offsets from the center of the pie menu container -->
<div class="absolute" style="left: {x}px; top: {y}px; transform: translate(-50%, -50%);">
    <button
        type="button"
        class="bg-amber-400 flex items-center p-0.5 min-w-0 rounded-md shadow-md"
        style="width: {width}rem; height: {height}rem;"
        onclick={onclick}
    >
        {#if properties?.icon_path}
            {#if properties.icon_path.endsWith('.svg')}
                {#if svgPromise}
                    {#await svgPromise}
                        <div class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 animate-pulse bg-gray-300 rounded"></div>
                        <!-- Placeholder -->
                    {:then svgContent}
                        {@html svgContent}
                    {:catch error}
                        <div class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 text-red-500" title={error?.message || 'Error loading SVG'}>⚠️</div>
                        <!-- Error icon -->
                    {/await}
                {/if}
            {:else}
                <img src={properties.icon_path} alt="icon"
                     class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 object-contain"/>
            {/if}
        {:else if taskType === 'empty'}
            <span class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 flex items-center justify-center text-lg">➕</span>
        {:else if taskType === 'disabled'}
            <span class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 flex items-center justify-center text-lg">🚫</span>
        {:else}
            <span class="h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 flex items-center justify-center text-lg">🔲</span>
        {/if}

        <span class="flex flex-col flex-1 min-w-0">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight">{buttonTextUpper}</span>
            {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-xs leading-tight">{buttonTextLower}</span>
            {/if}
        </span>
    </button>
</div>