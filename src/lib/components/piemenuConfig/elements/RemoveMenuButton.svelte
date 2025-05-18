<script lang="ts">
    export let onClick: () => void;
    export let disabled: boolean = false;
</script>

<button
        class="bg-red-500 hover:bg-red-600 text-white font-semibold py-2 px-4 rounded shadow transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-red-400 focus:ring-opacity-75 disabled:opacity-50 disabled:cursor-not-allowed"
        disabled={disabled}
        on:click={onClick}
        type="button"
>
    − Remove Menu
</button>
