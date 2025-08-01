use std::collections::HashMap;
use std::env;
use std::sync::OnceLock;

// Store baked-in environment variables in a static HashMap
static BAKED_ENV_VARS: OnceLock<HashMap<&'static str, &'static str>> = OnceLock::new();

// Initialize the baked-in environment variables
fn get_baked_env_vars() -> &'static HashMap<&'static str, &'static str> {
    BAKED_ENV_VARS.get_or_init(|| {
        let mut vars = HashMap::new();

        // Add all environment variables that were baked in at build time
        // The env! macro will cause a compile error if the variable doesn't exist
        // We use option_env! instead which returns None if the variable doesn't exist
        if let Some(val) = option_env!("NATS_SERVER_URL") {
            vars.insert("NATS_SERVER_URL", val);
        }
        if let Some(val) = option_env!("NATS_AUTH_TOKEN") {
            vars.insert("NATS_AUTH_TOKEN", val);
        }
        // Add other critical environment variables as needed

        vars
    })
}

#[tauri::command]
pub fn get_private_env_var(key: String) -> Result<String, String> {
    // First check if we have a baked-in value from build time
    let baked_vars = get_baked_env_vars();
    if let Some(value) = baked_vars.get(key.as_str()) {
        return Ok((*value).to_string());
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
