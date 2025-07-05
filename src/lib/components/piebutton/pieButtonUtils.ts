import {ButtonType} from '$lib/data/piebuttonTypes.ts';

// Border class map shared by both components
const borderClassMap: Record<ButtonType | 'default', string> = {
    [ButtonType.ShowAnyWindow]: 'border-[var(--color-accent-anywin)]',
    [ButtonType.ShowProgramWindow]: 'border-[var(--color-accent-programwin)]',
    [ButtonType.LaunchProgram]: 'border-[var(--color-accent-launch)]',
    [ButtonType.CallFunction]: 'border-[var(--color-accent-function)]',
    [ButtonType.Disabled]: 'border-zinc-500 dark:border-grey-600',
    default: 'border-zinc-500 dark:border-grey-600',
};

// Returns the base button classes (can be extended if needed)
const baseButtonClasses =
    'flex items-center p-0.5 min-w-0 border-solid border rounded-lg';

// Returns the final button classes based on state
export function composePieButtonClasses({
                                            isDisabled,
                                            taskType,
                                            extraClasses = '',
                                            allowSelectWhenDisabled = false,
                                        }: {
    isDisabled: boolean;
    taskType: ButtonType | 'default';
    extraClasses?: string;
    allowSelectWhenDisabled?: boolean;
}) {
    let staticBaseClasses: string;
    if (isDisabled) {
        staticBaseClasses = [
            baseButtonClasses,
            'bg-zinc-200 text-zinc-400',
            'dark:bg-zinc-800 dark:text-zinc-500',
            allowSelectWhenDisabled ? '' : 'select-none pointer-events-none',
        ].join(' ').trim();
    } else {
        staticBaseClasses = [
            baseButtonClasses,
            'bg-white text-zinc-900',
            'dark:bg-zinc-800 dark:text-white',
        ].join(' ');
    }
    const borderClass = borderClassMap[taskType ?? 'default'];
    return `${staticBaseClasses} ${borderClass} ${extraClasses}`.trim();
}

// Fetches SVG as string, returns a Promise<string> (with error SVG fallback)
export async function fetchSvgIcon(iconPath?: string): Promise<string> {
    const errorSvg = `<svg class=\"h-full w-full\" viewBox=\"0 0 24 24\"><path fill=\"red\" d=\"M12 2 L2 22 L22 22 Z M11 10 L13 10 L13 16 L11 16 Z M11 18 L13 18 L13 20 L11 20 Z\"></path></svg>`;
    if (!iconPath || !iconPath.endsWith('.svg')) return errorSvg;
    try {
        const r = await fetch(iconPath);
        if (!r.ok) return errorSvg;
        const text = await r.text();
        return text.replace(/<svg /, '<svg class="h-full w-full" ');
    } catch (err) {
        return errorSvg;
    }
}
