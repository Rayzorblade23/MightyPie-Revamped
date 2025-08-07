use log::{info, warn};
use std::env;
use std::fs::{self, File};
use std::io::Write;
use std::path::PathBuf;
use std::process::Command;
use std::thread;
use std::time::Duration;

#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;

#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x08000000;

// Path to the temporary file used to signal admin restart
fn get_admin_flag_path() -> PathBuf {
    let mut path = env::temp_dir();
    path.push("mightypie_admin_restart.flag");
    path
}

#[tauri::command]
/// Check if the current process is running with administrator privileges
pub fn is_running_as_admin() -> bool {
    #[cfg(target_os = "windows")]
    {
        // On Windows, we can use the "net session" command to check for admin rights
        // This command will succeed only if the process has admin privileges
        match Command::new("net")
            .args(["session"])
            .creation_flags(CREATE_NO_WINDOW)
            .output() 
        {
            Ok(output) => {
                let success = output.status.success();
                info!("Admin check result: {}", success);
                success
            },
            Err(e) => {
                warn!("Failed to check admin status: {}", e);
                false
            },
        }
    }
    
    #[cfg(not(target_os = "windows"))]
    {
        false // Non-Windows platforms always return false
    }
}


/// Restart the application with administrator privileges
/// Returns true if the restart was initiated, false otherwise
#[tauri::command]
pub fn restart_as_admin() -> bool {
    #[cfg(target_os = "windows")]
    {
        // Get the path to the current executable
        if let Ok(exe_path) = env::current_exe() {
            if let Some(exe_path_str) = exe_path.to_str() {
                info!(
                    "Attempting to restart with admin privileges: {}",
                    exe_path_str
                );

                // Create a temporary file to signal that we want to restart with admin privileges
                // This helps with the single-instance plugin issue
                let flag_path = get_admin_flag_path();
                if let Ok(mut file) = File::create(&flag_path) {
                    if file.write_all(b"admin").is_err() {
                        warn!("Failed to write admin flag file");
                    }
                    info!("Created admin flag file at {:?}", flag_path);
                } else {
                    warn!("Failed to create admin flag file");
                }

                // Try using PowerShell to elevate privileges with proper escaping
                let ps_script = format!(
                    "Start-Process -FilePath '{}' -Verb RunAs",
                    exe_path_str.replace("'", "''")
                );

                info!("Executing PowerShell elevation script: {}", ps_script);
                match Command::new("powershell")
                    .args([
                        "-NoProfile",
                        "-ExecutionPolicy",
                        "Bypass",
                        "-Command",
                        &ps_script,
                    ])
                    .creation_flags(CREATE_NO_WINDOW)
                    .spawn()
                {
                    Ok(_) => {
                        info!("Successfully initiated admin restart using PowerShell");
                        // Add a delay to allow the new process to start before this one exits
                        thread::sleep(Duration::from_millis(1500));
                        
                        // Exit the current process to allow the new admin process to take over
                        // This is critical for singleton mode to work properly
                        info!("Exiting current process to allow admin process to start");
                        std::process::exit(0);
                    }
                    Err(e) => {
                        warn!(
                            "Failed to restart with admin privileges using PowerShell: {}",
                            e
                        );

                        // Fallback to using cmd.exe with 'runas' verb
                        info!("Trying fallback method with cmd.exe");
                        let cmd_args = [
                            "/C",
                            &format!("runas /user:Administrator \"{}\"", exe_path_str),
                        ];

                        match Command::new("cmd.exe")
                            .args(cmd_args)
                            .creation_flags(CREATE_NO_WINDOW)
                            .spawn()
                        {
                            Ok(_) => {
                                info!(
                                    "Successfully initiated admin restart using cmd.exe fallback"
                                );
                                thread::sleep(Duration::from_millis(1500));
                                
                                // Exit the current process to allow the new admin process to take over
                                info!("Exiting current process to allow admin process to start");
                                std::process::exit(0);
                            }
                            Err(e) => {
                                warn!("Failed to restart with admin privileges using cmd.exe fallback: {}", e);
                                // Clean up flag file if all restart attempts failed
                                if let Err(e) = fs::remove_file(&flag_path) {
                                    warn!("Failed to remove admin flag file: {}", e);
                                }
                            }
                        }
                    }
                }
            }
        }
        false
    }
}
