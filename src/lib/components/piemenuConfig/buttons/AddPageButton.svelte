<!-- src/lib/components/piemenuConfig/AddPageButton.svelte -->
<script lang="ts">
    let {onClick} = $props<{
        onClick: () => void;
    }>();
</script>

<div class="flex-shrink-0 flex items-center justify-center p-1 ml-2 self-center">
    <button
            title="Add a new page to this menu"
            aria-label="Add a new page to the current menu"
            class="p-3 bg-green-500 hover:bg-green-600 text-white rounded-full shadow-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-opacity-50 transition-colors"
            onclick={onClick}
            data-plus-button
    >
        <!-- Heroicon name: solid/plus -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-6 h-6">
            <path fill-rule="evenodd"
                  d="M12 3.75a.75.75 0 0 1 .75.75v6.75h6.75a.75.75 0 0 1 0 1.5h-6.75v6.75a.75.75 0 0 1-1.5 0v-6.75H4.5a.75.75 0 0 1 0-1.5h6.75V4.5a.75.75 0 0 1 .75-.75Z"
                  clip-rule="evenodd"/>
        </svg>
    </button>
</div>