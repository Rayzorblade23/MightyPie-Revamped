use std::fs::File;
use std::io::{self, BufRead, BufReader};
use std::path::Path;

/// Represents the NATS configuration
pub struct NatsConfig {
    pub websocket_host: String,
    pub websocket_port: u16,
    pub listen_host: String,
    pub listen_port: u16,
}

impl Default for NatsConfig {
    fn default() -> Self {
        NatsConfig {
            websocket_host: "127.0.0.1".to_string(),
            websocket_port: 9090,
            listen_host: "127.0.0.1".to_string(),
            listen_port: 4222,
        }
    }
}

/// Parse a NATS configuration file
pub fn parse_nats_config<P: AsRef<Path>>(path: P) -> io::Result<NatsConfig> {
    let file = File::open(path)?;
    let reader = BufReader::new(file);
    
    let mut config = NatsConfig::default();
    let mut in_websocket_block = false;
    
    for line in reader.lines() {
        let line = line?;
        let trimmed = line.trim();
        
        // Skip comments and empty lines
        if trimmed.starts_with('#') || trimmed.is_empty() {
            continue;
        }
        
        // Check for websocket block
        if trimmed == "websocket {" {
            in_websocket_block = true;
            continue;
        }
        
        if trimmed == "}" && in_websocket_block {
            in_websocket_block = false;
            continue;
        }
        
        // Parse key-value pairs
        if let Some((key, value)) = parse_key_value(trimmed) {
            if in_websocket_block {
                match key.as_str() {
                    "host" => config.websocket_host = value,
                    "port" => {
                        if let Ok(port) = value.parse() {
                            config.websocket_port = port;
                        }
                    }
                    _ => {}
                }
            } else if key.as_str() == "listen" {
                // Parse listen address (format: host:port)
                if let Some((host, port)) = parse_host_port(&value) {
                    config.listen_host = host;
                    if let Ok(port_num) = port.parse() {
                        config.listen_port = port_num;
                    }
                }
            }
        }
    }
    
    Ok(config)
}

/// Parse a key-value pair from a line
fn parse_key_value(line: &str) -> Option<(String, String)> {
    let parts: Vec<&str> = line.splitn(2, ':').collect();
    if parts.len() == 2 {
        let key = parts[0].trim().to_string();
        let value = parts[1].trim().trim_matches(|c| c == '"' || c == '\'').to_string();
        Some((key, value))
    } else {
        let parts: Vec<&str> = line.splitn(2, '=').collect();
        if parts.len() == 2 {
            let key = parts[0].trim().to_string();
            let value = parts[1].trim().trim_matches(|c| c == '"' || c == '\'').to_string();
            Some((key, value))
        } else {
            None
        }
    }
}

/// Parse a host:port string
fn parse_host_port(addr: &str) -> Option<(String, String)> {
    let parts: Vec<&str> = addr.split(':').collect();
    if parts.len() == 2 {
        Some((parts[0].to_string(), parts[1].to_string()))
    } else {
        None
    }
}

/// Update a NATS configuration file with new port values
pub fn update_nats_config<P: AsRef<Path>>(path: P, websocket_port: u16, listen_port: u16) -> io::Result<()> {
    let config_content = std::fs::read_to_string(&path)?;
    
    // Replace the websocket port
    let websocket_port_regex = regex::Regex::new(r"websocket\s*\{[^}]*port\s*:\s*(\d+)").unwrap();
    let updated_content = websocket_port_regex.replace(&config_content, |caps: &regex::Captures| {
        let original = caps.get(0).unwrap().as_str();
        let port_part = caps.get(1).unwrap().as_str();
        original.replace(port_part, &websocket_port.to_string())
    });
    
    // Replace the listen port
    let listen_regex = regex::Regex::new(r"listen\s*:\s*([^:]+):(\d+)").unwrap();
    let final_content = listen_regex.replace(&updated_content, |caps: &regex::Captures| {
        let host = caps.get(1).unwrap().as_str();
        format!("listen: {}:{}", host, listen_port)
    });
    
    std::fs::write(path, final_content.as_ref())?;
    Ok(())
}
