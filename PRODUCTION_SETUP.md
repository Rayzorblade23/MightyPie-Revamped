# MightyPie-Revamped Production Setup Guide

This document outlines how environment variables and Go binaries are handled in the MightyPie-Revamped application for both development and production environments.

## Environment Variables

### Overview

Environment variables are now **baked into the Rust binary at build time** using the `build.rs` script. This approach:

- Works consistently in both development and production environments
- Eliminates the need to bundle sensitive `.env` files with the production build
- Removes runtime environment variable loading errors

### How It Works

1. During the build process, `build.rs` reads environment variables from:
   - `.env` file in the project root
   - `.env.local` file in the project root (overrides values from `.env`)

2. These variables are passed to the Rust compiler as build-time environment variables using `cargo:rustc-env` directives.

3. In the Rust code, these variables are accessed using the `option_env!` macro, which is evaluated at compile time.

4. The `get_private_env_var` Tauri command first checks for baked-in variables before falling back to runtime environment variables.

### Adding New Environment Variables

When adding new environment variables:

1. Add them to your local `.env` or `.env.local` file
2. Update the `get_baked_env_vars` function in `env_utils.rs` to include the new variable
3. Rebuild the application to bake in the new variable

## Go Executables

### Overview

Go executables are built to `src-tauri/assets/src-go/bin` and bundled with the application in production.

### Configuration

1. Go executables are listed in the `externalBin` array in `tauri.conf.json` to ensure they're properly bundled as executables.
2. The launcher code looks for Go executables in `assets/src-go/bin` in production mode.

### Build Process

1. Go executables are built using the `build.bat` script in the `src-go` directory.
2. The build script outputs the executables to `src-tauri/assets/src-go/bin`.
3. During the Tauri build process, these executables are bundled with the application.

## Best Practices

1. **Never bundle `.env` files in production** - Environment variables should be baked into the binary at build time.
2. **Never hardcode sensitive environment variables in source code** - Use the build-time environment variable approach instead.
3. **Always rebuild the application after changing environment variables** - Changes to `.env` or `.env.local` files require a rebuild to take effect.

## Troubleshooting

If environment variables are not available in production:

1. Ensure the variables are defined in `.env` or `.env.local` during the build process.
2. Verify that the variable is included in the `get_baked_env_vars` function in `env_utils.rs`.
3. Check the build logs for warnings about missing environment variables.
