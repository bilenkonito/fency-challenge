// Entry point: load config, wire the data-service client and start the server.
import { loadConfig } from './config.js';
import { createApp } from './app.js';
import { DataClient } from './dataClient.js';

function main() {
    let config;
    try {
        config = loadConfig();
    } catch (err) {
        console.error(`configuration error: ${err.message}`);
        process.exit(1);
    }

    // Persistence (and seeding) live in the internal data-service.
    const store = new DataClient({
        baseUrl: config.dataServiceUrl,
        apiKey: config.dataServiceApiKey,
    });
    const app = createApp({ config, store });

    const server = app.listen(config.port, () => {
        console.log(`blacklist-service listening on :${config.port}`);
    });

    const shutdown = (signal) => {
        console.log(`received ${signal}, shutting down`);
        server.close(() => process.exit(0));
    };
    process.on('SIGINT', () => shutdown('SIGINT'));
    process.on('SIGTERM', () => shutdown('SIGTERM'));
}

main();
