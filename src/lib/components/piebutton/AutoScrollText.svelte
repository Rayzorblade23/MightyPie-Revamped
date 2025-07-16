<script lang="ts">
    const {
        text = '',
        enabled = false,
        mode = 'normal', // 'normal' (pause at start/end) or 'hover' (pause only at end)
        className = '',
        style = ''
    } = $props<{
        text?: string;
        enabled?: boolean;
        mode?: 'normal' | 'hover';
        className?: string;
        style?: string;
    }>();

    let textEl = $state<HTMLSpanElement | null>(null);
    let containerEl = $state<HTMLSpanElement | null>(null);
    let textKey = $state(0);

    $effect(() => {
        void text;
        void enabled;
        setTimeout(() => textKey = (textKey + 1) % 1000, 0);
    });

    let shouldScroll = $derived.by(() => {
        if (!enabled) return false;
        if (!textEl || !containerEl) return false;
        void textKey;
        return textEl.scrollWidth > containerEl.offsetWidth;
    });

    let scrollDistance = $derived.by(() => {
        if (!enabled || !textEl || !containerEl) return 0;
        void textKey;
        const textWidth = textEl.scrollWidth;
        const containerWidth = containerEl.offsetWidth;
        if (textWidth <= containerWidth) return 0;
        return Math.round(textWidth - containerWidth);
    });

    const pxPerSecond = 30;
    const pauseDuration = 2; // seconds, fixed
    let dynamicKeyframes = $state('');
    let animationName = $state('');

    let scrollDuration = $derived.by(() => {
        const distance = scrollDistance;
        if (distance <= 0) return pauseDuration * 2;
        const scrollTime = distance / pxPerSecond;
        return mode === 'hover'
            ? scrollTime + pauseDuration
            : scrollTime + 2 * pauseDuration;
    });

    $effect(() => {
        if (!shouldScroll) {
            dynamicKeyframes = '';
            animationName = '';
            return;
        }
        const distance = scrollDistance;
        const scrollTime = distance / pxPerSecond;
        let totalDuration, startPausePct, endScrollPct, animName, kf;
        if (mode === 'hover') {
            totalDuration = scrollTime + pauseDuration;
            startPausePct = 0;
            endScrollPct = (scrollTime / totalDuration) * 100;
            animName = `scroll_horizontal_hover_${distance}_${Math.round(totalDuration * 100)}`;
            kf = `@keyframes ${animName} {\n` +
                `  0% { margin-left: 0; }\n` +
                `  ${endScrollPct}% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }\n` +
                `  100% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }\n` +
                `}`;
        } else {
            totalDuration = scrollTime + 2 * pauseDuration;
            startPausePct = (pauseDuration / totalDuration) * 100;
            endScrollPct = ((pauseDuration + scrollTime) / totalDuration) * 100;
            animName = `scroll_horizontal_${distance}_${Math.round(totalDuration * 100)}`;
            kf = `@keyframes ${animName} {\n` +
                `  0% { margin-left: 0; }\n` +
                `  ${startPausePct}% { margin-left: 0; }\n` +
                `  ${endScrollPct}% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }\n` +
                `  100% { margin-left: calc(-1 * var(--scroll-distance, 0px)); }\n` +
                `}`;
        }
        animationName = animName;
        dynamicKeyframes = kf;
    });
</script>

<svelte:head>
    {#if dynamicKeyframes}
        {@html `<style>${dynamicKeyframes}</style>`}
    {/if}
</svelte:head>

<span bind:this={containerEl} class="scroll-clip {className}" style={style}>
    {#if enabled && shouldScroll}
        {#key textKey}
            <span class="scrolling-text"
                  bind:this={textEl}
                  style="font-size: inherit; --scroll-distance: {scrollDistance}px; --scroll-duration: {scrollDuration}s; animation-name: {animationName}; animation-duration: {scrollDuration}s; animation-timing-function: linear; animation-iteration-count: infinite;">
                {text}
            </span>
        {/key}
    {:else}
        <span class="truncate-text" bind:this={textEl} style="font-size: inherit;">{text}</span>
    {/if}
</span>

<style>
    .scroll-clip {
        display: flex;
        align-items: center;
        min-width: 0;
        overflow: hidden;
        vertical-align: middle;
        font-size: inherit;
        white-space: nowrap;
    }

    .scrolling-text {
        font-size: inherit;
    }

    .truncate-text {
        text-overflow: ellipsis;
        overflow: hidden;
        white-space: nowrap;
    }
</style>
