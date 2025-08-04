use chrono;
use std::cell::RefCell;
use std::env;
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Mutex, OnceLock};

// Define a struct for log entries
#[derive(Clone, serde::Serialize)]
pub struct LogEntry {
    pub timestamp: String,
    pub level: String,
    pub message: String,
}

// Define the circular log buffer
pub struct CircularLogBuffer {
    buffer: RefCell<Vec<Option<LogEntry>>>,
    head: AtomicUsize,
    capacity: usize,
    pub log_file_path: PathBuf,
    max_log_size: usize,
    max_log_files: usize,
}

impl CircularLogBuffer {
    pub fn new(capacity: usize, log_dir: PathBuf) -> Self {
        let mut buffer = Vec::with_capacity(capacity);
        // Initialize with None values
        for _ in 0..capacity {
            buffer.push(None);
        }

        // Create logs directory if it doesn't exist
        if !log_dir.exists() {
            if let Err(e) = fs::create_dir_all(&log_dir) {
                eprintln!("Failed to create log directory: {}", e);
            }
        }

        let log_file_path = log_dir.join("mightypie.log");

        // Get max log size and max log files from environment or use defaults
        let max_log_size = match env::var("LOG_MAX_SIZE_KB") {
            Ok(val) => val.parse::<usize>().unwrap_or(1024) * 1024, // Default 1MB
            Err(_) => 1024 * 1024,                                  // 1MB
        };

        let max_log_files = match env::var("LOG_MAX_FILES") {
            Ok(val) => val.parse().unwrap_or(5),
            Err(_) => 5,
        };

        CircularLogBuffer {
            buffer: RefCell::new(buffer),
            head: AtomicUsize::new(0),
            capacity,
            log_file_path,
            max_log_size,
            max_log_files,
        }
    }

    pub fn add(&self, entry: LogEntry) {
        let current_head = self.head.load(Ordering::Relaxed);
        let next_head = (current_head + 1) % self.capacity;

        // Store the log entry in memory buffer using RefCell for interior mutability
        self.buffer.borrow_mut()[current_head] = Some(entry.clone());

        // Update the head position
        self.head.store(next_head, Ordering::Relaxed);

        // Write to log file
        self.write_to_file(&entry);
    }

    fn write_to_file(&self, entry: &LogEntry) {
        // Check if log file exists and its size
        let file_exists = self.log_file_path.exists();
        let file_size = if file_exists {
            fs::metadata(&self.log_file_path)
                .map(|m| m.len() as usize)
                .unwrap_or(0)
        } else {
            0
        };

        // If file size exceeds max, rotate logs
        if file_exists && file_size > self.max_log_size {
            self.rotate_logs();
        }

        // Open file in append mode or create if it doesn't exist
        let mut file = match OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.log_file_path)
        {
            Ok(file) => file,
            Err(e) => {
                eprintln!("Failed to open log file: {}", e);
                return;
            }
        };

        // Convert log level to abbreviated format to match console output
        let abbreviated_level = match entry.level.to_lowercase().as_str() {
            "error" => "ERR",
            "warn" => "WRN",
            "info" => "INF",
            "debug" => "DBG",
            _ => "LOG",
        };

        // Write log entry to file - use the same format as console output
        let log_line = format!(
            "[SVELTE] {} [{}] {}\n",
            entry.timestamp, abbreviated_level, entry.message
        );
        if let Err(e) = file.write_all(log_line.as_bytes()) {
            eprintln!("Failed to write to log file: {}", e);
        }
    }

    fn rotate_logs(&self) {
        // Rename existing log files
        for i in (1..self.max_log_files).rev() {
            let src = self
                .log_file_path
                .parent()
                .unwrap()
                .join(format!("mightypie_{}.log", i));
            let dst = self
                .log_file_path
                .parent()
                .unwrap()
                .join(format!("mightypie_{}.log", i + 1));

            if src.exists() {
                if let Err(e) = fs::rename(&src, &dst) {
                    eprintln!("Failed to rotate log file {}: {}", i, e);
                }
            }
        }

        // Rename current log file
        let backup = self.log_file_path.parent().unwrap().join("mightypie_1.log");
        if let Err(e) = fs::rename(&self.log_file_path, &backup) {
            eprintln!("Failed to rotate current log file: {}", e);
        }
    }

    pub fn get_logs(&self) -> Vec<LogEntry> {
        let current_head = self.head.load(Ordering::Relaxed);
        let mut logs = Vec::new();

        // Collect logs in chronological order
        for i in 0..self.capacity {
            let idx = (current_head + i) % self.capacity;
            if let Some(entry) = &self.buffer.borrow()[idx] {
                logs.push(entry.clone());
            }
        }

        logs
    }

    pub fn get_log_dir(&self) -> PathBuf {
        self.log_file_path
            .parent()
            .unwrap_or(Path::new("."))
            .to_path_buf()
    }
}

// Global static circular log buffer
static LOG_BUFFER: OnceLock<Mutex<CircularLogBuffer>> = OnceLock::new();

// Helper to get or initialize the log buffer
pub fn get_log_buffer() -> &'static Mutex<CircularLogBuffer> {
    LOG_BUFFER.get_or_init(|| {
        // Default capacity of 1000 log entries
        let capacity = match env::var("LOG_BUFFER_CAPACITY") {
            Ok(val) => val.parse().unwrap_or(1000),
            Err(_) => 1000,
        };

        // Get app name from environment variable
        let app_name =
            env::var("PUBLIC_APPNAME").unwrap_or_else(|_| "MightyPieRevamped".to_string());

        // Use AppData\Local for logs in all environments
        let app_data_dir = {
            let local_app_data = env::var("LOCALAPPDATA").unwrap_or_else(|_| {
                // Fallback to APPDATA if LOCALAPPDATA is not available
                env::var("APPDATA").unwrap_or_else(|_| ".".to_string())
            });
            PathBuf::from(local_app_data).join(app_name)
        };
        let log_dir = app_data_dir.join("logs");

        Mutex::new(CircularLogBuffer::new(capacity, log_dir))
    })
}

// Helper function to log directly to file
pub fn log_to_file(message: &str) {
    // Get app name from environment variable
    let app_name = env::var("PUBLIC_APPNAME").unwrap_or_else(|_| "MightyPieRevamped".to_string());

    // Use AppData\Local for logs in all environments
    let local_app_data = env::var("LOCALAPPDATA").unwrap_or_else(|_| {
        // Fallback to APPDATA if LOCALAPPDATA is not available
        env::var("APPDATA").unwrap_or_else(|_| ".".to_string())
    });
    let app_data_dir = PathBuf::from(local_app_data).join(app_name);
    let log_dir = app_data_dir.join("logs");

    // Create directory if it doesn't exist
    std::fs::create_dir_all(&log_dir).unwrap_or_else(|_| {});

    // Open log file
    let log_file_path = log_dir.join("mightypie.log");
    let mut file = std::fs::OpenOptions::new()
        .create(true)
        .append(true)
        .open(log_file_path)
        .unwrap_or_else(|_| panic!("Failed to open log file"));

    // Write message
    writeln!(file, "{}", message).unwrap_or_else(|_| {});
}

// Command to log from frontend
#[tauri::command]
pub fn log_from_frontend(level: &str, message: &str) {
    // Use a more sophisticated approach for deduplication
    use std::collections::HashMap;
    use std::sync::Mutex;
    use std::sync::OnceLock;
    
    // Store message counts by level+message
    static MESSAGE_COUNTS: OnceLock<Mutex<HashMap<String, u32>>> = OnceLock::new();
    let message_counts = MESSAGE_COUNTS.get_or_init(|| Mutex::new(HashMap::new()));
    
    // Store the last message sequence
    static LAST_MESSAGE_KEY: OnceLock<Mutex<String>> = OnceLock::new();
    let last_message_key = LAST_MESSAGE_KEY.get_or_init(|| Mutex::new(String::new()));
    
    // Create a unique key for this level+message
    let message_key = format!("{}:{}", level, message);
    
    // Try to lock the mutexes
    if let (Ok(mut counts), Ok(mut last_key)) = (message_counts.lock(), last_message_key.lock()) {
        // Check if this is the same as the last message
        if message_key == *last_key {
            // Increment the count for this message
            let count = counts.entry(message_key.clone()).or_insert(0);
            *count += 1;
            
            // Don't output anything yet, just count it
            return;
        } else {
            // Different message than before
            
            // First, check if we need to output the previous message's count
            if !last_key.is_empty() {
                if let Some(count) = counts.get(&*last_key) {
                    if *count > 0 {
                        // Split the key back into level and message
                        if let Some((prev_level, prev_message)) = last_key.split_once(':') {
                            // Output the previous message with count
                            output_svelte_log(prev_level, prev_message, *count + 1);
                            
                            // Also add to circular buffer with repeat count
                            let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S").to_string();
                            let entry = LogEntry {
                                timestamp: timestamp.clone(),
                                level: prev_level.to_string(),
                                message: format!("{} (repeated {} times)", prev_message, count + 1),
                            };
                            
                            if let Ok(buffer) = get_log_buffer().lock() {
                                buffer.add(entry);
                            }
                        }
                    }
                }
                
                // Remove the previous message from counts
                counts.remove(&*last_key);
            }
            
            // Update the last message key
            *last_key = message_key.clone();
            
            // Reset the count for this new message
            counts.insert(message_key, 0);
            
            // Output this message normally
            output_svelte_log(level, message, 0);
            
            // Add to circular buffer
            let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S").to_string();
            let entry = LogEntry {
                timestamp: timestamp.clone(),
                level: level.to_string(),
                message: message.to_string(),
            };
            
            if let Ok(buffer) = get_log_buffer().lock() {
                buffer.add(entry);
            }
        }
    }
}

// Helper function to output Svelte logs with consistent formatting
fn output_svelte_log(level: &str, message: &str, repeat_count: u32) {
    let timestamp = chrono::Local::now().format("%Y/%m/%d %H:%M:%S").to_string();
    
    // Format the repeat count suffix if provided
    let repeat_suffix = if repeat_count > 1 {
        format!(" (repeated {} times)", repeat_count)
    } else {
        String::new()
    };
    
    match level {
        "error" => eprintln!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [\x1b[31mERR\x1b[0m] {}{}",
            timestamp, message, repeat_suffix
        ),
        "warn" => eprintln!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [\x1b[33mWRN\x1b[0m] {}{}",
            timestamp, message, repeat_suffix
        ),
        "info" => eprintln!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [\x1b[32mINF\x1b[0m] {}{}",
            timestamp, message, repeat_suffix
        ),
        "debug" => println!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [\x1b[36mDBG\x1b[0m] {}{}",
            timestamp, message, repeat_suffix
        ),
        "trace" => println!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [\x1b[35mTRC\x1b[0m] {}{}",
            timestamp, message, repeat_suffix
        ),
        _ => println!(
            "\x1b[38;5;180m[SVELTE]\x1b[0m {} [LOG] {}{}",
            timestamp, message, repeat_suffix
        ),
    }
}

// New command to retrieve logs
#[tauri::command]
pub fn get_logs() -> Vec<LogEntry> {
    let log_buffer = get_log_buffer();

    match log_buffer.lock() {
        Ok(buffer) => buffer.get_logs(),
        Err(_) => Vec::new(),
    }
}

// Get log file location
#[tauri::command]
pub fn get_log_file_path() -> String {
    let log_buffer = get_log_buffer();

    match log_buffer.lock() {
        Ok(buffer) => buffer.log_file_path.to_string_lossy().to_string(),
        Err(_) => "Unable to access log file path".to_string(),
    }
}

// Get log directory
#[tauri::command]
pub fn get_log_dir() -> String {
    let log_buffer = get_log_buffer();

    match log_buffer.lock() {
        Ok(buffer) => buffer.get_log_dir().to_string_lossy().to_string(),
        Err(_) => "Unable to access log directory".to_string(),
    }
}

// Get the current log level from RUST_LOG environment variable
#[tauri::command]
pub fn get_log_level() -> String {
    std::env::var("RUST_LOG").unwrap_or_else(|_| "info".to_string())
}
