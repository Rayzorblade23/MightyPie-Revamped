// src/lib/env.ts

import {invoke} from "@tauri-apps/api/core";

export async function getPrivateEnvVar(key: string): Promise<string> {
    try {
        return await invoke('get_private_env_var', {key});
    } catch (error) {
        console.error(`Failed to fetch env var ${key}:`, error);
        throw error;
    }
} 
