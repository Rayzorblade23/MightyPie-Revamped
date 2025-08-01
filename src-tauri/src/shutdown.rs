use lazy_static::lazy_static;
use log::{error, info};
use std::process::{Child, Command};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;

// Global flag to track if shutdown has been initiated
static SHUTDOWN_INITIATED: AtomicBool = AtomicBool::new(false);

// Stores child processes that need to be terminated on shutdown
// Using Mutex for thread-safe access instead of unsafe static mut
lazy_static! {
    static ref CHILD_PROCESSES: Mutex<Vec<u32>> = Mutex::new(Vec::new());
}

/// Register a process to be terminated on application shutdown
pub fn register_process(child: &Child) {
    if let Ok(mut processes) = CHILD_PROCESSES.lock() {
        processes.push(child.id());
        info!(
            "Registered child process with PID {} for shutdown management",
            child.id()
        );
    } else {
        error!("Failed to acquire lock for process registration");
    }
}

/// Initialize the shutdown handler
pub fn init() {
    info!("Initializing shutdown handler");

    // Create a thread that periodically checks if shutdown has been initiated
    thread::spawn(|| loop {
        if SHUTDOWN_INITIATED.load(Ordering::SeqCst) {
            info!("Shutdown initiated, terminating child processes");
            terminate_all_processes();
            break;
        }
        thread::sleep(Duration::from_millis(100));
    });

    // Register a shutdown handler for normal termination
    let original_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |panic_info| {
        info!("Application panic detected, initiating shutdown");
        SHUTDOWN_INITIATED.store(true, Ordering::SeqCst);
        terminate_all_processes();
        original_hook(panic_info);
    }));
}

/// Manually trigger shutdown (can be called from anywhere)
#[allow(dead_code)]
pub fn trigger_shutdown() {
    info!("Manual shutdown triggered");
    SHUTDOWN_INITIATED.store(true, Ordering::SeqCst);
}

/// Terminate all registered child processes
fn terminate_all_processes() {
    // Safely access child processes with mutex
    if let Ok(mut processes) = CHILD_PROCESSES.lock() {
        for pid in processes.iter() {
            info!("Terminating process with PID: {}", pid);

            // First try graceful termination
            #[cfg(windows)]
            {
                // On Windows, send CTRL+C signal
                let _ = Command::new("taskkill")
                    .args(&["/PID", &pid.to_string(), "/T"])
                    .output();
            }

            #[cfg(unix)]
            {
                // On Unix, send SIGTERM using kill command
                let _ = Command::new("kill")
                    .args(&["-15", &pid.to_string()])
                    .output();
            }

            // Give the process a moment to shut down
            thread::sleep(Duration::from_millis(100));

            // Force kill if still running
            #[cfg(windows)]
            {
                let _ = Command::new("taskkill")
                    .args(&["/F", "/PID", &pid.to_string(), "/T"])
                    .output();
            }

            #[cfg(unix)]
            {
                let _ = Command::new("kill")
                    .args(&["-9", &pid.to_string()])
                    .output();
            }
        }

        // Clear the list
        processes.clear();
    } else {
        error!("Failed to acquire lock for process termination");
    }

    info!("All child processes terminated");
}
