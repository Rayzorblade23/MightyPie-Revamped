<script lang="ts">
    /**
     * ExpandedButton component - A reusable button with consistent styling
     * Based on StandardButton but with support for bind:this and focus management
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

    // Export the button element for external access
    let buttonElement: HTMLButtonElement;

    // Method to focus the button
    export function focus() {
        buttonElement?.focus();
    }
</script>

<button
        bind:this={buttonElement}
        aria-label={finalAriaLabel}
        class="{commonButtonStyles} {variantClass} {bold ? 'font-semibold' : ''} {focusStyles} {disabled ? disabledStyles : ''}"
        onclick={onClick}
        {disabled}
        {type}
        style={style}
        tabindex="0"
>
    {label}
</button>
