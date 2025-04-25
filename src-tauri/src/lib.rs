use enigo::{Enigo, Mouse, Settings};
use tauri::Manager;
use tauri::command;
use std::path::PathBuf;
use dotenvy::from_path_iter;
use std::collections::HashMap;

// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}

pub struct MousePos {
    pub x: i32,
    pub y: i32,
}

#[command]
fn get_mouse_pos() -> (i32, i32) {
    let enigo = Enigo::new(&Settings::default());
    enigo.unwrap().location().unwrap()
}


#[command]
fn get_all_public_env_vars() -> Result<HashMap<String, String>, String> {
    let mut path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    path.pop(); // Go to project root
    path.push(".env.public");

    // Use from_path_iter to parse the .env file
    let iter = from_path_iter(path).map_err(|e| e.to_string())?;

    let mut env_vars = HashMap::new();

    // Iterate through the parsed items
    for item in iter {
        match item {
            Ok((key, value)) => {
                // If parsing was successful for the line, insert into map
                env_vars.insert(key, value);
            }
            Err(e) => {
                // Log errors for individual lines but continue parsing
                eprintln!("Error parsing .env.public line: {}", e);
                // You might choose to return an error here if *any* line fails,
                // but logging and skipping is often acceptable for env files.
            }
        }
    }

    Ok(env_vars)
}


#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            let window = app.get_webview_window("main").unwrap();
            window.set_always_on_top(true)?;
            Ok(())
        })
//         .plugin(tauri_plugin_log::Builder::new()
// //           .level(log::LevelFilter::Trace)
//             .format(|out, message, record| {
//                     if record.level() == log::Level::Debug {
//                       return; // Skip DEBUG logs entirely (they won't appear)
//                     }
//             out.finish(format_args!(
//               "[{}] {}",
//               record.level(),
// //               record.target(),
//               message
//             ))
//           })
//           .build())
        .plugin(tauri_plugin_positioner::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet, get_mouse_pos,get_all_public_env_vars])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
