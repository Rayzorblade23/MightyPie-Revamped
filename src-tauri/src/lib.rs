use enigo::{Enigo, Mouse, Settings};


// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}

pub struct MousePos {
    pub x: i32,
    pub y: i32,
}

#[tauri::command]
fn get_mouse_pos() -> (i32, i32) {
    let enigo = Enigo::new(&Settings::default());
    enigo.unwrap().location().unwrap()
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_log::Builder::new()
          .format(|out, message, record| {
            out.finish(format_args!(
              "[{}] {}",
              record.level(),
//               record.target(),
              message
            ))
          })
          .build())
        .plugin(tauri_plugin_positioner::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet, get_mouse_pos])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
