<script lang="ts">
    /**
     * StandardButton component - A reusable button with consistent styling
     * Used throughout the application for primary actions
     */
    import { getVariantClass, focusStyles, commonButtonStyles, disabledStyles, type ButtonVariant } from '$lib/styles/buttonStyles';

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
            variant = "primary"
        } = $props<{
            label?: string;
            ariaLabel?: string;
            onClick?: () => void;
            type?: "button" | "submit" | "reset";
            disabled?: boolean;
            style?: string;
            bold?: boolean;
            variant?: ButtonVariant;
        }>();

    // If ariaLabel is not provided, use label
    const finalAriaLabel = $derived(ariaLabel || label);

    // Get the variant class from shared styles
    const variantClass = $derived(getVariantClass(variant));
</script>

<button
        aria-label={finalAriaLabel}
        class="{commonButtonStyles} {variantClass} {bold ? 'font-semibold' : ''} {focusStyles} {disabled ? disabledStyles : ''}"
        onclick={onClick}
        {disabled}
        {type}
        style={style}
>
    {label}
</button>
