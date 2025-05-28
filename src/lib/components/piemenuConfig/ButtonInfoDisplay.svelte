<!-- src/lib/components/piemenuConfig/ButtonInfoDisplay.svelte -->
<script lang="ts">
    import {type Button, ButtonType} from "$lib/data/piebuttonTypes.ts";
    import {getMenuConfiguration} from "$lib/data/configHandler.svelte.ts";
    import {getDefaultButton, getDropdownFields} from "$lib/data/pieButtonDefaults.ts";

    import {getInstalledAppsInfo} from '$lib/data/installedAppsInfoManager.svelte.ts';

    // Child components
    import ButtonTypeSelector from './ButtonTypeSelector.svelte';
    import CallFunctionConfig from './CallFunctionConfig.svelte'; // New
    import ProgramButtonConfig from './ProgramButtonConfig.svelte'; // New
    import GenericPropertiesDisplay from './GenericPropertiesDisplay.svelte';
    import {PUBLIC_DIR_BUTTONFUNCTIONS} from "$env/static/public";

    let {
        selectedButtonDetails,
        onConfigChange // This is the prop to notify the page of any change
    } = $props<{
        selectedButtonDetails: {
            menuID: number; pageID: number; buttonID: number; slotIndex: number; button: Button;
        } | undefined;
        onConfigChange: (payload: {
            menuID: number; pageID: number; buttonID: number; newButton: Button;
        }) => void;
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

    // Helper functions needed by GenericPropertiesDisplay or this component
    function getPropertyFriendlyName(propertyKey: string, buttonType: ButtonType): string {
        // ... (full implementation as before)
        switch (buttonType) {
            case ButtonType.ShowProgramWindow:
                if (propertyKey === 'button_text_upper') return 'Window Title';
                if (propertyKey === 'button_text_lower') return 'Selected Application';
                if (propertyKey === 'icon_path') return 'Icon (from selected app)';
                if (propertyKey === 'window_handle') return 'Window Handle (Internal)';
                break;
            case ButtonType.ShowAnyWindow:
                if (propertyKey === 'button_text_upper') return 'Window Title';
                if (propertyKey === 'button_text_lower') return 'Application Name';
                if (propertyKey === 'icon_path') return 'Icon Path';
                if (propertyKey === 'window_handle') return 'Window Handle (Internal)';
                break;
            case ButtonType.LaunchProgram:
                if (propertyKey === 'button_text_upper') return 'Selected Application to Launch';
                if (propertyKey === 'button_text_lower') return 'Display Subtext';
                if (propertyKey === 'icon_path') return 'Icon (from selected app)';
                break;
            case ButtonType.CallFunction:
                if (propertyKey === 'button_text_upper') return 'Selected Function';
                if (propertyKey === 'button_text_lower') return 'Subtext (should be empty)';
                if (propertyKey === 'icon_path') return 'Icon Path (from selected function)';
                break;
        }
        if (propertyKey === 'button_text_upper') return 'Primary Text';
        if (propertyKey === 'button_text_lower') return 'Secondary Text';
        if (propertyKey === 'icon_path') return 'Icon Path';
        return propertyKey;
    }

    function getFriendlyButtonTypeName(buttonType: ButtonType | undefined): string {
        if (buttonType === undefined) {
            return "Unknown Type";
        }
        return buttonTypeFriendlyNames[buttonType] || buttonType.toString();
    }

    function getFilteredProperties(button: Button | undefined) {
        if (!button || button.button_type === ButtonType.Disabled) {
            let relevantKeysForType: string[] = [];
            if (button && (button as Button).button_type !== ButtonType.Disabled) {
                relevantKeysForType = getDropdownFields((button as Button).button_type);
            }
            return {props: button?.properties || {}, relevantKeys: relevantKeysForType};
        }
        const relevantKeys = getDropdownFields(button.button_type);
        if (relevantKeys.length === 0) {
            return {props: {}, relevantKeys: []};
        }
        const filtered = Object.fromEntries(
            Object.entries(button.properties)
                .filter(([key]) => relevantKeys.includes(key))
                .filter(([key]) => {
                    if (button.button_type === ButtonType.CallFunction) {
                        return !['button_text_upper', 'icon_path', 'button_text_lower'].includes(key);
                    }
                    if (button.button_type === ButtonType.ShowProgramWindow) {
                        return !['button_text_lower', 'icon_path'].includes(key);
                    }
                    if (button.button_type === ButtonType.LaunchProgram) {
                        return !['button_text_upper', 'icon_path'].includes(key);
                    }
                    return true;
                })
        );
        const finalRelevantKeys = relevantKeys.filter(key => filtered.hasOwnProperty(key));
        return {props: filtered, relevantKeys: finalRelevantKeys};
    }


    // Derived state
    let currentButtonLocal = $derived(selectedButtonDetails?.button); // The actual button object
    let currentButtonTypeValue = $derived(currentButtonLocal?.button_type); // Just the type for conditional rendering

    const {props: displayableGenericProperties, relevantKeys: genericDropdownPropertyKeys} = $derived.by(() => {
        return getFilteredProperties(currentButtonLocal);
    });

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

</script>

{#if selectedButtonDetails && currentButtonLocal} <!-- Ensure currentButtonLocal is defined -->
    {@const {menuID, pageID, buttonID, slotIndex} = selectedButtonDetails}
    {@const button = currentButtonLocal}
    {@const isTrulyEmptySlot = button.button_type === ButtonType.Disabled && !getMenuConfiguration().get(menuID)?.get(pageID)?.has(buttonID)}
    {@const friendlyButtonTypeName = getFriendlyButtonTypeName(button.button_type)}

    <div class="p-4 border rounded-md shadow-sm bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-100 border-gray-200 dark:border-gray-700 w-full min-w-0">
        <div class="flex items-center justify-between mb-3">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Button Details</h2>
            <p class="text-right">Slot: {slotIndex + 1} <span
                    class="text-gray-600 dark:text-gray-400">(Page: {pageID + 1})</span></p>
        </div>
        <div class="text-sm space-y-2">
            {#if isTrulyEmptySlot && button.button_type === ButtonType.Disabled}
                <p class="text-yellow-700 dark:text-yellow-400 font-medium"><strong>Status:</strong> Empty Slot</p>
                <p class="text-gray-600 dark:text-gray-400 mb-2">Select a type below to configure this button.</p>
            {/if}

            <ButtonTypeSelector
                    currentType={currentButtonTypeValue}
                    {buttonTypeKeys}
                    {buttonTypeFriendlyNames}
                    disabled={!selectedButtonDetails}
                    onChange={handleTypeChange}
            />

            {#if currentButtonTypeValue === ButtonType.CallFunction}
                <CallFunctionConfig
                        button={button}
                        functionDefinitions={typedAvailableFunctions}
                        onUpdate={handleSpecificConfigUpdate}
                />
            {:else if currentButtonTypeValue === ButtonType.ShowProgramWindow || currentButtonTypeValue === ButtonType.LaunchProgram}
                <ProgramButtonConfig
                        button={button}
                        {installedAppsMap}
                        onUpdate={handleSpecificConfigUpdate}
                />
            {/if}

            <!-- Generic Properties Display Section -->
            {#if !(isTrulyEmptySlot && button.button_type === ButtonType.Disabled) && button.button_type !== ButtonType.Disabled}
                {#if button.properties}
                    {@const hasSpecializedUI =
                    button.button_type === ButtonType.CallFunction ||
                    button.button_type === ButtonType.ShowProgramWindow ||
                    button.button_type === ButtonType.LaunchProgram}

                    {#if (!hasSpecializedUI && (Object.keys(displayableGenericProperties).length > 0 || genericDropdownPropertyKeys.length > 0)) || (hasSpecializedUI && Object.keys(displayableGenericProperties).length > 0) }
                        <GenericPropertiesDisplay
                                displayableProperties={displayableGenericProperties}
                                buttonType={button.button_type}
                                getPropertyFriendlyNameFn={getPropertyFriendlyName}
                                dropdownPropertyKeys={genericDropdownPropertyKeys}
                                {friendlyButtonTypeName}
                        />
                    {:else if !hasSpecializedUI}
                        <p class="text-gray-600 dark:text-gray-400 mt-2">
                            {friendlyButtonTypeName} has no other specific properties to configure here.
                        </p>
                    {/if}
                {:else if button.button_type !== ButtonType.Disabled}
                    <p class="text-gray-600 dark:text-gray-400 mt-2">
                        No properties are defined for this button (type: <span
                            class="font-medium">{friendlyButtonTypeName}</span>).
                    </p>
                {/if}
            {/if}
        </div>
    </div>
{:else}
    <div class="p-4 text-center">
        <p class="text-gray-500 dark:text-gray-400">
            Select a button from a pie menu preview to see its details.
        </p>
    </div>
{/if}