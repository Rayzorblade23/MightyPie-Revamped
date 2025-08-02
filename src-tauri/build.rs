use tauri_build;
use std::env;
use std::fs;
use std::path::Path;
use std::collections::HashMap;
use serde_json;

fn main() {
    println!("cargo:rerun-if-changed=.env");
    // We no longer need to watch .env.local since we're not using it anymore
    // println!("cargo:rerun-if-changed=.env.local");

    // Run the standard Tauri build process
    tauri_build::build();

    // Load environment variables from .env files
    let mut env_vars = HashMap::new();
    if let Ok(project_root) = env::var("CARGO_MANIFEST_DIR") {
        let project_root = Path::new(&project_root).parent().unwrap_or_else(|| Path::new(&project_root));

        // Load variables from .env
        let env_path = project_root.join(".env");
        if env_path.exists() {
            println!("cargo:warning=Loading environment variables from {:?}", env_path);
            if let Ok(env_content) = fs::read_to_string(&env_path) {
                for line in env_content.lines() {
                    if !line.starts_with('#') && line.contains('=') {
                        let mut parts = line.splitn(2, '=');
                        if let (Some(key), Some(value)) = (parts.next(), parts.next()) {
                            let key = key.trim().to_string();
                            let mut value = value.trim().to_string();

                            // Remove surrounding quotes if present
                            if (value.starts_with('"') && value.ends_with('"'))
                                || (value.starts_with('\'') && value.ends_with('\''))
                            {
                                value = value[1..value.len() - 1].to_string();
                            }

                            env_vars.insert(key, value);
                        }
                    }
                }
            } else {
                println!("cargo:warning=Failed to read .env file");
            }
        } else {
            println!(
                "cargo:warning=.env file not found at {:?}",
                env_path.display()
            );
        }
    }

    // Serialize all environment variables as JSON and bake them into the binary
    let env_json = serde_json::to_string(&env_vars).unwrap();
    println!("cargo:rustc-env=BAKED_ENV_JSON={}", env_json);

    // Also set individual environment variables as Cargo build-time environment variables
    for (key, value) in env_vars {
        println!("cargo:rustc-env={}={}", key, value);
    }
}
