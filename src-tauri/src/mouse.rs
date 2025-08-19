use std::thread;
use std::time::{Duration, Instant};
use tauri::Window;

// Windows API for cursor position and movement
#[cfg(target_os = "windows")]
use windows::Win32::{
    Foundation::POINT,
    UI::WindowsAndMessaging::{
        GetCursorPos,
        SetCursorPos,
    },
};

#[cfg(target_os = "windows")]
fn get_cursor_pos_win32() -> Result<(i32, i32), String> {
    unsafe {
        let mut pt = POINT { x: 0, y: 0 };
        match GetCursorPos(&mut pt) {
            Ok(()) => Ok((pt.x, pt.y)),
            Err(e) => Err(format!("GetCursorPos failed: {:?}", e)),
        }
    }
}

#[cfg(not(target_os = "windows"))]
fn get_cursor_pos_win32() -> Result<(i32, i32), String> {
    Err("Win32 GetCursorPos not available on this platform".to_string())
}

#[tauri::command]
pub fn get_mouse_pos(window: Window) -> Result<(i32, i32), String> {
    // Retry only while we see Access Denied, up to a time cap tuned for post-sleep (e.g., ~2s).
    const MAX_RETRY_DURATION_MS: u64 = 2000;
    const RETRY_DELAY_MS: u64 = 16; // ~1 frame at 60Hz

    let start = Instant::now();
    loop {
        match get_cursor_pos_win32() {
            Ok((x, y)) => return Ok((x, y)),
            Err(e) => {
                let is_access_denied = e.contains("0x80070005") || e.to_lowercase().contains("access is denied");
                if is_access_denied && start.elapsed() < Duration::from_millis(MAX_RETRY_DURATION_MS) {
                    // Transient post-sleep error: wait and retry without recording last_err yet
                    thread::sleep(Duration::from_millis(RETRY_DELAY_MS));
                    continue;
                }
                // Non-retryable error or retries exhausted: exit the loop and perform fallback
                break;
            }
        }
    }

    // After exhausting retries (or for non-AccessDenied errors), fall back to window center to avoid frontend error paths.
    // Proactively hide the window as we may be a transparent overlay; this mirrors previous frontend intent.
    // Best-effort hide; ignore result to avoid introducing new errors
    let _ = window.hide();
    // Best-effort center computation; if any call fails, return a generic error
    let pos = match window.outer_position() {
        Ok(p) => p,
        Err(_) => return Err("Failed to read window position".to_string()),
    };
    let size = match window.outer_size() {
        Ok(s) => s,
        Err(_) => return Err("Failed to read window size".to_string()),
    };
    let cx = pos.x + (size.width as i32) / 2;
    let cy = pos.y + (size.height as i32) / 2;
    return Ok((cx, cy));
}

#[tauri::command]
pub fn set_mouse_pos(x: i32, y: i32) {
    // Windows native absolute move
    #[cfg(target_os = "windows")]
    unsafe {
        match SetCursorPos(x, y) {
            Ok(()) => {
                // success
            }
            Err(e) => {
                eprintln!("SetCursorPos failed for ({}, {}): {:?}", x, y, e);
            }
        }
    }
}
