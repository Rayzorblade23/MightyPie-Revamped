// Tauri doesn't have a Node.js server to do proper SSR
// so we will use adapter-static to prerender the app (SSG)
// See: https://v2.tauri.app/start/frontend/sveltekit/ for more info
export const prerender = true;
export const ssr = false;

import { setupLogging } from '$lib/logger';

// Browser-only code
if (typeof window !== 'undefined') {
    // Initialize logging as soon as possible in the browser context
    setupLogging();
}