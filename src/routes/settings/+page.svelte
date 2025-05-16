<script lang="ts">
    function handleButtonClick(buttonId: number) {
        console.log(`Button ${buttonId} clicked`);
        // Add your button-specific logic here
    }
</script>

<div class="container">
    <h1>Settings</h1>
    <div class="button-grid">
        {#each Array(8) as _, i}
            <button on:click={() => handleButtonClick(i + 1)}>Button {i + 1}</button>
        {/each}
    </div>
</div>

<style>
    :global(body) {
        background-color: #333; /* Solid dark grey background */
        margin: 0;
        font-family: Arial, sans-serif;
    }

    .container {
        min-height: 100vh; /* Ensure it covers the full viewport */
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
    }

    h1 {
        text-align: center;
        margin-bottom: 20px;
    }

    .button-grid {
        display: flex;
        gap: 10px;
        justify-content: center;
        align-items: center;
        padding: 20px;
    }

    button {
        padding: 10px 20px;
        font-size: 16px;
        cursor: pointer;
        border: 1px solid #ccc;
        border-radius: 5px;
        background-color: #f0f0f0;
        color: #333; /* Darker text color for better contrast */
        transition: background-color 0.3s;
    }

    button:hover {
        background-color: #ddd;
    }
</style>

