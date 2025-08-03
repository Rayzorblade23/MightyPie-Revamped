use std::collections::HashMap;
use std::env;
use std::sync::OnceLock;
use serde_json;
use std::path::{Path};

// Store baked-in environment variables in a static HashMap
static BAKED_ENV_VARS: OnceLock<HashMap<String, String>> = OnceLock::new();

// Initialize the baked-in environment variables
fn get_baked_env_vars() -> &'static HashMap<String, String> {
    BAKED_ENV_VARS.get_or_init(|| {
        let mut vars = HashMap::new();

        // Load from the JSON blob that was baked in at build time
        if let Some(json_str) = option_env!("BAKED_ENV_JSON") {
            if let Ok(json_vars) = serde_json::from_str::<HashMap<String, String>>(json_str) {
                vars.extend(json_vars);
            }
        }

        vars
    })
}

#[tauri::command]
pub fn get_private_env_var(key: String) -> Result<String, String> {
    // First check if we have a baked-in value from build time
    let baked_vars = get_baked_env_vars();
    if let Some(value) = baked_vars.get(&key) {
        return Ok(value.clone());
    }

    // If not baked in, try to get from runtime environment
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


// Command to get the app data directory path
#[tauri::command]
pub fn get_app_data_dir() -> String {
    let app_name = env::var("PUBLIC_APPNAME").unwrap_or_else(|_| "MightyPieRevamped".to_string());
    let local_app_data = env::var("LOCALAPPDATA")
        .unwrap_or_else(|_| env::var("APPDATA").unwrap_or_else(|_| ".".to_string()));

    Path::new(&local_app_data).join(app_name).to_string_lossy().to_string()
}
