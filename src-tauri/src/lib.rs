use dotenvy::from_filename;
use enigo::{Coordinate, Enigo, Mouse, Settings};
use std::env;
use tauri::{
    command,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Emitter, Manager,
};

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
    let mut enigo =
        Enigo::new(&Settings::default()).expect("Failed to initialize Enigo for set_mouse_pos");

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
        .plugin(tauri_plugin_dialog::init())
        .setup(|app| {
            let window = app.get_webview_window("main").unwrap();
            window.set_always_on_top(true)?;

            // Create menu items
            let settings_item = MenuItem::with_id(app, "settings", "Settings", true, None::<&str>)?;
            let piemenuconfig_item =
                MenuItem::with_id(app, "piemenuconfig", "Pie Menu Config", true, None::<&str>)?;
            let exit_item = MenuItem::with_id(app, "exit", "Exit", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&settings_item, &piemenuconfig_item, &exit_item])?;

            // Tray icon setup using Tauri 2.x API, minimal example from docs
            TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("Mighty Pie")
                .menu(&menu)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "settings" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_always_on_top(true);
                            let _ = window.set_focus();
                            let _ = window.emit("show-settings", ());
                        }
                    }
                    "piemenuconfig" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_always_on_top(true);
                            let _ = window.set_focus();
                            let _ = window.emit("show-piemenuconfig", ());
                        }
                    }
                    "exit" => {
                        app.exit(0);
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| match event {
                    TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } => {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_always_on_top(true);
                            let _ = window.set_focus();
                            let _ = window.emit("show-specialMenu", ());
                        }
                    }
                    _ => {}
                })
                .build(app)?;
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
        .plugin(tauri_plugin_prevent_default::init())
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
