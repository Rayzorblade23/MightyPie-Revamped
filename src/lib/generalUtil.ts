import {invoke} from "@tauri-apps/api/core";
import {createLogger} from "$lib/logger";

// Create a logger for this module
const logger = createLogger('GeneralUtil');

export function convertRemToPixels(rem: number) {
    return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}

export function horizontalScroll(node: HTMLElement) {
    let position = node.scrollLeft;
    let velocity = 0;
    let ticking = false;
    let ignoreMomentum = false;

    function animate() {
        if (ignoreMomentum) {
            velocity = 0;
            ticking = false;
            return;
        }
        if (Math.abs(velocity) < 0.1) {
            velocity = 0;
            ticking = false;
            return;
        }
        position += velocity;
        velocity *= 0.9;
        const maxScroll = node.scrollWidth - node.clientWidth;
        position = Math.max(0, Math.min(maxScroll, position));
        node.scrollLeft = position;
        requestAnimationFrame(animate);
    }

    function handleWheel(event: WheelEvent) {
        if (event.deltaY === 0) return;
        if (node.scrollWidth <= node.clientWidth) return;
        event.preventDefault();
        velocity += (event.deltaY + event.deltaX) * 0.2;
        position = node.scrollLeft;
        if (!ticking) {
            ticking = true;
            animate();
        }
    }

    // Expose a method to temporarily disable momentum
    (node as any).lockMomentum = function lockMomentum() {
        ignoreMomentum = true;
        velocity = 0;
        ticking = false;
        setTimeout(() => {
            ignoreMomentum = false;
        }, 100); // 100ms is enough for scrollIntoView to finish
    };

    node.addEventListener('wheel', handleWheel, {passive: false});

    return {
        destroy() {
            node.removeEventListener('wheel', handleWheel);
        }
    };
}

export async function getPrivateEnvVar(key: string): Promise<string> {
    try {
        return await invoke('get_private_env_var', {key});
    } catch (error) {
        logger.error(`Failed to fetch env var ${key}:`, error);
        throw error;
    }
}