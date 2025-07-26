import {spawn} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';
import dotenv from 'dotenv';
import chalk from 'chalk';

// --- Configuration ---
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');

// Load environment variables from .env and .env.local
dotenv.config({ path: path.join(projectRoot, '.env') });
dotenv.config({ path: path.join(projectRoot, '.env.local') });

const commands = {
    go: {
        cmd: path.join(projectRoot, 'src-go', 'bin', 'main-x86_64-pc-windows-msvc.exe'),
        args: [],
        cwd: path.join(projectRoot, 'src-go'),
        color: chalk.blue,
        env: { ...process.env, APP_ENV: 'development' }
    }, vite: {
        cmd: 'node',
        args: [path.join(projectRoot, 'node_modules', 'vite', 'bin', 'vite.js'), 'dev'],
        cwd: projectRoot,
        color: chalk.magenta,
    },
};

const children = [];

// --- Main Execution ---
async function main() {
    try {
        console.log(chalk.yellow('[LAUNCHER] Starting services...'));
        spawnProcess('go', commands.go);
        spawnProcess('vite', commands.vite);

    } catch (err) {
        console.error(chalk.red.bold('[LAUNCHER] A critical error occurred:'), err);
        shutdown();
    }
}

// --- Helper Functions ---
function spawnProcess(name, {cmd, args, cwd, color, env}) {
    const p = spawn(cmd, args, {cwd, stdio: 'pipe', env});
    children.push(p);

    const prefix = color(`[${name.toUpperCase()}]`);

    const handleData = (stream, data) => {
        const lines = data.toString().split(/\r?\n/);
        const rest = lines.pop(); // The last element is either an empty string or an incomplete line

        for (const line of lines) {
            stream.write(`${prefix} ${line}\n`);
        }

        // If there's an incomplete line, keep it in a buffer (not implemented here for simplicity,
        // but for robustness, you'd handle this)
        if (rest) {
            stream.write(`${prefix} ${rest}`);
        }
    };

    p.stdout.on('data', (data) => handleData(process.stdout, data));
    p.stderr.on('data', (data) => handleData(process.stderr, data));
    p.on('close', (code) => console.log(`${prefix} exited with code ${code}.`));
    p.on('error', (err) => console.error(`${prefix} ${chalk.red.bold('Error:')}`, err));

    return p;
}

function shutdown() {
    console.log(chalk.yellow('\n[LAUNCHER] Shutdown signal received. Terminating all processes...'));
    for (const child of children) {
        child.kill();
    }
    process.exit();
}

// --- Signal Handling ---
process.on('SIGINT', shutdown); // Ctrl+C
process.on('SIGTERM', shutdown); // kill

main().catch((err) => console.error(chalk.red.bold('[LAUNCHER] A critical error occurred:'), err));