use std::net::{SocketAddr, TcpListener};
use std::str::FromStr;

/// Check if a port is available on the specified host
pub fn is_port_available(host: &str, port: u16) -> bool {
    let addr = format!("{}:{}", host, port);
    match SocketAddr::from_str(&addr) {
        Ok(socket_addr) => {
            match TcpListener::bind(socket_addr) {
                Ok(_) => true, // Port is available
                Err(_) => false, // Port is in use
            }
        }
        Err(_) => false, // Invalid address format
    }
}

/// Find an available port starting from the given port
pub fn find_available_port(host: &str, start_port: u16) -> Option<u16> {
    // Try up to 100 ports starting from start_port
    for port in start_port..start_port + 100 {
        if is_port_available(host, port) {
            return Some(port);
        }
    }
    None
}
