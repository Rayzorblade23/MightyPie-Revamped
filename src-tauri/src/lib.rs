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

#[cfg(target_os = "windows")]
#[tauri::command]
fn show_pause_indicator_without_focus(app: tauri::AppHandle) -> Result<(), String> {
    use raw_window_handle::{HasWindowHandle, RawWindowHandle};
    
    if let Some(window) = app.get_webview_window("shortcut_pause_indicator") {
        if let Ok(handle) = window.window_handle() {
            if let RawWindowHandle::Win32(win32_handle) = handle.as_raw() {
                unsafe {
                    let hwnd = windows::Win32::Foundation::HWND(win32_handle.hwnd.get() as *mut _);
                    let _ = ShowWindow(hwnd, SW_SHOWNOACTIVATE);
                }
                return Ok(());
            }
        }
    }
    Err("Failed to show pause indicator".to_string())
}

#[cfg(not(target_os = "windows"))]
#[tauri::command]
fn show_pause_indicator_without_focus(app: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app.get_webview_window("shortcut_pause_indicator") {
        let _ = window.show();
        Ok(())
    } else {
        Err("Failed to show pause indicator".to_string())
    }
}

#[cfg(target_os = "windows")]
#[tauri::command]
fn hide_pause_indicator(app: tauri::AppHandle) -> Result<(), String> {
    use raw_window_handle::{HasWindowHandle, RawWindowHandle};
    
    if let Some(window) = app.get_webview_window("shortcut_pause_indicator") {
        if let Ok(handle) = window.window_handle() {
            if let RawWindowHandle::Win32(win32_handle) = handle.as_raw() {
                unsafe {
                    let hwnd = windows::Win32::Foundation::HWND(win32_handle.hwnd.get() as *mut _);
                    let _ = ShowWindow(hwnd, SW_HIDE);
                }
                return Ok(());
            }
        }
    }
    Err("Failed to hide pause indicator".to_string())
}

#[cfg(not(target_os = "windows"))]
#[tauri::command]
fn hide_pause_indicator(app: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app.get_webview_window("shortcut_pause_indicator") {
        let _ = window.hide();
        Ok(())
    } else {
        Err("Failed to hide pause indicator".to_string())
    }
}

#[tauri::command]
fn exit_app(app: tauri::AppHandle) {
    app.exit(0);
}

#[tauri::command]
fn update_tray_pause_menu_item(app: tauri::AppHandle, is_paused: bool) -> Result<(), String> {
    use tauri::menu::{Menu, MenuItem};
    
    // Recreate the menu with updated text
    let settings_item = MenuItem::with_id(&app, "settings", "Settings", true, None::<&str>)
        .map_err(|e| format!("Failed to create settings item: {}", e))?;
    let piemenuconfig_item = MenuItem::with_id(&app, "piemenuconfig", "Pie Menu Config", true, None::<&str>)
        .map_err(|e| format!("Failed to create piemenuconfig item: {}", e))?;
    
    let toggle_text = if is_paused {
        "Resume Pie Menu shortcut Detection"
    } else {
        "Pause Pie Menu shortcut Detection"
    };
    let toggle_pause_item = MenuItem::with_id(&app, "toggle_pause", toggle_text, true, None::<&str>)
        .map_err(|e| format!("Failed to create toggle_pause item: {}", e))?;
    
    let exit_item = MenuItem::with_id(&app, "exit", "Exit", true, None::<&str>)
        .map_err(|e| format!("Failed to create exit item: {}", e))?;
    
    let menu = Menu::with_items(&app, &[&settings_item, &piemenuconfig_item, &toggle_pause_item, &exit_item])
        .map_err(|e| format!("Failed to create menu: {}", e))?;
    
    if let Some(tray) = app.tray_by_id("main") {
        tray.set_menu(Some(menu))
            .map_err(|e| format!("Failed to set menu: {}", e))?;
    }
    
    Ok(())
}

use env_logger::{self, Builder, Env};
use std::env;
use tauri::{
    LogicalSize, PhysicalPosition,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Emitter, Manager,
};

#[cfg(target_os = "windows")]
use windows::Win32::UI::WindowsAndMessaging::{
    GetWindowLongW, SetWindowLongW, ShowWindow, GWL_EXSTYLE, SW_HIDE, SW_SHOWNOACTIVATE, WS_EX_NOACTIVATE
};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    const SHORTCUT_PAUSE_INDICATOR_SIZE_PX: f64 = 36.0;
    const SHORTCUT_PAUSE_INDICATOR_SIZE_PX_I32: i32 = 36;
    const SHORTCUT_PAUSE_INDICATOR_EDGE_INSET_PX_I32: i32 = 8;

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

            {
                if let Ok(aux) = tauri::WebviewWindowBuilder::new(
                    app,
                    "shortcut_pause_indicator",
                    tauri::WebviewUrl::App("/shortcutPauseIndicator".into()),
                )
                .title("Shortcut Paused")
                .inner_size(
                    SHORTCUT_PAUSE_INDICATOR_SIZE_PX,
                    SHORTCUT_PAUSE_INDICATOR_SIZE_PX,
                )
                .resizable(false)
                .decorations(false)
                .transparent(true)
                .shadow(false)
                .skip_taskbar(true)
                .build()
                {
                    let _ = aux.set_ignore_cursor_events(true);
                    let _ = aux.set_always_on_top(true);
                    
                    // Set WS_EX_NOACTIVATE on Windows to prevent focus stealing
                    #[cfg(target_os = "windows")]
                    {
                        use raw_window_handle::{HasWindowHandle, RawWindowHandle};
                        if let Ok(handle) = aux.window_handle() {
                            if let RawWindowHandle::Win32(win32_handle) = handle.as_raw() {
                                unsafe {
                                    let hwnd = windows::Win32::Foundation::HWND(win32_handle.hwnd.get() as *mut _);
                                    let ex_style = GetWindowLongW(hwnd, GWL_EXSTYLE);
                                    SetWindowLongW(hwnd, GWL_EXSTYLE, ex_style | WS_EX_NOACTIVATE.0 as i32);
                                }
                            }
                        }
                    }
                    
                    let _ = aux.hide();

                    // Make the window 18x18 *physical* pixels (DPI-aware)
                    let scale = aux.scale_factor().unwrap_or(1.0);
                    let logical = SHORTCUT_PAUSE_INDICATOR_SIZE_PX / scale;
                    let _ = aux.set_size(tauri::Size::Logical(LogicalSize::new(logical, logical)));

                    // Position on the right edge of the primary monitor, 1/3 from the top
                    if let Ok(Some(monitor)) = aux.primary_monitor() {
                        let pos = monitor.position();
                        let size = monitor.size();

                        let x = pos.x
                            + (size.width as i32)
                            - SHORTCUT_PAUSE_INDICATOR_SIZE_PX_I32
                            - SHORTCUT_PAUSE_INDICATOR_EDGE_INSET_PX_I32;
                        let y = pos.y + (((size.height as i32) * 7) / 8);
                        let _ = aux.set_position(tauri::Position::Physical(PhysicalPosition::new(x, y)));
                    }
                }
            }

            // Create menu items
            let settings_item = MenuItem::with_id(app, "settings", "Settings", true, None::<&str>)?;
            let piemenuconfig_item =
                MenuItem::with_id(app, "piemenuconfig", "Pie Menu Config", true, None::<&str>)?;
            let toggle_pause_item = MenuItem::with_id(app, "toggle_pause", "Toggle Pause", true, None::<&str>)?;
            let exit_item = MenuItem::with_id(app, "exit", "Exit", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&settings_item, &piemenuconfig_item, &toggle_pause_item, &exit_item])?;

            // Tray icon setup using Tauri 2.x API, minimal example from docs
            TrayIconBuilder::with_id("main")
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
                    "toggle_pause" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.emit("toggle-pause", ());
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
            get_app_data_dir,
            get_icon_data_url,
            read_button_functions,
            get_log_dir,
            get_log_file_path,
            get_logs,
            log_from_frontend,
            get_log_level,
            is_running_as_admin,
            restart_as_admin,
            create_startup_task,
            remove_startup_task,
            exit_app,
            is_startup_task_enabled,
            is_startup_task_admin,
            show_pause_indicator_without_focus,
            hide_pause_indicator,
            update_tray_pause_menu_item
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
