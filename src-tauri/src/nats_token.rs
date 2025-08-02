use rand::Rng;

/// Generate a secure random token for NATS authentication
pub fn generate_token() -> String {
    // Generate a secure random token
    const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    const TOKEN_LEN: usize = 32;
    
    let mut rng = rand::thread_rng();
    let token: String = (0..TOKEN_LEN)
        .map(|_| {
            let idx = rng.gen_range(0..CHARSET.len());
            CHARSET[idx] as char
        })
        .collect();
    
    token
}
