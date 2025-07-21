<!-- src/lib/components/piemenuConfig/ProgramButtonConfig.svelte -->
<script lang="ts">
    import {
        type Button,
        ButtonType,
        type LaunchProgramProperties,
        type ShowProgramWindowProperties
    } from '$lib/data/types/pieButtonTypes.ts';
    import type {InstalledAppsMap} from '$lib/data/installedAppsInfoManager.svelte.ts';
    import ApplicationSelector from '../selectors/ApplicationSelector.svelte';

    type ProgramButton =
        | { button_type: ButtonType.ShowProgramWindow; properties: ShowProgramWindowProperties }
        | { button_type: ButtonType.LaunchProgram; properties: LaunchProgramProperties };

    let {
        button, // The current ShowProgramWindow or LaunchProgram button
        installedAppsMap,
        onUpdate
    } = $props<{
        button: ProgramButton;
        installedAppsMap: InstalledAppsMap;
        onUpdate: (updatedButton: Button) => void;
    }>();

    let selectedAppName = $derived.by(() => {
        if (button.button_type === ButtonType.ShowProgramWindow) {
            return button.properties.button_text_lower || '';
        }
        // Must be LaunchProgram
        return button.properties.button_text_upper || '';
    });

    function handleAppSelect(selectedAppNameKey: string) {
        const appInfo = selectedAppNameKey ? installedAppsMap.get(selectedAppNameKey) : undefined;
        let newButton: Button = {...button}; // Start with a copy

        if (button.button_type === ButtonType.ShowProgramWindow) {
            const currentProps = button.properties; // Already known to be ShowProgramWindowProperties
            let newProps: ShowProgramWindowProperties = {
                ...currentProps,
                button_text_lower: selectedAppNameKey, // Only update button_text_lower
                icon_path: appInfo?.iconPath || "",
            };
            newButton = {button_type: ButtonType.ShowProgramWindow, properties: newProps};
        } else { // LaunchProgram
            const currentProps = button.properties; // Already known to be LaunchProgramProperties
            newButton = {
                button_type: ButtonType.LaunchProgram,
                properties: {
                    ...currentProps,
                    button_text_upper: selectedAppNameKey,
                    icon_path: appInfo?.iconPath || ""
                }
            };
        }
        onUpdate(newButton);
    }
</script>

<div class="w-full min-w-0">
    <ApplicationSelector
            {selectedAppName}
            {installedAppsMap}
            onSelect={handleAppSelect}
    />
</div>