use log::{debug, error};
use tauri_plugin_autostart::ManagerExt;
use tauri::AppHandle;

/// Enable autostart for the application
#[tauri::command]
pub fn enable_autostart(app_handle: AppHandle) -> Result<(), String> {
    debug!("Backend: Enabling autostart");
    match app_handle.autolaunch().enable() {
        Ok(_) => {
            debug!("Backend: Autostart enabled successfully");
            Ok(())
        },
        Err(e) => {
            error!("Backend: Failed to enable autostart: {:?}", e);
            Err(format!("Failed to enable autostart: {:?}", e))
        }
    }
}

/// Disable autostart for the application
#[tauri::command]
pub fn disable_autostart(app_handle: AppHandle) -> Result<(), String> {
    debug!("Backend: Disabling autostart");
    match app_handle.autolaunch().disable() {
        Ok(_) => {
            debug!("Backend: Autostart disabled successfully");
            Ok(())
        },
        Err(e) => {
            error!("Backend: Failed to disable autostart: {:?}", e);
            Err(format!("Failed to disable autostart: {:?}", e))
        }
    }
}

/// Check if autostart is enabled for the application
#[tauri::command]
pub fn is_autostart_enabled(app_handle: AppHandle) -> Result<bool, String> {
    debug!("Backend: Checking if autostart is enabled");
    match app_handle.autolaunch().is_enabled() {
        Ok(enabled) => {
            debug!("Backend: Autostart is {}", if enabled { "enabled" } else { "disabled" });
            Ok(enabled)
        },
        Err(e) => {
            error!("Backend: Failed to check if autostart is enabled: {:?}", e);
            Err(format!("Failed to check if autostart is enabled: {:?}", e))
        }
    }
}
