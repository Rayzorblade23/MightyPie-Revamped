use enigo::{Coordinate, Enigo, Mouse, Settings};
use std::sync::{Mutex, OnceLock};

// Global static Enigo instance
static ENIGO_INSTANCE: OnceLock<Mutex<Option<Enigo>>> = OnceLock::new();

// Helper to get or initialize the Enigo instance
fn get_enigo_instance() -> Result<std::sync::MutexGuard<'static, Option<Enigo>>, String> {
    let instance = ENIGO_INSTANCE.get_or_init(|| Mutex::new(None));

    let mut guard = instance
        .lock()
        .map_err(|_| "Failed to acquire lock on Enigo instance".to_string())?;

    if guard.is_none() {
        *guard = Some(
            Enigo::new(&Settings::default())
                .map_err(|e| format!("Failed to initialize Enigo: {:?}", e))?,
        );
    }

    Ok(guard)
}

#[tauri::command]
pub fn get_mouse_pos() -> Result<(i32, i32), String> {
    let guard = get_enigo_instance()?;
    let enigo = guard.as_ref().unwrap();

    enigo.location().map_err(|e| format!("Failed to get mouse location: {:?}", e))
}

#[tauri::command]
pub fn set_mouse_pos(x: i32, y: i32) {
    let mut guard = get_enigo_instance().expect("Failed to get Enigo instance for set_mouse_pos");
    let enigo = guard.as_mut().unwrap();

    match enigo.move_mouse(x, y, Coordinate::Abs) {
        Ok(_) => {
            // Successfully moved the mouse.
        }
        Err(e) => {
            // Failed to move the mouse. Log the error.
            eprintln!("Failed to move mouse to ({}, {}) absolutely: {:?}", x, y, e);
        }
    }
}
