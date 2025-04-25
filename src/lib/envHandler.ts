import {invoke} from '@tauri-apps/api/core';

// Cache for environment variables
let envVars: Record<string, string> | null = null;

/**
 * Initialize environment variables from Rust backend.
 * Should be called once before using getEnvVar.
 */
export async function initializeEnvVars(): Promise<void> {
    if (envVars) return;

    try {
        envVars = await invoke<Record<string, string>>('get_all_public_env_vars');
        console.log('Environment variables initialized');
    } catch (error) {
        console.error('Failed to initialize env vars:', error);
        throw error;
    }
}

/**
 * Get environment variable by key.
 * @throws Error if variables not initialized or key not found
 */
export function getEnvVar(key: string): string {
    if (!envVars) {
        throw new Error('Call initializeEnvVars() first');
    }

    const value = envVars[key];
    if (!value) {
        throw new Error(`Environment variable "${key}" not found`);
    }

    return value;
}