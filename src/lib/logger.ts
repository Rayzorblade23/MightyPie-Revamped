// We need to make sure Tauri API is only imported in browser context
// to avoid SSR errors

/**
 * Structured logger for MightyPie frontend
 * Creates consistent log format with categories and uses custom Rust logging
 */

// List of patterns to filter out from console logs
const filterPatterns = [
    'using default settings',
    'connection established on windows',
    'location()',
    'released all held keys'
];

// Global logger instance for App category
let globalAppLogger: Logger | null = null;

// Track if console forwarding is already set up
let consoleForwardingInitialized = false;

/**
 * Create a logger for a specific category
 * @param category The category for the logger
 * @returns A logger instance
 */
export function createLogger(category: string): Logger {
    return new Logger(category);
}

/**
 * Logger class for structured logging with categories
 */
export class Logger {
    private category: string;
    private logLevel: string = 'info'; // Default log level
    private logLevelPromise: Promise<string> | null = null;

    constructor(category: string) {
        this.category = category;
        // Initialize log level from backend
        this.refreshLogLevel();
    }

    /**
     * Fetch the current log level from the backend
     */
    async refreshLogLevel(): Promise<string> {
        if (!this.logLevelPromise) {
            this.logLevelPromise = new Promise<string>(async (resolve) => {
                try {
                    if (typeof window !== 'undefined') {
                        const {invoke} = await import('@tauri-apps/api/core');
                        const level = await invoke<string>('get_log_level');
                        this.logLevel = level.toLowerCase();
                        resolve(this.logLevel);
                    } else {
                        resolve(this.logLevel);
                    }
                } catch (e) {
                    console.error('Failed to fetch log level from backend:', e);
                    resolve(this.logLevel); // Keep using default
                }
            });
        }
        return this.logLevelPromise;
    }

    /**
     * Check if a log level should be displayed based on current log level
     */
    private async shouldLog(level: string): Promise<boolean> {
        await this.refreshLogLevel();

        const levels: Record<string, number> = {
            'error': 0,
            'warn': 1,
            'info': 2,
            'debug': 3,
            'trace': 4,
            'log': 2 // Treat regular log as info level
        };

        const currentLevel = levels[this.logLevel] || 2; // Default to info if unknown
        const messageLevel = levels[level] || 2;

        return messageLevel <= currentLevel;
    }

    log(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('log').then(should => {
            if (should) {
                this.logToBackend('log', message).catch(() => {
                });
            }
        });
    }

    debug(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('debug').then(should => {
            if (should) {
                this.logToBackend('debug', message).catch(() => {
                });
            }
        });
    }

    info(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('info').then(should => {
            if (should) {
                this.logToBackend('info', message).catch(() => {
                });
            }
        });
    }

    warn(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('warn').then(should => {
            if (should) {
                this.logToBackend('warn', message).catch(() => {
                });
            }
        });
    }

    error(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('error').then(should => {
            if (should) {
                this.logToBackend('error', message).catch(() => {
                });
            }
        });
    }

    trace(...args: any[]): void {
        const message = `[${this.category}] ${args.map(arg => String(arg)).join(' ')}`;
        this.shouldLog('trace').then(should => {
            if (should) {
                this.logToBackend('trace', message).catch(() => {
                });
            }
        });
    }

    private async logToBackend(level: string, message: string): Promise<void> {
        try {
            // Only import and use Tauri API in browser context
            if (typeof window !== 'undefined') {
                const {invoke} = await import('@tauri-apps/api/core');
                await invoke('log_from_frontend', {level, message});
            }
        } catch (e) {
            // If we can't send to backend, fall back to console
            console.error('Failed to send log to backend:', e);
        }
    }
}

/**
 * Set up logging for the application
 * Only call this in browser context
 */
export function setupLogging(): void {
    if (typeof window === 'undefined') {
        return; // Only run in browser
    }

    // Create global app logger if it doesn't exist
    if (!globalAppLogger) {
        globalAppLogger = createLogger('App');
    }

    // Only set up console forwarding once
    if (!consoleForwardingInitialized) {
        // Replace console methods with our logger
        replaceConsoleMethods(globalAppLogger);
        consoleForwardingInitialized = true;

        // Send a test log message
        globalAppLogger.info('Frontend logging initialized');
    }
}

/**
 * Replace console methods with our logger
 * @param logger The logger instance to use
 */
function replaceConsoleMethods(logger: Logger): void {
    const methods: Array<'log' | 'debug' | 'info' | 'warn' | 'error'> = [
        'log', 'info', 'warn', 'error'
    ];

    for (const method of methods) {
        // Store original for debugging emergencies
        // const original = console[method];

        console[method] = (...args) => {
            // Check if this message should be filtered
            const messageStr = args.map(arg => String(arg)).join(' ');
            const shouldFilter = filterPatterns.some(pattern => messageStr.includes(pattern));

            if (shouldFilter) {
                // Don't output filtered messages
                return;
            }

            // For direct console access in case of emergency debugging
            // Uncomment the line below if needed during development
            // original(...args);

            // Forward to our logger based on the method
            switch (method) {
                case 'log':
                    logger.log(...args);
                    break;
                case 'info':
                    logger.info(...args);
                    break;
                case 'warn':
                    logger.warn(...args);
                    break;
                case 'error':
                    logger.error(...args);
                    break;
            }
        };
    }
}

// If we're in the browser, set up logging immediately
if (typeof window !== 'undefined') {
    // Wait for DOM to be ready
    if (document.readyState === 'complete' || document.readyState === 'interactive') {
        setupLogging();
    } else {
        document.addEventListener('DOMContentLoaded', () => {
            setupLogging();
        });
    }
}
