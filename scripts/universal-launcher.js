import {spawn} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';
import chalk from 'chalk';
import fs from 'fs';

// --- Configuration ---
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');

// Determine environment from variables provided by Rust
// No need to load .env files as Rust has already loaded them
const isProd = process.env.APP_ENV === 'production';
console.log(chalk.yellow(`[LAUNCHER] Running in ${isProd ? 'production' : 'development'} mode`));

// Is this launcher being started from Tauri's Rust code?
const isStartedFromTauri = process.env.LAUNCHER_STARTED_FROM_TAURI === 'true';
if (isStartedFromTauri) {
    console.log(chalk.yellow(`[LAUNCHER] Started from Tauri Rust code`));
    
    // Check if parent process still exists periodically
    // If the parent process (Tauri) exits, we should exit too to clean up our child processes
    const parentPid = process.ppid;
    console.log(chalk.yellow(`[LAUNCHER] Parent process ID: ${parentPid}`));
    
    // Set up a periodic check for parent process
    const checkParentInterval = setInterval(() => {
        try {
            // On Windows, trying to signal process 0 throws an error
            process.kill(parentPid, 0);
            // If we get here, the parent is still running
        } catch (e) {
            // Error means the parent process no longer exists
            console.log(chalk.red(`[LAUNCHER] Parent process ${parentPid} no longer exists, shutting down...`));
            clearInterval(checkParentInterval);
            shutdown();
        }
    }, 1000); // Check every second
}

// In production, the Go executable is in the same directory as the current executable
// In development, it's in src-go/bin
let goExecutablePath;
let goCwd;

if (isProd) {
    // In production, the executable is bundled with Tauri
    goExecutablePath = path.join(process.env.TAURI_RESOURCE_DIR || '.', 'main');
    goCwd = process.env.TAURI_RESOURCE_DIR || '.';
    
    // On Windows, add .exe extension
    if (process.platform === 'win32') {
        goExecutablePath += '.exe';
    }
} else {
    // In development
    goExecutablePath = path.join(projectRoot, 'src-go', 'bin', 'main-x86_64-pc-windows-msvc.exe');
    goCwd = path.join(projectRoot, 'src-go');
}

// Ensure the executable exists
if (!fs.existsSync(goExecutablePath)) {
    console.error(chalk.red(`[LAUNCHER] Go executable not found at ${goExecutablePath}`));
    process.exit(1);
}

const commands = {
    go: {
        cmd: goExecutablePath,
        args: [],
        cwd: goCwd,
        color: chalk.blue,
        env: { 
            ...process.env,
            // APP_ENV is already set by Rust
        }
    }
};

// Only include Vite in production mode, or in development mode if not started from Tauri
// (because Tauri will start Vite separately via beforeDevCommand)
if (isProd || !isStartedFromTauri) {
    commands.vite = {
        cmd: 'node',
        args: [path.join(projectRoot, 'node_modules', 'vite', 'bin', 'vite.js'), isProd ? 'preview' : 'dev'],
        cwd: projectRoot,
        color: chalk.magenta,
    };
}

const children = [];

// --- Main Execution ---
async function main() {
    try {
        console.log(chalk.yellow('[LAUNCHER] Starting services...'));
        spawnProcess('go', commands.go);
        
        // Start Vite if applicable
        if (commands.vite) {
            console.log(chalk.yellow(`[LAUNCHER] Starting Vite in ${isProd ? 'production' : 'development'} mode`));
            spawnProcess('vite', commands.vite);
        } else {
            console.log(chalk.yellow(`[LAUNCHER] Skipping Vite start - will be started by Tauri`));
        }
    } catch (err) {
        console.error(chalk.red.bold('[LAUNCHER] A critical error occurred:'), err);
        shutdown();
    }
}

// --- Helper Functions ---
function spawnProcess(name, {cmd, args, cwd, color, env}) {
    console.log(chalk.yellow(`[LAUNCHER] Starting ${name} process: ${cmd} ${args.join(' ')}`));
    
    const p = spawn(cmd, args, {cwd, stdio: 'pipe', env, detached: false});
    children.push(p);

    const prefix = color(`[${name.toUpperCase()}]`);

    const handleData = (stream, data) => {
        const lines = data.toString().split(/\r?\n/);
        const rest = lines.pop(); // The last element is either an empty string or an incomplete line

        for (const line of lines) {
            stream.write(`${prefix} ${line}\n`);
        }

        // If there's an incomplete line, keep it in a buffer
        if (rest) {
            stream.write(`${prefix} ${rest}`);
        }
    };

    p.stdout.on('data', (data) => handleData(process.stdout, data));
    p.stderr.on('data', (data) => handleData(process.stderr, data));
    p.on('close', (code) => {
        console.log(`${prefix} exited with code ${code}.`);
        
        // If a child process exits, log it but don't exit the launcher
        // This allows for potential restart logic in the future
    });
    
    p.on('error', (err) => {
        console.error(`${prefix} failed to start:`, err);
    });
    
    return p;
}

function shutdown() {
    console.log(chalk.yellow('\n[LAUNCHER] Shutting down all services...'));
    
    for (const child of children) {
        if (!child.killed) {
            console.log(chalk.yellow(`[LAUNCHER] Killing process ${child.pid}...`));
            
            // Use process group killing on non-Windows platforms
            if (process.platform !== 'win32' && child.pid) {
                try {
                    process.kill(-child.pid);
                } catch (e) {
                    console.error(chalk.red(`[LAUNCHER] Failed to kill process group ${-child.pid}: ${e.message}`));
                    try {
                        child.kill();
                    } catch (e2) {
                        console.error(chalk.red(`[LAUNCHER] Failed to kill process ${child.pid}: ${e2.message}`));
                    }
                }
            } else {
                try {
                    // On Windows, use child.kill()
                    child.kill('SIGTERM');
                } catch (e) {
                    console.error(chalk.red(`[LAUNCHER] Failed to kill process ${child.pid}: ${e.message}`));
                }
            }
        }
    }
    
    console.log(chalk.yellow('[LAUNCHER] Shutdown complete'));
    process.exit(0);
}

// --- Signal Handling ---
process.on('SIGINT', shutdown); // Ctrl+C
process.on('SIGTERM', shutdown); // kill
process.on('disconnect', () => {
    console.log(chalk.yellow('[LAUNCHER] Parent process disconnected, shutting down...'));
    shutdown();
});

// Start everything
main();
