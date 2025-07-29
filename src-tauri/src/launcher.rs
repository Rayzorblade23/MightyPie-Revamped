use std::env;
use std::path::{Path, PathBuf};
use std::thread;
use std::process::{Command, Stdio};
use std::io::{BufRead, BufReader};
use tauri::Manager;
use crate::logging::log_to_file;
use crate::env_utils::{is_debug, set_env_var};
use crate::shutdown;

pub fn start_launcher_thread(app_handle: tauri::AppHandle) {
    // Colorized log function similar to chalk in JS
    let log_info = |msg: &str| {
        let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S");
        println!("\x1b[36m[TAURI ]\x1b[0m {} {}", timestamp, msg);
        
        // Log directly to file
        let log_message = format!("[TAURI ] {} {}", timestamp, msg);
        log_to_file(&log_message);
    };

    let log_error = |msg: &str| {
        let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S");
        println!(
            "\x1b[31m[TAUERR]\x1b[0m {} {}",
            timestamp, msg
        );
        
        // Log directly to file
        let log_message = format!("[TAUERR] {} {}", timestamp, msg);
        log_to_file(&log_message);
    };

    // Put launcher start code in a separate thread
    thread::spawn(move || {
        log_info("Initializing launcher");

        // Determine if we're in debug/dev mode or production
        let is_debug = is_debug();

        // Get resource directory from Tauri
        let resources_dir = app_handle
            .path()
            .resolve(".", tauri::path::BaseDirectory::Resource)
            .unwrap_or_else(|_| PathBuf::from("./"));

        // Get the current executable path
        let current_exe = env::current_exe().unwrap_or_else(|_| PathBuf::from("./"));
        
        // In development mode, we need to navigate from the executable to the project root
        // The executable is typically in src-tauri/target/debug/
        let project_dir = if is_debug {
            current_exe
                .parent() // src-tauri/target/debug -> src-tauri/target
                .and_then(|p| p.parent()) // src-tauri/target -> src-tauri
                .and_then(|p| p.parent()) // src-tauri -> project root
                .map_or_else(|| PathBuf::from("."), |p| p.to_path_buf())
        } else {
            // In production, resources_dir is already set correctly
            resources_dir.clone()
        };
        
        // Determine if we're in development or production mode
        let is_dev = is_debug;

        if is_dev {
            log_info("Running in development mode");
        } else {
            log_info("Running in production mode");
        }
        
        // For development mode, we need to go up one level to find .env files in project root
        let env_dir = if is_dev {
            project_dir
                .parent() // src-tauri -> project root
                .unwrap_or_else(|| Path::new("."))
                .to_path_buf()
        } else {
            project_dir.clone()
        };
        
        // Load environment variables from .env files
        let env_path = env_dir.join(".env");
        log_info(&format!("Looking for .env at: {:?}", env_path));
        if env_path.exists() {
            match dotenvy::from_filename(&env_path) {
                Ok(_) => {
                    log_info("Successfully loaded .env file");
                },
                Err(e) => {
                    log_error(&format!("Could not open env file {:?}: {}", env_path, e));
                }
            }
        } else {
            log_error(&format!(".env file not found at {:?}", env_path));
        }

        // Also check for .env.local which takes precedence
        let env_local_path = env_dir.join(".env.local");
        log_info(&format!("Looking for .env.local at: {:?}", env_local_path));
        if env_local_path.exists() {
            match dotenvy::from_filename(&env_local_path) {
                Ok(_) => {
                    log_info("Successfully loaded .env.local file");
                },
                Err(e) => {
                    log_error(&format!(
                        "Could not open env.local file {:?}: {}",
                        env_local_path, e
                    ));
                }
            }
        } else {
            log_info(&format!(".env.local file not found at {:?}", env_local_path));
        }

        // Set environment variables for the launcher
        if is_dev {
            env::set_var("APP_ENV", "development");
            
            // Set the project root directory for the Go backend to use
            if let Some(root_dir) = project_dir.parent() {
                set_env_var("MIGHTYPIE_ROOT_DIR", root_dir.to_str().unwrap_or(""));
                log_info(&format!("Set MIGHTYPIE_ROOT_DIR to: {:?}", root_dir));
            }
        } else {
            env::set_var("APP_ENV", "production");
            env::set_var("TAURI_RESOURCE_DIR", resources_dir.to_str().unwrap());
            
            // In production, the root is the project_dir itself
            set_env_var("MIGHTYPIE_ROOT_DIR", project_dir.to_str().unwrap_or(""));
            log_info(&format!("Set MIGHTYPIE_ROOT_DIR to: {:?}", project_dir));
        }

        // Flag that we're starting from Tauri
        env::set_var("LAUNCHER_STARTED_FROM_TAURI", "true");

        // Find the Go executable path
        let go_executable_path = if is_dev {
            // In development, the Go executable is in src-tauri/assets/src-go/bin/
            let go_path = project_dir
                .join("assets")
                .join("src-go")
                .join("bin")
                .join("main.exe");
            
            go_path
        } else {
            // In production, the Go executable should be in the resources directory
            resources_dir.join("bin").join("main.exe")
        };

        // Get the Go executable's directory to use as working directory
        let go_working_dir = go_executable_path.parent().unwrap_or_else(|| Path::new(".")).to_path_buf();
        
        // Start the Go executable
        log_info(&format!("Starting Go executable: {:?}", go_executable_path));
        log_info(&format!("Working directory: {:?}", go_working_dir));

        if !go_executable_path.exists() {
            log_error(&format!("Go executable not found at {:?}", go_executable_path));
            return;
        }

        // Create a command with piped stdout and stderr
        let mut command = Command::new(&go_executable_path);
        
        // Get the current process ID to pass to the Go process
        let current_pid = std::process::id();
        log_info(&format!("Current Tauri process ID: {}", current_pid));
        
        command.current_dir(&go_working_dir)
               .stdout(Stdio::piped())
               .stderr(Stdio::piped())
               .env("LAUNCHER_STARTED_FROM_TAURI", "1")
               .env("TAURI_PROCESS_PID", current_pid.to_string());

        match command.spawn() {
            Ok(mut child) => {
                let pid = child.id();
                log_info(&format!(
                    "Successfully started Go process with id: {:?}",
                    pid
                ));
                
                // Store stdout and stderr before registering the process
                let stdout = child.stdout.take();
                let stderr = child.stderr.take();
                
                // Register the process with the shutdown handler
                shutdown::register_process(&child);
                
                // Handle stdout in a separate thread with [GO] prefix
                if let Some(stdout) = stdout {
                    thread::spawn(move || {
                        let reader = BufReader::new(stdout);
                        for line in reader.lines() {
                            if let Ok(line) = line {
                                // Add color based on log level
                                let colored_line = colorize_go_log(&line);
                                
                                // Format with [GO] prefix and print to console
                                println!("\x1b[34m[  GO  ]\x1b[0m {}", colored_line);
                                
                                // Log to file with [GO] prefix (without color codes)
                                let clean_line = strip_ansi_codes(&line);
                                log_to_file(&format!("[  GO  ] {}", clean_line));
                            }
                        }
                    });
                }
                
                // Handle stderr in a separate thread with [GO] prefix
                if let Some(stderr) = stderr {
                    thread::spawn(move || {
                        let reader = BufReader::new(stderr);
                        for line in reader.lines() {
                            if let Ok(line) = line {
                                // Add color based on log level
                                let colored_line = colorize_go_log(&line);
                                
                                // Format with [GO] prefix and print to console
                                println!("\x1b[31m[  GO  ]\x1b[0m {}", colored_line);
                                
                                // Log to file with [GO] prefix (without color codes)
                                let clean_line = strip_ansi_codes(&line);
                                log_to_file(&format!("[  GO  ] {}", clean_line));
                            }
                        }
                    });
                }
            }
            Err(e) => {
                log_error(&format!("Failed to start Go executable: {}", e));
            }
        }

        log_info("Launcher initialization complete");
    });
}

// Function to strip ANSI color codes from a string
fn strip_ansi_codes(input: &str) -> String {
    // Regular expression to match ANSI escape codes
    // This is a simple version that matches the most common color codes
    let mut result = String::with_capacity(input.len());
    let mut in_escape = false;
    
    for c in input.chars() {
        if in_escape {
            // If we're in an escape sequence and we see 'm', it's the end of the sequence
            if c == 'm' {
                in_escape = false;
            }
            // Skip this character as it's part of an escape sequence
            continue;
        } else if c == '\x1b' {
            // Start of an escape sequence
            in_escape = true;
            continue;
        }
        
        // Normal character, add it to the result
        result.push(c);
    }
    
    result
}

// Function to colorize Go log output based on log level
fn colorize_go_log(line: &str) -> String {
    // Check if the line contains a log level indicator
    if let Some(level_start) = line.find(" [") {
        if let Some(level_end) = line[level_start..].find("] ") {
            let level = &line[level_start + 2..level_start + level_end];
            
            // Apply color based on log level
            let color_code = match level {
                "DBG" => "\x1b[36m", // Cyan for Debug
                "INF" => "\x1b[32m", // Green for Info
                "WRN" => "\x1b[33m", // Yellow for Warning
                "ERR" => "\x1b[31m", // Red for Error
                "FTL" => "\x1b[35m", // Magenta for Fatal
                _ => "",             // No color for unknown levels
            };
            
            if !color_code.is_empty() {
                // Format: timestamp [LEVEL] [component] message
                // We want to color just the level part
                let before_level = &line[0..level_start + 1];
                let after_level = &line[level_start + level_end + 1..];
                return format!("{}[{}{}{}\x1b[0m]{}", before_level, color_code, level, "\x1b[0m", after_level);
            }
        }
    }
    
    // Return the original line if no level was found or if the format was unexpected
    line.to_string()
}
