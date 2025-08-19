// Define modules
mod env_utils;
mod file_fetch_utils;
mod launcher;
mod logging;
mod mouse;
mod nats_config;
mod nats_token;
mod port_checker;
mod shutdown;
mod admin;
mod task_scheduler;

// Re-export items from modules for external use
pub use env_utils::{get_private_env_var, set_env_var, get_app_data_dir};
pub use file_fetch_utils::{get_icon_data_url, read_button_functions};
pub use logging::{get_log_dir, get_log_file_path, get_logs, log_from_frontend, get_log_level};
pub use mouse::{get_mouse_pos, set_mouse_pos};
pub use admin::{is_running_as_admin, restart_as_admin};
pub use task_scheduler::{create_startup_task, remove_startup_task, is_startup_task_enabled, is_startup_task_admin};

use env_logger::{self, Builder, Env};
use std::env;
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Emitter, Manager,
};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Environment variables are now baked into the binary at build time
    println!("Using environment variables baked into the binary at build time");

    // Initialize shutdown handler
    shutdown::init();

    // Configure env_logger to respect RUST_LOG from environment
    // with sensible defaults if not set
    let env = Env::default().filter_or("RUST_LOG", "info");

    let mut builder = Builder::from_env(env);

    // Add any module-specific filters here if needed (Enigo removed)

    builder
        .format(|buf, record| {
            use std::io::Write;
            let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S");
            let level = record.level().to_string();
            let message = format!("{}", record.args());

            // Add to circular buffer
            let log_buffer = logging::get_log_buffer();
            if let Ok(buffer) = log_buffer.lock() {
                buffer.add(logging::LogEntry {
                    timestamp: timestamp.to_string(),
                    level,
                    message: message.clone(),
                });
            }

            // Use [TAURI] prefix for logs from this crate, [SVELTE] for others
            let prefix = if record
                .module_path()
                .unwrap_or("")
                .starts_with("mightypie_revamped")
            {
                "\x1b[36m[TAURI ]\x1b[0m"
            } else {
                "\x1b[38;5;180m[SVELTE]\x1b[0m"
            };

            // Format with appropriate prefix
            writeln!(buf, "{} {} {}", prefix, timestamp, record.args())
        })
        .init();

    // Use standardized log format with timestamp and level
    log::info!("MightyPie logging initialized with circular buffer");

    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .setup(|app| {
            // Start the universal launcher at app initialization
            launcher::start_launcher_thread(app.handle().clone());

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
            get_mouse_pos,
            set_mouse_pos,
            get_private_env_var,
            log_from_frontend,
            get_logs,
            get_log_file_path,
            get_log_dir,
            read_button_functions,
            get_icon_data_url,
            get_log_level,
            get_app_data_dir,
            is_running_as_admin,
            restart_as_admin,
            create_startup_task,
            remove_startup_task,
            is_startup_task_enabled,
            is_startup_task_admin
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
