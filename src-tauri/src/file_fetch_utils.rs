use base64::{engine::general_purpose, Engine as _};
use std::env;
use std::fs;
use std::path::{Path};

// Command to read buttonFunctions.json file
#[tauri::command]
pub fn read_button_functions() -> Result<String, String> {
    let root_dir = env::var("MIGHTYPIE_ROOT_DIR").map_err(|e| e.to_string())?;
    let assets_dir =
        env::var("PUBLIC_DIR_ASSETS").unwrap_or_else(|_| "src-tauri/assets".to_string());
    let button_functions_path = env::var("PUBLIC_DIR_BUTTONFUNCTIONS")
        .unwrap_or_else(|_| "data/buttonFunctions.json".to_string());

    // Use PathBuf for proper cross-platform path handling
    let full_path = Path::new(&root_dir)
        .join(&assets_dir)
        .join(&button_functions_path);

    fs::read_to_string(&full_path).map_err(|e| e.to_string())
}

// Command to convert an icon path to a data URL
#[tauri::command]
pub fn get_icon_data_url(icon_path: &str) -> Result<String, String> {
    let path = Path::new(icon_path);
    let full_path = if path.is_absolute() {
        path.to_path_buf()
    } else {
        let app_name =
            env::var("PUBLIC_APPNAME").unwrap_or_else(|_| "MightyPieRevamped".to_string());
        let clean_path = if icon_path.starts_with('/') || icon_path.starts_with('\\') {
            &icon_path[1..]
        } else {
            icon_path
        };
        let local_app_data = env::var("LOCALAPPDATA")
            .unwrap_or_else(|_| env::var("APPDATA").unwrap_or_else(|_| ".".to_string()));
        let app_data_dir = Path::new(&local_app_data).join(app_name);
        app_data_dir.join(clean_path)
    };
    let image_data = fs::read(&full_path).map_err(|e| e.to_string())?;
    let mime_type = match full_path.extension().and_then(|ext| ext.to_str()) {
        Some("png") => "image/png",
        Some("jpg") | Some("jpeg") => "image/jpeg",
        Some("svg") => "image/svg+xml",
        Some("gif") => "image/gif",
        Some("webp") => "image/webp",
        _ => "application/octet-stream",
    };
    let base64_data = general_purpose::STANDARD.encode(&image_data);
    Ok(format!("data:{};base64,{}", mime_type, base64_data))
}