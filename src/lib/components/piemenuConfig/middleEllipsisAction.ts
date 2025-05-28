// Svelte action for dynamic, pixel-perfect middle ellipsis
// Usage: <span use:middleEllipsis={fullText} ...>

export function middleEllipsis(node: HTMLElement, text: string) {
    let observer: ResizeObserver | null = null;
    let originalText = text;
    let font = getFont(node);

    function getFont(el: HTMLElement): string {
        const style = getComputedStyle(el);
        return `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    }

    function measure(text: string): number {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        if (!ctx) return 0;
        ctx.font = font;
        return ctx.measureText(text).width;
    }

    function update() {
        if (!originalText) {
            node.textContent = '';
            return;
        }
        font = getFont(node);
        const available = node.offsetWidth;
        if (measure(originalText) <= available) {
            node.textContent = originalText;
            node.title = originalText;
            return;
        }
        // Binary search for best fit
        let left = 0;
        let right = originalText.length;
        let best = originalText;
        while (left < right) {
            const mid = Math.floor((left + right) / 2);
            const keep = Math.max(2, Math.floor((mid - 3) / 2));
            const truncated = originalText.slice(0, keep) + '...' + originalText.slice(-keep);
            if (measure(truncated) > available) {
                right = mid;
            } else {
                best = truncated;
                left = mid + 1;
            }
        }
        node.textContent = best;
        node.title = originalText;
    }

    update();
    observer = new ResizeObserver(update);
    observer.observe(node);

    return {
        update(newText: string) {
            originalText = newText;
            update();
        },
        destroy() {
            if (observer) observer.disconnect();
        }
    };
}
