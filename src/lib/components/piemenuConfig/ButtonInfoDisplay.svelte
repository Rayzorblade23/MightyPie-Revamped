<!-- src/lib/components/piemenuConfig/ButtonInfoDisplay.svelte -->
<script lang="ts">
    import {type Button, ButtonType} from "$lib/data/types/pieButtonTypes.ts";
    import {getPieMenuButtons} from "$lib/data/configManager.svelte.ts";
    import {getDefaultButton} from "$lib/data/types/pieButtonDefaults.ts";
    import {createLogger} from "$lib/logger";
    import {getInstalledAppsInfo} from '$lib/data/installedAppsInfoManager.svelte.ts';
    import {onMount} from 'svelte';

    // Child components
    import ButtonTypeSelector from './selectors/ButtonTypeSelector.svelte';
    import CallFunctionButtonConfig from './buttonConfigs/CallFunctionButtonConfig.svelte'; // New
    import ShowProgramButtonConfig from './buttonConfigs/ShowProgramButtonConfig.svelte'; // New
    import OpenPageButtonConfig from './buttonConfigs/OpenPageButtonConfig.svelte';
    import OpenResourceButtonConfig from './buttonConfigs/OpenResourceButtonConfig.svelte';
    import {getButtonFunctions} from "$lib/fileAccessUtils.ts";

    // Create a logger for this component
    const logger = createLogger('ButtonInfoDisplay');

    let {
        selectedButtonDetails,
        onConfigChange,
        menuConfig
    } = $props<{
        selectedButtonDetails: {
            menuID: number; pageID: number; buttonID: number; slotIndex: number; button: Button;
        } | undefined;
        onConfigChange: (payload: {
            menuID: number; pageID: number; buttonID: number; newButton: Button;
        }) => void;
        menuConfig: any;
    }>();

    // Data sources, passed down
    interface FunctionDefinition {
        function_name: string;
        icon_path: string;
        description?: string;
    }

    type AvailableFunctionsMap = Record<string, FunctionDefinition>;

    const installedAppsMap = getInstalledAppsInfo();

    const buttonTypeFriendlyNames: Record<ButtonType, string> = {
        [ButtonType.ShowProgramWindow]: "Show Program Window",
        [ButtonType.ShowAnyWindow]: "Show Any Window",
        [ButtonType.CallFunction]: "Call Function",
        [ButtonType.LaunchProgram]: "Launch Program",
        [ButtonType.OpenSpecificPieMenuPage]: "Open Page",
        [ButtonType.OpenResource]: "Open Resource",
        [ButtonType.Disabled]: "Disabled",
    };

    // Add explanations for each button type
    const buttonTypeExplanations: Record<ButtonType, string> = {
        [ButtonType.ShowProgramWindow]: "This button is assigned windows that belong to the specified application.\n\nLeft-click brings the window to the foreground.\nLeft-click while the window is focused will minimize the window.\nMiddle-click closes the window.",
        [ButtonType.ShowAnyWindow]: "This button is assigned any open window.\n\nLeft-click brings the window to the foreground.\nLeft-click while the window is focused will minimize the window.\nMiddle-click closes the window.",
        [ButtonType.CallFunction]: "Executes a predefined function from the available functions list.",
        [ButtonType.LaunchProgram]: "Launches a program from the list of installed applications.",
        [ButtonType.OpenSpecificPieMenuPage]: "Opens any page in any menu.\nDisplays a custom text label.",
        [ButtonType.OpenResource]: "Opens a file or folder specified by the resource path, using the default application.\nDisplays a custom text label.",
        [ButtonType.Disabled]: "This button is disabled and will not perform any action when clicked.",
    };

    const buttonTypeKeys = Object.keys(buttonTypeFriendlyNames) as ButtonType[];

    let availableFunctionsData = $state<AvailableFunctionsMap>({});

    let typedAvailableFunctions = $derived(availableFunctionsData);

    // State for tooltip visibility
    let showTooltip = $state(false);
    let questionMarkButton = $state<HTMLElement | null>(null);

    // Function to toggle tooltip visibility
    function toggleTooltip(event: MouseEvent) {
        event.stopPropagation();
        showTooltip = !showTooltip;
    }

    // Close tooltip when clicking outside
    function handleClickOutside(event: MouseEvent) {
        if (showTooltip && 
            questionMarkButton && 
            event.target instanceof Node && 
            !questionMarkButton.contains(event.target)) {
            showTooltip = false;
        }
    }

    // Add global click handler when component is mounted
    onMount(() => {
        document.addEventListener('click', handleClickOutside);

        return () => {
            document.removeEventListener('click', handleClickOutside);
        };
    });

    $effect(() => {
        (async () => {
            try {
                // Get the buttonFunctions.json parsed data using the utility function
                availableFunctionsData = await getButtonFunctions<AvailableFunctionsMap>();
            } catch (error) {
                logger.error('Error loading buttonFunctions:', error);
            }
        })();
    });

    // Load button functions data when component initializes
    (async () => {
        try {
            availableFunctionsData = await getButtonFunctions<AvailableFunctionsMap>();
        } catch (error) {
            logger.error('Error loading buttonFunctions.json:', error);
        }
    })();

    function getFriendlyButtonTypeName(buttonType: ButtonType | undefined): string {
        if (buttonType === undefined) {
            return "Unknown Type";
        }
        return buttonTypeFriendlyNames[buttonType] || buttonType.toString();
    }

    // Derived state
    let currentButtonLocal = $derived(selectedButtonDetails?.button); // The actual button object
    let currentButtonTypeValue = $derived(currentButtonLocal?.button_type); // Just the type for conditional rendering

    // Get current button type explanation
    let currentButtonTypeExplanation = $derived(
        currentButtonTypeValue !== undefined ? buttonTypeExplanations[currentButtonTypeValue as ButtonType] : ""
    );

    // Event Handlers
    function handleTypeChange(newType: ButtonType) {
        if (!selectedButtonDetails) return;
        const {menuID, pageID, buttonID} = selectedButtonDetails;

        let newButtonDefaultConfig = getDefaultButton(newType);

        if (
            (newType === ButtonType.ShowProgramWindow || newType === ButtonType.LaunchProgram) &&
            'properties' in newButtonDefaultConfig && newButtonDefaultConfig.properties
        ) {
            const appName = newType === ButtonType.ShowProgramWindow
                ? newButtonDefaultConfig.properties.button_text_lower
                : newButtonDefaultConfig.properties.button_text_upper;

            const appInfo = installedAppsMap.get(appName || "");
            if (appInfo) {
                newButtonDefaultConfig.properties.icon_path = appInfo.iconPath || "";
            }
        } else if (newType === ButtonType.CallFunction && 'properties' in newButtonDefaultConfig && newButtonDefaultConfig.properties) {
            const functionName = newButtonDefaultConfig.properties.button_text_upper;
            const functionInfo = typedAvailableFunctions[functionName || ""];
            if (functionInfo) {
                newButtonDefaultConfig.properties.icon_path = functionInfo.icon_path || "";
            }
        }

        onConfigChange({menuID: menuID, pageID: pageID, buttonID: buttonID, newButton: newButtonDefaultConfig});
    }

    // Generic handler for updates from CallFunctionConfig or ProgramButtonConfig
    function handleSpecificConfigUpdate(updatedButton: Button) {
        if (!selectedButtonDetails) return;
        const {menuID, pageID, buttonID} = selectedButtonDetails;
        onConfigChange({menuID: menuID, pageID: pageID, buttonID: buttonID, newButton: updatedButton});
    }

    function handleButtonChange(updatedButton: Button) {
        if (!selectedButtonDetails) return;
        const {menuID, pageID, buttonID} = selectedButtonDetails;
        onConfigChange({menuID: menuID, pageID: pageID, buttonID: buttonID, newButton: updatedButton});
    }

</script>

{#if selectedButtonDetails && currentButtonLocal} <!-- Ensure currentButtonLocal is defined -->
    {@const {menuID, pageID, buttonID, slotIndex} = selectedButtonDetails}
    {@const button = currentButtonLocal}
    {@const isTrulyEmptySlot = button.button_type === ButtonType.Disabled && !getPieMenuButtons().get(menuID)?.get(pageID)?.has(buttonID)}

    <div class="p-4 border border-none bg-zinc-200/60 dark:bg-neutral-900/60 opacity-90 rounded-xl shadow-md w-full min-w-0">
        <div class="flex items-center justify-between mb-3">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white">Button Details</h2>
            <p class="text-right text-zinc-600 dark:text-zinc-400">Slot: {slotIndex + 1} <span
                    class="text-zinc-600 dark:text-zinc-400">(Page: {pageID + 1})</span></p>
        </div>
        <div class="text-sm space-y-2">
            {#if isTrulyEmptySlot && button.button_type === ButtonType.Disabled}
                <p class="text-yellow-700 dark:text-yellow-400 font-medium"><strong>Status:</strong> Empty Slot</p>
                <p class="text-zinc-600 dark:text-zinc-300 mb-2">Select a type below to configure this button.</p>
            {/if}

            {#if !selectedButtonDetails}
                <p class="text-zinc-600 dark:text-zinc-300 mb-2">Select a button to configure.</p>
            {:else if !currentButtonTypeValue}
                <p class="text-zinc-600 dark:text-zinc-300 mb-2">Select a type below to configure this button.</p>
            {/if}

            <div class="flex justify-between items-center mb-1">
                <span class="text-sm font-medium text-zinc-700 dark:text-zinc-400">Button Type:</span>
                <div class="relative">
                    <button
                            class="flex items-center justify-center w-4 h-4 mr-1 rounded-full bg-purple-800 dark:bg-purple-950 text-zinc-100 hover:bg-violet-800 dark:hover:bg-violet-950 active:bg-purple-700 dark:active:bg-indigo-950 transition-colors text-xs font-medium"
                            onclick={toggleTooltip}
                            aria-label="Show button type explanation"
                            bind:this={questionMarkButton}
                    >
                        ?
                    </button>
                </div>
            </div>

            <ButtonTypeSelector
                    currentType={currentButtonTypeValue}
                    {buttonTypeKeys}
                    {buttonTypeFriendlyNames}
                    disabled={!selectedButtonDetails}
                    onChange={handleTypeChange}
            />

            {#if currentButtonTypeValue === ButtonType.CallFunction}
                <CallFunctionButtonConfig
                        button={button}
                        functionDefinitions={typedAvailableFunctions}
                        onUpdate={handleSpecificConfigUpdate}
                />
            {:else if currentButtonTypeValue === ButtonType.ShowProgramWindow || currentButtonTypeValue === ButtonType.LaunchProgram}
                <ShowProgramButtonConfig
                        button={button}
                        {installedAppsMap}
                        onUpdate={handleSpecificConfigUpdate}
                />
            {:else if currentButtonTypeValue === ButtonType.OpenSpecificPieMenuPage}
                <OpenPageButtonConfig button={button} menuConfig={menuConfig} onUpdate={handleButtonChange}/>
            {:else if currentButtonTypeValue === ButtonType.OpenResource}
                <OpenResourceButtonConfig button={button} onUpdate={handleButtonChange}/>
            {:else if button.button_type !== ButtonType.Disabled}
                <p class="text-zinc-600 dark:text-zinc-300 mt-2">
                    {getFriendlyButtonTypeName(button.button_type)} has no other specific properties to configure here.
                </p>
            {/if}

        </div>
    </div>
{:else}
    <div class="p-4 text-center">
        <p class="text-zinc-500 dark:text-zinc-300">
            Select a button from a pie menu preview to see its details.
        </p>
    </div>
{/if}

{#if showTooltip && questionMarkButton}
    {@const buttonRect = questionMarkButton.getBoundingClientRect()}
    <div class="fixed inset-0 z-[100] pointer-events-none">
        <div 
            class="absolute bg-white dark:bg-zinc-800 p-3 rounded-md shadow-lg w-80 text-sm text-zinc-800 dark:text-zinc-200 border border-zinc-200 dark:border-zinc-700 whitespace-pre-line pointer-events-auto"
            style="left: {Math.max(10, buttonRect.right - 320)}px; top: {Math.max(10, buttonRect.top - 10)}px; transform: translateY(-100%);"
        >
            {currentButtonTypeExplanation}
        </div>
    </div>
{/if}