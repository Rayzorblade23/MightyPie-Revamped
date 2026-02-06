use serde::Serialize;

#[derive(Debug, Serialize)]
pub struct TextScaleInfo {
    pub scale_factor: f64,
}

#[cfg(target_os = "windows")]
#[tauri::command]
pub fn get_text_scale_factor() -> Result<f64, String> {
    use winreg::enums::*;
    use winreg::RegKey;

    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let text_scale_factor_path = r"Software\Microsoft\Accessibility";
    
    match hkcu.open_subkey(text_scale_factor_path) {
        Ok(key) => {
            let text_scale_factor_value: u32 = key
                .get_value("TextScaleFactor")
                .unwrap_or(100);
            let scale_factor = text_scale_factor_value as f64 / 100.0;
            log::debug!("Windows Text Scale Factor: {}", scale_factor);
            Ok(scale_factor)
        }
        Err(e) => {
            log::warn!("Failed to read TextScaleFactor from registry: {}", e);
            Ok(1.0) // Default to 1.0 if registry key doesn't exist
        }
    }
}

#[cfg(not(target_os = "windows"))]
#[tauri::command]
pub fn get_text_scale_factor() -> Result<f64, String> {
    Ok(1.0) // Non-Windows platforms don't have this setting
}
