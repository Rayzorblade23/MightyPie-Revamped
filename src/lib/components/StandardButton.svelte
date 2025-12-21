<script lang="ts">
    /**
     * StandardButton component - A reusable button with consistent styling
     * Used throughout the application for primary actions
     */
    import {
        type ButtonVariant,
        commonButtonStyles,
        disabledStyles,
        focusStyles,
        getVariantClass
    } from '$lib/styles/buttonStyles';

    // Props
    const {
        label = "",
        ariaLabel = "",
        onClick = () => {
        },
        type = "button",
        disabled = false,
        style = "",
        bold = false,
        variant = "primary",
        iconSrc = undefined,
        iconImgClasses = "w-5 h-5",
        iconSlotClasses = "w-5 h-5",
        tooltipText = "",
    } = $props<{
        label?: string;
        ariaLabel?: string;
        onClick?: () => void;
        type?: "button" | "submit" | "reset";
        disabled?: boolean;
        style?: string;
        bold?: boolean;
        variant?: ButtonVariant;
        iconSrc?: string;
        iconImgClasses?: string;
        iconSlotClasses?: string;
        tooltipText?: string;
    }>();

    // If ariaLabel is not provided, use label
    const finalAriaLabel = $derived(ariaLabel || label);

    const showLabelText = $derived(label.trim().length > 0);

    // Get the variant class from shared styles
    const variantClass = $derived(getVariantClass(variant));

    let tooltipVisible = $state(false);
    let tooltipTimer: ReturnType<typeof setTimeout> | undefined;

    function handleMouseEnter() {
        if (disabled) return;
        if (!tooltipText) return;
        if (tooltipTimer) clearTimeout(tooltipTimer);
        tooltipTimer = setTimeout(() => {
            tooltipVisible = true;
        }, 1500);
    }

    function handleMouseLeave() {
        if (tooltipTimer) clearTimeout(tooltipTimer);
        tooltipTimer = undefined;
        tooltipVisible = false;
    }
</script>

<button
        aria-label={finalAriaLabel}
        class="{commonButtonStyles} {variantClass} {bold ? 'font-semibold' : ''} {focusStyles} {disabled ? disabledStyles : ''} relative"
        onclick={onClick}
        onmouseenter={handleMouseEnter}
        onmouseleave={handleMouseLeave}
        {disabled}
        {type}
        style={style}
>
    {#if iconSrc}
        <span class="flex items-center gap-2 leading-none">
            <span class="relative flex-shrink-0 overflow-visible {iconSlotClasses}">
                <img
                        alt={finalAriaLabel || 'icon'}
                        src={iconSrc}
                        class="block absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 max-w-none max-h-none {iconImgClasses}"
                />
            </span>
            {#if showLabelText}
                <span class="leading-none">{label}</span>
            {/if}
        </span>
    {:else}
        {label}
    {/if}

    {#if tooltipText && tooltipVisible}
        <span class="pointer-events-none absolute left-1/2 -translate-x-1/2 bottom-full mb-2 whitespace-nowrap rounded-md bg-black/80 px-2 py-1 text-xs text-white shadow-lg">
            {tooltipText}
        </span>
    {/if}
</button>
