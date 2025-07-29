use std::env;

#[tauri::command]
pub fn get_private_env_var(key: String) -> Result<String, String> {
    // Simply access the environment variable - it should already be loaded at startup
    match env::var(&key) {
        Ok(value) => Ok(value),
        Err(_) => Err(format!("Environment variable '{}' not found", key)),
    }
}

// Helper to determine if we're in debug/dev mode
pub fn is_debug() -> bool {
    cfg!(debug_assertions)
}

// Set an environment variable that will be inherited by child processes
pub fn set_env_var(key: &str, value: &str) {
    env::set_var(key, value);
}
