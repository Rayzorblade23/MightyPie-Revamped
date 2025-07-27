use dotenvy::from_filename;
use enigo::{Coordinate, Enigo, Mouse, Settings};
use std::env;
use std::io::BufRead;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::thread;

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
fn get_mouse_pos() -> Result<(i32, i32), String> {
    let enigo = Enigo::new(&Settings::default())
        .map_err(|e| format!("Failed to initialize Enigo: {:?}", e))?;
    enigo
        .location()
        .map_err(|e| format!("Failed to get mouse location: {:?}", e))
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

// Helper function to read key-value pairs from .env files
fn read_env_file(file_path: &Path) -> std::io::Result<Vec<(String, String)>> {
    let file = match std::fs::File::open(file_path) {
        Ok(file) => file,
        Err(e) => {
            println!(
                "[TAURI LAUNCHER] Could not open env file {:?}: {}",
                file_path, e
            );
            return Ok(vec![]); // Return empty vec if file doesn't exist
        }
    };

    let reader = std::io::BufReader::new(file);
    let mut env_vars = Vec::new();

    for line in reader.lines() {
        let line = line?;
        let line = line.trim();

        // Skip comments and empty lines
        if line.is_empty() || line.starts_with('#') {
            continue;
        }

        // Parse key=value
        if let Some(idx) = line.find('=') {
            let (key, value) = line.split_at(idx);
            let key = key.trim();
            // Skip the '=' character
            let value = value.get(1..).unwrap_or("").trim();

            // Remove quotes if present
            let value = value.trim_matches('"').trim_matches('\'');

            if !key.is_empty() {
                env_vars.push((key.to_string(), value.to_string()));
                println!("[TAURI LAUNCHER] Loaded env var: {}={}", key, value);
            }
        }
    }

    Ok(env_vars)
}

// Helper function to find the project root directory
fn find_project_dir(resources_dir: &Path) -> PathBuf {
    if cfg!(debug_assertions) {
        // For debug builds, navigate up from target/debug to find the project root
        Path::new(resources_dir)
            .parent()
            .unwrap() // Go up from target directory
            .parent()
            .unwrap() // Go up from debug/release directory
            .parent()
            .unwrap() // Go up from src-tauri directory to project root
            .to_path_buf()
    } else {
        // If not in target/debug, try to find project root another way
        let current_dir =
            env::current_dir().unwrap_or_else(|_| Path::new(".").to_path_buf());
        println!("[TAURI LAUNCHER] Current directory: {:?}", current_dir);
        // Try to find project root by going up if we're in src-tauri
        if current_dir.ends_with("src-tauri") {
            current_dir.parent().unwrap_or(&current_dir).to_path_buf()
        } else {
            current_dir
        }
    }
}

fn start_launcher_thread(app_handle: tauri::AppHandle) {
    // Put launcher start code in a separate thread
    thread::spawn(move || {
        println!("[TAURI LAUNCHER] Starting universal launcher in a separate thread");

        // Keep a reference to the app handle for potential future use
        let _handle = app_handle.clone();

        // Determine if we're in debug/dev mode or production
        let is_debug = cfg!(debug_assertions);
        println!("[TAURI LAUNCHER] Debug assertions: {}", is_debug);

        let executable_path = env::current_exe().unwrap_or_else(|_| std::path::PathBuf::from("./"));
        println!(
            "[TAURI LAUNCHER] Current executable path: {:?}",
            executable_path
        );

        let resources_dir = executable_path
            .parent()
            .unwrap_or_else(|| Path::new("."));
        println!("[TAURI LAUNCHER] Resources directory: {:?}", resources_dir);

        // Check if we're in development or production mode
        let is_dev = cfg!(debug_assertions)
            || env::var("APP_ENV")
                .map(|val| val == "development")
                .unwrap_or(false);

        if is_dev {
            println!("[TAURI LAUNCHER] Running in development mode");
        } else {
            println!("[TAURI LAUNCHER] Running in production mode");
        }

        // Prepare the environment for the launcher
        let mut env_vars = env::vars().collect::<Vec<_>>();

        // Try to load environment variables from .env and .env.local files
        let project_dir = find_project_dir(resources_dir);
        println!("[TAURI LAUNCHER] Project directory: {:?}", project_dir);

        // Load environment variables from .env and .env.local files
        println!("[TAURI LAUNCHER] Looking for .env files in project directory");
        let env_path = project_dir.join(".env");
        let env_local_path = project_dir.join(".env.local");

        println!("[TAURI LAUNCHER] Checking for .env file at: {:?}", env_path);
        if let Ok(env_vars_from_file) = read_env_file(&env_path) {
            for (key, value) in env_vars_from_file {
                // Don't override existing environment variables
                if !env_vars.iter().any(|(k, _)| k == &key) {
                    env_vars.push((key, value));
                }
            }
        }

        println!(
            "[TAURI LAUNCHER] Checking for .env.local file at: {:?}",
            env_local_path
        );
        if let Ok(env_vars_from_file) = read_env_file(&env_local_path) {
            for (key, value) in env_vars_from_file {
                // .env.local overrides .env
                // First remove any existing entry with the same key
                env_vars.retain(|(k, _)| k != &key);
                env_vars.push((key, value));
            }
        }

        // Set mandatory environment variables
        if !is_dev {
            println!("[TAURI LAUNCHER] Setting APP_ENV to production");

            // Remove any existing APP_ENV
            env_vars.retain(|(k, _)| k != "APP_ENV");
            env_vars.push(("APP_ENV".to_string(), "production".to_string()));

            println!(
                "[TAURI LAUNCHER] Setting TAURI_RESOURCE_DIR to {:?}",
                resources_dir
            );
            env_vars.push((
                "TAURI_RESOURCE_DIR".to_string(),
                resources_dir.to_string_lossy().to_string(),
            ));
        } else {
            println!("[TAURI LAUNCHER] Setting APP_ENV to development");

            // Remove any existing APP_ENV
            env_vars.retain(|(k, _)| k != "APP_ENV");
            env_vars.push(("APP_ENV".to_string(), "development".to_string()));
        }

        // Mark that the launcher is being started from Tauri
        env_vars.push((
            "LAUNCHER_STARTED_FROM_TAURI".to_string(),
            "true".to_string(),
        ));
        println!("[TAURI LAUNCHER] Setting LAUNCHER_STARTED_FROM_TAURI flag");

        // Find the Node.js executable
        let node_path = if cfg!(target_os = "windows") {
            "node.exe"
        } else {
            "node"
        };
        println!("[TAURI LAUNCHER] Using Node.js executable: {}", node_path);

        // Find the launcher script - in production, it should be in the resources directory
        let launcher_script = if !is_dev {
            let script_path = Path::new(resources_dir).join("universal-launcher.js");
            println!(
                "[TAURI LAUNCHER] Production launcher script path: {:?}",
                script_path
            );
            script_path
        } else {
            // In development, use the script from the project directory
            let script_path = project_dir.join("scripts").join("universal-launcher.js");
            println!(
                "[TAURI LAUNCHER] Development launcher script path: {:?}",
                script_path
            );

            // Double-check if the script exists at the expected path
            if !script_path.exists() {
                println!(
                    "[TAURI LAUNCHER] Script not found at expected path, trying alternate path"
                );
                // Try a different path as fallback (current directory)
                let alt_script_path = Path::new(".")
                    .join("scripts")
                    .join("universal-launcher.js");
                println!("[TAURI LAUNCHER] Alternate path: {:?}", alt_script_path);

                if alt_script_path.exists() {
                    println!("[TAURI LAUNCHER] Found script at alternate path");
                    alt_script_path
                } else {
                    script_path // stick with original path for error reporting
                }
            } else {
                script_path
            }
        };

        // Ensure the launcher script exists
        if !launcher_script.exists() {
            eprintln!(
                "[TAURI LAUNCHER] ERROR: Launcher script not found at {:?}",
                launcher_script
            );
            return;
        }
        println!("[TAURI LAUNCHER] Confirmed launcher script exists");

        println!(
            "[TAURI LAUNCHER] Starting universal launcher: {} {:?}",
            node_path, launcher_script
        );

        // Spawn the Node.js process running our launcher
        let result = Command::new(node_path)
            .arg(launcher_script)
            .envs(env_vars)
            .spawn();

        match result {
            Ok(child) => println!(
                "[TAURI LAUNCHER] Successfully started launcher process with id: {:?}",
                child.id()
            ),
            Err(e) => eprintln!(
                "[TAURI LAUNCHER] ERROR: Failed to start universal launcher: {}",
                e
            ),
        }

        println!("[TAURI LAUNCHER] Universal launcher initialization complete");
    });
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .setup(|app| {
            // Start the universal launcher at app initialization
            println!("[TAURI LAUNCHER] Starting universal launcher initialization");
            start_launcher_thread(app.handle().clone());

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
                            let _ = window.emit("show-quickMenu", ());
                        }
                    }
                    _ => {}
                })
                .build(app)?;

            Ok(())
        })
        .plugin(tauri_plugin_positioner::init())
        .plugin(tauri_plugin_prevent_default::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_single_instance::init(|app, argv, cwd| {
            println!("{}, {argv:?}, {cwd}", app.package_info().name);
        }))
        .invoke_handler(tauri::generate_handler![
            greet,
            get_mouse_pos,
            get_private_env_var,
            set_mouse_pos,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
