use log::{debug, error, info};
use std::env;
use std::process::Command;

#[cfg(windows)]
use std::os::windows::process::CommandExt;

/// Task name used for the scheduled task
const TASK_NAME: &str = "MightyPieRevamped_Autostart";

/// Creates or updates a Windows Task Scheduler task to start the application at login
///
/// # Arguments
/// * `run_as_admin` - Whether the task should run with administrator privileges
///
/// # Returns
/// * `Result<(), String>` - Ok if successful, Err with error message otherwise
#[tauri::command]
pub fn create_startup_task(run_as_admin: bool) -> Result<(), String> {
    info!("Creating startup task with admin rights: {}", run_as_admin);

    // Get the path to the current executable
    let exe_path =
        env::current_exe().map_err(|e| format!("Failed to get executable path: {}", e))?;

    let exe_path_str = exe_path
        .to_str()
        .ok_or_else(|| "Invalid executable path".to_string())?;

    // First delete any existing task
    // Ignore errors if the task doesn't exist
    let _ = run_hidden_command("schtasks", &["/Delete", "/TN", TASK_NAME, "/F"]);

    // Create the new task with fixed arguments
    let result = if run_as_admin {
        run_hidden_command("schtasks", &["/Create", "/TN", TASK_NAME, "/TR", exe_path_str, "/SC", "ONLOGON", "/RL", "HIGHEST", "/F"])
    } else {
        run_hidden_command("schtasks", &["/Create", "/TN", TASK_NAME, "/TR", exe_path_str, "/SC", "ONLOGON", "/F"])
    }?;
    
    if !result.0 {
        let error_msg = result.2; // stderr
        error!("Failed to create startup task: {}", error_msg);
        return Err(format!("Failed to create startup task: {}", error_msg));
    }

    info!("Startup task created successfully");
    Ok(())
}

/// Removes the Windows Task Scheduler task for starting the application at login
///
/// # Returns
/// * `Result<(), String>` - Ok if successful, Err with error message otherwise
#[tauri::command]
pub fn remove_startup_task() -> Result<(), String> {
    info!("Removing startup task");

    let result = run_hidden_command("schtasks", &["/Delete", "/TN", TASK_NAME, "/F"])?;
    
    if !result.0 {
        let error_msg = result.2; // stderr
        // Don't consider it an error if the task doesn't exist
        if error_msg.contains("The system cannot find the file specified") {
            info!("No startup task found to remove");
            return Ok(());
        }
        error!("Failed to remove startup task: {}", error_msg);
        return Err(format!("Failed to remove startup task: {}", error_msg));
    }

    info!("Startup task removed successfully");
    Ok(())
}

/// Checks if a Windows Task Scheduler task exists for starting the application at login
///
/// # Returns
/// * `Result<bool, String>` - Ok(true) if task exists, Ok(false) if it doesn't, Err with error message on error
#[tauri::command]
pub fn is_startup_task_enabled() -> Result<bool, String> {

    let result = run_hidden_command("schtasks", &["/Query", "/TN", TASK_NAME])?;
    
    let exists = result.0;
    debug!("Startup task exists: {}", exists);
    Ok(exists)
}

/// Checks if the Windows Task Scheduler task is configured to run with administrator privileges
///
/// # Returns
/// * `Result<bool, String>` - Ok(true) if task runs as admin, Ok(false) if it doesn't or doesn't exist,
///                           Err with error message on error
#[tauri::command]
pub fn is_startup_task_admin() -> Result<bool, String> {

    // First check if the task exists
    if !is_startup_task_enabled()? {
        return Ok(false);
    }
    
    // Use schtasks to get XML output and check for RunLevel="HighestAvailable"
    let result = run_hidden_command("schtasks", &["/Query", "/TN", TASK_NAME, "/XML"])?;
    
    if result.0 {
        let xml_output = result.1; // Use stdout from the command
        
        // Check for different possible formats of RunLevel in the XML
        let is_admin = xml_output.contains("RunLevel=\"HighestAvailable\"") || 
                       xml_output.contains("RunLevel='HighestAvailable'") ||
                       xml_output.contains("<RunLevel>HighestAvailable</RunLevel>");
        
        debug!("Startup task runs as admin: {}", is_admin);
        Ok(is_admin)
    } else {
        let error_msg = result.2; // Use stderr for error message
        error!("Failed to get task XML: {}", error_msg);
        Err(format!("Failed to get task XML: {}", error_msg))
    }
}

/// Run a command without showing a window
/// This approach uses the standard Command with DETACHED_PROCESS flag
/// which should not trigger Windows Defender false positives
fn run_hidden_command(command: &str, args: &[&str]) -> Result<(bool, String, String), String> {
    // Use the windows_process_extensions crate feature to set creation flags
    #[cfg(windows)]
    {
        // Use DETACHED_PROCESS flag instead of CREATE_NO_WINDOW
        // This detaches the process from the console without triggering Defender
        const DETACHED_PROCESS: u32 = 0x00000008;
        
        let output = Command::new(command)
            .args(args)
            .creation_flags(DETACHED_PROCESS)
            .output();
            
        match output {
            Ok(output) => {
                let success = output.status.success();
                let stdout = String::from_utf8_lossy(&output.stdout).to_string();
                let stderr = String::from_utf8_lossy(&output.stderr).to_string();
                
                Ok((success, stdout, stderr))
            },
            Err(e) => Err(format!("Failed to execute command: {}", e))
        }
    }
    
    #[cfg(not(windows))]
    {
        // Fallback for non-Windows platforms
        let output = Command::new(command)
            .args(args)
            .output();
            
        match output {
            Ok(output) => {
                let success = output.status.success();
                let stdout = String::from_utf8_lossy(&output.stdout).to_string();
                let stderr = String::from_utf8_lossy(&output.stderr).to_string();
                
                Ok((success, stdout, stderr))
            },
            Err(e) => Err(format!("Failed to execute command: {}", e))
        }
    }
}
