<!-- PieButtonSettings.svelte -->
<script lang="ts">

    // Props
    import type {ButtonData} from "$lib/components/piebutton/piebuttonTypes.ts";

    let {
        taskData,         // The configuration for this button, or undefined if the slot is empty
        isSelected = false, // Whether this button is currently selected in the UI
        onClick,          // Callback function when this button is clicked
        // width and height can be passed if variable, or styled directly below if fixed for settings
    }: {
        taskData: ButtonData | undefined;
        isSelected?: boolean;
        onClick: () => void;
        // width?: number; // e.g., in rem
        // height?: number; // e.g., in rem
    } = $props();

    // Derived display properties from taskData
    const properties = $derived(taskData?.properties);
    const buttonTextUpper = $derived(properties?.button_text_upper || (taskData ? "Action" : "")); // Default to "Action" if data exists but no text
    const buttonTextLower = $derived(properties?.button_text_lower || "");
    const iconPath = $derived(properties?.icon_path);

    // SVG loading logic (similar to original PieButton, can be refined)
    const ICON_CLASSES = "h-[1.75rem] w-[1.75rem] flex-shrink-0 mr-1 object-contain"; // Added object-contain
    const ICON_LOADING_PLACEHOLDER_SVG = `<div class="${ICON_CLASSES} animate-pulse bg-slate-600 rounded"></div>`;
    const ICON_ERROR_PLACEHOLDER_SVG = `<div class="${ICON_CLASSES} flex items-center justify-center text-red-500">⚠️</div>`;

    let svgContentPromise = $state<Promise<string> | null>(null);

    $effect(() => {
        const currentIconPath = properties?.icon_path; // Re-check from properties as taskData might change
        if (currentIconPath?.endsWith('.svg')) {
            svgContentPromise = fetch(currentIconPath)
                .then(response => {
                    if (!response.ok) {
                        console.error(`Failed to load SVG (${currentIconPath}): ${response.status}`);
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    return response.text();
                })
                .catch(error => {
                    console.error(`Error fetching SVG (${currentIconPath}):`, error);
                    return ICON_ERROR_PLACEHOLDER_SVG; // Return error placeholder HTML
                });
        } else {
            svgContentPromise = null; // Reset if not an SVG or path changes
        }
    });
</script>

<button
        type="button"
        class="flex items-center p-1 min-w-0 w-full h-full border-2 rounded transition-colors"
        class:border-sky-500={isSelected && taskData}
        class:ring-2={isSelected && taskData}
        class:ring-sky-400={isSelected && taskData}
        class:border-slate-600={!isSelected && taskData}
        class:hover:border-slate-500={!isSelected && taskData}
        class:bg-slate-700={taskData && !isSelected}
        class:hover:bg-slate-650={taskData && !isSelected}
        class:bg-sky-700={isSelected && taskData}
        class:border-dashed={!taskData}
        class:border-slate-700={!taskData}
        class:hover:border-slate-600={!taskData}
        class:bg-slate-800={!taskData}
        class:cursor-not-allowed={!taskData}
        onclick={taskData ? onClick : undefined}
        disabled={!taskData}
        title={taskData ? (buttonTextUpper || 'Configure Action') : 'Empty Slot'}
>
    {#if taskData}
        <!-- Icon Area -->
        {#if iconPath}
            {#if iconPath.endsWith('.svg')}
                {#await svgContentPromise}
                    {@html ICON_LOADING_PLACEHOLDER_SVG}
                {:then svgString}
                    <span class="{ICON_CLASSES} inline-flex items-center justify-center">{@html svgString}</span>
                {:catch _error}
                    {@html ICON_ERROR_PLACEHOLDER_SVG}
                {/await}
            {:else}
                <img src={iconPath} alt="icon" class={ICON_CLASSES}/>
            {/if}
        {:else}
            <!-- Placeholder if no icon but there is task data -->
            <div class="{ICON_CLASSES} flex items-center justify-center text-slate-500 text-2xl">✲</div>
        {/if}

        <!-- Text Area -->
        <div class="flex flex-col flex-1 min-w-0 text-left ml-1">
            <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-sm leading-tight font-medium">
                {buttonTextUpper || "Unnamed Action"}
            </span>
            {#if buttonTextLower}
                <span class="w-full whitespace-nowrap overflow-hidden text-ellipsis text-xs leading-tight text-slate-300">
                    {buttonTextLower}
                </span>
            {/if}
        </div>
    {:else}
        <span class="text-xs text-slate-500 w-full text-center">Empty</span>
    {/if}
</button>