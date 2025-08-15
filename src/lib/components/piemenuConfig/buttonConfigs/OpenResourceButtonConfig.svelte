<!-- Component for configuring OpenResource button type -->
<script lang="ts">
    import {type Button, ButtonType, type OpenResourceProperties} from '$lib/data/types/pieButtonTypes.ts';
    import {getDefaultButton} from "$lib/data/types/pieButtonDefaults.ts";
    import {open} from '@tauri-apps/plugin-dialog';
    import {createLogger} from "$lib/logger";
    import StandardButton from '$lib/components/StandardButton.svelte';

    const RESOURCE_PATH_KEY: keyof OpenResourceProperties = "resource_path";
    const DISPLAY_NAME_KEY: keyof OpenResourceProperties = "button_text_upper";

    // Create a logger for this component
    const logger = createLogger('OpenResourceButtonConfig');

    let {button, onUpdate} = $props<{
        button: { button_type: ButtonType.OpenResource; properties: OpenResourceProperties },
        onUpdate: (updatedButton: Button) => void
    }>();

    // Use the same default as in pieButtonDefaults
    const defaultDisplayName = getDefaultButton(ButtonType.OpenResource).properties.button_text_upper;
    const defaultResourcePath = (getDefaultButton(ButtonType.OpenResource).properties as OpenResourceProperties).resource_path;

    let displayName = $derived.by(() => {
        if (button.properties.button_text_upper === defaultDisplayName) {
            return "";
        } else {
            return button.properties.button_text_upper;
        }
    });

    let resourcePath = $derived.by(() => button.properties.resource_path || defaultResourcePath);

    function handleChange<K extends keyof OpenResourceProperties>(key: K, value: OpenResourceProperties[K]) {
        const newProperties = {...button.properties, [key]: value};
        onUpdate({...button, properties: newProperties});
    }

    async function handleBrowseForFile() {
        try {
            const selected = await open({
                multiple: false,
                directory: false,
                filters: [{
                    name: 'All Files',
                    extensions: ['*']
                }]
            });

            if (selected && !Array.isArray(selected)) {
                handleChange(RESOURCE_PATH_KEY, selected);
            }
        } catch (error) {
            logger.error('Error selecting file:', error);
        }
    }

    async function handleBrowseForFolder() {
        try {
            const selected = await open({
                multiple: false,
                directory: true
            });

            if (selected && !Array.isArray(selected)) {
                handleChange(RESOURCE_PATH_KEY, selected);
            }
        } catch (error) {
            logger.error('Error selecting folder:', error);
        }
    }
</script>

<div class="w-full min-w-0">
    <div class="mt-3 space-y-1 relative">
        <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1" for="openResourceButtonText">
            Display Name:
        </label>
        <div class="relative">
            <input
                    class="w-full pl-3 pr-10 py-2 text-base border-2  border-zinc-200 dark:border-zinc-800 focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-100 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
                    id="openResourceButtonText"
                    oninput={e => { handleChange(DISPLAY_NAME_KEY, e.currentTarget.value || defaultDisplayName); }}
                    type="text"
                    value={displayName}
                    autocomplete="off"
                    placeholder={defaultDisplayName}
            />
        </div>
    </div>

    <div class="mt-3 space-y-1 relative">
        <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1" for="resourcePath">
            Resource Path:
        </label>
        <div class="flex flex-col gap-2">
            <input
                    class="w-full pl-3 pr-10 py-2 text-base border-2  border-zinc-200 dark:border-zinc-800 focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-100 dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
                    id="resourcePath"
                    oninput={e => { handleChange(RESOURCE_PATH_KEY, e.currentTarget.value); }}
                    type="text"
                    value={resourcePath}
                    autocomplete="off"
                    placeholder="Path to resource..."
            />
            <div class="flex gap-2">
                <StandardButton
                        label="Browse Files"
                        onClick={handleBrowseForFile}
                        variant="primary"
                        style="flex: 1;"
                />
                <StandardButton
                        label="Browse Folders"
                        onClick={handleBrowseForFolder}
                        variant="primary"
                        style="flex: 1;"
                />
            </div>
        </div>
    </div>
</div>
