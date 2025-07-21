<!-- src/lib/components/piemenuConfig/ButtonInfoDisplay.svelte -->
<script lang="ts">
    import {type Button, ButtonType} from "$lib/data/types/pieButtonTypes.ts";
    import {getMenuConfiguration} from "$lib/data/configManager.svelte.ts";
    import {getDefaultButton} from "$lib/data/types/pieButtonDefaults.ts";

    import {getInstalledAppsInfo} from '$lib/data/installedAppsInfoManager.svelte.ts';

    // Child components
    import ButtonTypeSelector from './selectors/ButtonTypeSelector.svelte';
    import CallFunctionButtonConfig from './buttonConfigs/CallFunctionButtonConfig.svelte'; // New
    import ShowProgramButtonConfig from './buttonConfigs/ShowProgramButtonConfig.svelte'; // New
    import OpenPageButtonConfig from './buttonConfigs/OpenPageButtonConfig.svelte';
    import {PUBLIC_DIR_BUTTONFUNCTIONS} from "$env/static/public";

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
        [ButtonType.Disabled]: "Disabled",
    };
    const buttonTypeKeys = Object.keys(buttonTypeFriendlyNames) as ButtonType[];

    let availableFunctionsData = $state<AvailableFunctionsMap>({});

    let typedAvailableFunctions = $derived(availableFunctionsData);

    $effect(() => {
        fetch(PUBLIC_DIR_BUTTONFUNCTIONS)
            .then((response) => {
                if (!response.ok) {
                    throw new Error(`Failed to fetch buttonFunctions.json: ${response.statusText}`);
                }
                return response.json() as Promise<AvailableFunctionsMap>;
            })
            .then((data) => {
                availableFunctionsData = data;
            })
            .catch((error) => {
                console.error('Error loading buttonFunctions.json:', error);

            });
    });

    function getFriendlyButtonTypeName(buttonType: ButtonType | undefined): string {
        if (buttonType === undefined) {
            return "Unknown Type";
        }
        return buttonTypeFriendlyNames[buttonType] || buttonType.toString();
    }

    // Derived state
    let currentButtonLocal = $derived(selectedButtonDetails?.button); // The actual button object
    let currentButtonTypeValue = $derived(currentButtonLocal?.button_type); // Just the type for conditional rendering

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
    {@const isTrulyEmptySlot = button.button_type === ButtonType.Disabled && !getMenuConfiguration().get(menuID)?.get(pageID)?.has(buttonID)}

    <div class="p-4 border rounded-md shadow-sm bg-white dark:bg-zinc-800 text-zinc-800 dark:text-zinc-100 border-zinc-200 dark:border-zinc-700 w-full min-w-0">
        <div class="flex items-center justify-between mb-3">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white">Button Details</h2>
            <p class="text-right">Slot: {slotIndex + 1} <span
                    class="text-zinc-600 dark:text-zinc-400">(Page: {pageID + 1})</span></p>
        </div>
        <div class="text-sm space-y-2">
            {#if isTrulyEmptySlot && button.button_type === ButtonType.Disabled}
                <p class="text-yellow-700 dark:text-yellow-400 font-medium"><strong>Status:</strong> Empty Slot</p>
                <p class="text-zinc-600 dark:text-zinc-400 mb-2">Select a type below to configure this button.</p>
            {/if}

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
            {:else if button.button_type !== ButtonType.Disabled}
                <p class="text-zinc-600 mt-2">
                    {getFriendlyButtonTypeName(button.button_type)} has no other specific properties to configure here.
                </p>
            {/if}

        </div>
    </div>
{:else}
    <div class="p-4 text-center">
        <p class="text-zinc-500 dark:text-zinc-400">
            Select a button from a pie menu preview to see its details.
        </p>
    </div>
{/if}