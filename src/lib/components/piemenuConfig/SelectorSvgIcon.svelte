<!-- src/lib/components/piemenuConfig/SvgIcon.svelte -->
<script lang="ts">
    let {
        iconPath,
        svgClasses = "h-5 w-5", // Default classes
        alt = "icon",
        titleText
    } = $props<{
        iconPath: string | undefined;
        svgClasses?: string;
        alt?: string;
        titleText?: string;
    }>();

    let svgPromise = $state<Promise<string> | undefined>(undefined);

    $effect(() => {
        if (iconPath && typeof iconPath === 'string' && iconPath.trim() !== '' && iconPath.endsWith('.svg')) {
            svgPromise = fetch(iconPath)
                .then(r => {
                    if (!r.ok) throw new Error(`SVG Fetch Error: ${r.status} from ${iconPath}`);
                    return r.text();
                })
                .then(text => {
                    // A slightly more robust way to add classes
                    const parser = new DOMParser();
                    const svgDoc = parser.parseFromString(text, "image/svg+xml");
                    const svgElement = svgDoc.documentElement;
                    if (svgElement && svgElement.nodeName === 'svg') {
                        svgClasses.split(' ').forEach(cls => {
                            if (cls) svgElement.classList.add(cls);
                        });
                        return svgElement.outerHTML;
                    }
                    // Fallback if parsing fails or not an SVG root
                    return text.includes('class="') ? text.replace(/class="([^"]*)"/, `class="$1 ${svgClasses}"`) : text.replace(/<svg /, `<svg class="${svgClasses}" `);
                })
                .catch(err => {
                    console.error(`SVG Error for ${iconPath}:`, err);
                    return `<svg class="${svgClasses}" viewBox="0 0 24 24" fill="currentColor" color="red"><path d="M12 2L2 22L22 22Z M11 10 L13 10 L13 16 L11 16 Z M11 18 L13 18 L13 20 L11 20 Z"></path></svg>`;
                });
        } else {
            svgPromise = undefined;
        }
    });
</script>

{#if iconPath && iconPath.trim() !== ""}
    {#if iconPath.endsWith('.svg')}
        {#if svgPromise}
            {#await svgPromise}
                <div class="{svgClasses} animate-pulse bg-zinc-300 rounded"></div>
            {:then svgContent}
                {@html svgContent}
            {:catch error}
                <div class="{svgClasses} text-red-500 flex items-center justify-center"
                     title={error.message || 'Error loading SVG'}>⚠️
                </div>
            {/await}
        {/if}
        <!-- No specific 'else' here for SVG promise; if promise is undefined, nothing from this block renders -->
    {:else}
        <!-- Fallback for non-SVG images -->
        <img src={iconPath} {alt} class={svgClasses} title={titleText || iconPath}/>
    {/if}
{:else}
    <!-- Placeholder for when iconPath is empty or undefined -->
    <span class="text-zinc-400 {svgClasses} flex items-center justify-center">...</span>
{/if}