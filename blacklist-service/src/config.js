// Runtime configuration loaded from environment.

function requireEnv(name, { minLength = 0 } = {}) {
    const value = process.env[name];
    if (!value) {
        throw new Error(`Environment variable ${name} is required`);
    }
    if (value.length < minLength) {
        throw new Error(`Environment variable ${name} must be at least ${minLength} characters`);
    }
    return value;
}

function splitOrigins(raw, fallback) {
    if (!raw) return fallback;
    return raw
        .split(',')
        .map((o) => o.trim())
        .filter(Boolean);
}

export function loadConfig(env = process.env) {
    return {
        port: Number(env.PORT) || 3000,
        jwtSecret: requireEnv('JWT_SECRET', { minLength: 16 }),
        // Must match the Go auth service's JWT_ISSUER.
        jwtIssuer: env.JWT_ISSUER || 'fency-auth',
        allowedOrigins: splitOrigins(env.ALLOWED_ORIGINS, [
            'http://localhost:5173',
        ]),
        dataServiceUrl: env.DATA_SERVICE_URL || 'http://data-service:9090',
        dataServiceApiKey: requireEnv('DATA_SERVICE_API_KEY', { minLength: 16 }),
    };
}
