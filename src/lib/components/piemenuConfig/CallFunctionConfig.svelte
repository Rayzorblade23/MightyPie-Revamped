<!-- src/lib/components/piemenuConfig/CallFunctionConfig.svelte -->
<script lang="ts">
    import {type Button, ButtonType, type CallFunctionProperties} from '$lib/data/piebuttonTypes.ts';
    import {getDefaultButton} from '$lib/data/pieButtonDefaults.ts';
    import FunctionSelector from './FunctionSelector.svelte';

    // Data for function definitions
    interface FunctionDefinition {
        icon_path: string;
        description?: string;
    }

    type AvailableFunctionsMap = Record<string, FunctionDefinition>;

    let {
        button, // The current CallFunction button object
        functionDefinitions,
        onUpdate // Callback to update the button configuration
    } = $props<{
        button: { button_type: ButtonType.CallFunction; properties: CallFunctionProperties };
        functionDefinitions: AvailableFunctionsMap;
        onUpdate: (updatedButton: Button) => void;
    }>();

    // Derived from the button's properties
    let selectedFunctionName = $derived(button.properties.button_text_upper || '');

    function handleFunctionSelect(selectedKey: string) {
        const functionDefinition = functionDefinitions[selectedKey];

        let updatedProps: CallFunctionProperties;
        // Use existing properties as a base to preserve any other potential CallFunction props
        const baseProps = button.properties ||
            (getDefaultButton(ButtonType.CallFunction) as { properties: CallFunctionProperties }).properties;

        if (!selectedKey || !functionDefinition) { // Unselected or invalid
            updatedProps = {
                ...baseProps,
                button_text_upper: "",
                button_text_lower: "", // Ensured empty
                icon_path: ""
            };
        } else { // Valid function selected
            updatedProps = {
                ...baseProps,
                button_text_upper: selectedKey,
                button_text_lower: "", // Ensured empty
                icon_path: functionDefinition.icon_path || ""
            };
        }
        onUpdate({button_type: ButtonType.CallFunction, properties: updatedProps});
    }
</script>

<FunctionSelector
        {selectedFunctionName}
        {functionDefinitions}
        onSelect={handleFunctionSelect}
/>