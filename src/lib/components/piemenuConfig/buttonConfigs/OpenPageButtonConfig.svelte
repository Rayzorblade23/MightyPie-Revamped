<!-- Modern selector-based UI for OpenSpecificPieMenuPage, styled to match FunctionSelector/ProgramButtonConfig -->
<script lang="ts">
    import {type Button, ButtonType, type OpenSpecificPieMenuPageProperties} from '$lib/data/types/pieButtonTypes.ts';
    import {getDefaultButton} from "$lib/data/types/pieButtonDefaults.ts";

    const PAGE_ID_KEY: keyof OpenSpecificPieMenuPageProperties = "page_id";
    const MENU_ID_KEY: keyof OpenSpecificPieMenuPageProperties = "menu_id";
    const DISPLAY_NAME_KEY: keyof OpenSpecificPieMenuPageProperties = "button_text_upper";

    let {button, onUpdate, menuConfig} = $props<{
        button: { button_type: ButtonType.OpenSpecificPieMenuPage; properties: OpenSpecificPieMenuPageProperties },
        onUpdate: (updatedButton: Button) => void,
        menuConfig: any
    }>();

    // Use the same default as in pieButtonDefaults
    const defaultDisplayName = getDefaultButton(ButtonType.OpenSpecificPieMenuPage).properties.button_text_upper;

    let displayName = $derived.by(() => {
        if (button.properties.button_text_upper == defaultDisplayName) {
            return "";
        } else {
            return button.properties.button_text_upper;
        }
    });

    let selectedMenuId = $state(button.properties.menu_id);
    let selectedPageId = $state(button.properties.page_id);

    function getAvailableMenuIndexes(menuConfiguration: any): number[] {
        if (!menuConfiguration || typeof menuConfiguration.keys !== 'function') return [];
        return Array.from(menuConfiguration.keys()).map(key => Number(key)).filter(key => !isNaN(key)).sort((a: number, b: number) => a - b);
    }

    function getAvailablePageIndexes(menuConfiguration: any, menuId: number): number[] {
        if (!menuConfiguration || typeof menuConfiguration.get !== 'function') return [];
        const pagesMap = menuConfiguration.get(menuId);
        if (!pagesMap || typeof pagesMap.keys !== 'function') return [];
        return Array.from(pagesMap.keys()).map(key => Number(key)).filter(key => !isNaN(key)).sort((a: number, b: number) => a - b);
    }

    const availableMenus = $derived(() => getAvailableMenuIndexes(menuConfig));
    const availablePages = $derived(() => getAvailablePageIndexes(menuConfig, selectedMenuId));

    $effect(() => {
        const menus = availableMenus();
        if (menus.length && !menus.includes(selectedMenuId)) {
            selectedMenuId = menus[0];
            handleChange(MENU_ID_KEY, selectedMenuId);
        }
    });
    $effect(() => {
        const pages = availablePages();
        if (pages.length && !pages.includes(selectedPageId)) {
            selectedPageId = pages[0];
            handleChange(PAGE_ID_KEY, selectedPageId);
        }
    });

    function handleChange<K extends keyof OpenSpecificPieMenuPageProperties>(key: K, value: OpenSpecificPieMenuPageProperties[K]) {
        const newProperties = {...button.properties, [key]: value};
        onUpdate({...button, properties: newProperties});
    }
</script>

<div class="w-full min-w-0">
    <div class="mt-3 flex flex-row gap-4">
        <div class="flex-1 space-y-1">
            <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1" for="openPageMenuId">
                Menu:
            </label>
            <select
                    class="block w-full pl-3 pr-10 py-2 text-base border-none focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-700 transition-colors"
                    id="openPageMenuId"
                    oninput={e => { selectedMenuId = +e.currentTarget.value; handleChange(MENU_ID_KEY, selectedMenuId); }}
                    value={selectedMenuId}
            >
                {#each availableMenus() as menuId}
                    <option value={menuId}
                            class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">{menuId + 1}</option>
                {/each}
            </select>
        </div>
        <div class="flex-1 space-y-1">
            <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1" for="openPagePageId">
                Page:
            </label>
            <select
                    class="block w-full pl-3 pr-10 py-2 text-base border-none focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-300 dark:hover:bg-zinc-700 transition-colors"
                    id="openPagePageId"
                    oninput={e => { selectedPageId = +e.currentTarget.value; handleChange(PAGE_ID_KEY, selectedPageId); }}
                    value={selectedPageId}
            >
                {#each availablePages() as pageId}
                    <option value={pageId}
                            class="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100">{pageId + 1}</option>
                {/each}
            </select>
        </div>
    </div>
    <div class="mt-3 space-y-1 relative">
        <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-400 mb-1" for="openPageButtonText">
            Display Name:
        </label>
        <div class="relative">
            <input
                    class="w-full pl-3 pr-10 py-2 text-base border-none focus:outline-none focus:ring-2 focus:ring-amber-400 sm:text-sm rounded-lg shadow-sm bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
                    id="openPageButtonText"
                    oninput={e => { displayName = e.currentTarget.value; handleChange(DISPLAY_NAME_KEY, displayName); }}
                    type="text"
                    value={displayName}
                    autocomplete="off"
                    placeholder={defaultDisplayName}
            />
        </div>
    </div>
</div>