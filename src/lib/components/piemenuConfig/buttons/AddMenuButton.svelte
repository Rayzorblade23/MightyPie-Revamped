<script lang="ts">
    export let onClick: () => void;
</script>

<button
        class="bg-green-500 hover:bg-green-600 text-white font-semibold py-2 px-4 rounded shadow transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-green-400 focus:ring-opacity-75"
        on:click={onClick}
        type="button"
>
    + Add Menu
</button>
