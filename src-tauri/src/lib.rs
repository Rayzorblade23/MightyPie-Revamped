use enigo::{Enigo, Settings, Mouse, Coordinate};
use std::env;
use tauri::command;
use tauri::Manager;

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
fn set_mouse_pos(x: i32, y: i32) {
    let mut enigo = Enigo::new(&Settings::default())
        .expect("Failed to initialize Enigo for set_mouse_pos");

    // The ONLY change needed is here: Absolute -> Abs
    match enigo.move_mouse(x, y, Coordinate::Abs) {
        Ok(_) => {
            // Successfully moved the mouse.
            // e.g., println!("Mouse moved to ({}, {}) absolutely", x, y);
        }
        Err(e) => {
            // Failed to move the mouse. Log the error.
            eprintln!("Failed to move mouse to ({}, {}) absolutely: {:?}", x, y, e);
        }
    }
}

use dotenvy::from_filename;

#[command]
fn get_private_env_var(key: String) -> Result<String, String> {
    // Load from .env.local if not already loaded
    let _ = from_filename(".env.local");

    match env::var(&key) {
        Ok(value) => Ok(value),
        Err(_) => Err(format!("Environment variable '{}' not found", key)),
    }
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
        .invoke_handler(tauri::generate_handler![
            greet,
            get_mouse_pos,
            get_private_env_var,
            set_mouse_pos,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
