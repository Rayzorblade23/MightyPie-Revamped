/**
 * Shared button styles for consistent styling across the application
 * Used by StandardButton, ExpandedButton, and other button components
 */

export type ButtonVariant = 'primary' | 'warning' | 'special';

// Button variant classes
export const getVariantClass = (variant: ButtonVariant): string => {
    switch (variant) {
        case 'primary':
            return "bg-purple-800 dark:bg-purple-950 text-zinc-100 hover:bg-violet-800 dark:hover:bg-violet-950 active:bg-purple-700 dark:active:bg-indigo-950";
        case 'warning':
            return "bg-rose-500 dark:bg-rose-700 text-white hover:bg-rose-600 dark:hover:bg-rose-800";
        case 'special':
            return "bg-amber-500 text-white hover:bg-amber-600";
        default:
            return "bg-purple-800 dark:bg-purple-950 text-zinc-100 hover:bg-violet-800 dark:hover:bg-violet-950 active:bg-purple-700 dark:active:bg-indigo-950";
    }
};

// Focus styles for buttons
export const focusStyles = "focus:outline focus:outline-amber-500 focus:border-amber-500 focus:border-2";

// Common button styles
export const commonButtonStyles = "px-4 py-2 rounded-lg border border-none text-base transition cursor-pointer shadow-md";

// Disabled button styles
export const disabledStyles = "disabled:opacity-60 disabled:text-zinc-400 disabled:dark:text-zinc-500";
