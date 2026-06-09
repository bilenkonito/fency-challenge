// Express application wiring: middleware, routes and error handling.
import express from 'express';
import cors from 'cors';
import helmet from 'helmet';

import { requireAuth } from './auth.js';
import { BlacklistStore } from './store.js';
import { normalizeDomain, InvalidDomainError } from './domain.js';

/**
 * @param {object} options
 * @param {ReturnType<import('./config.js').loadConfig>} options.config
 * @param {BlacklistStore} [options.store]
 * @returns {import('express').Express}
 */
export function createApp({ config, store = new BlacklistStore() }) {
    const app = express();

    // Security headers.
    app.use(helmet());

    // Restrictive CORS.
    app.use(
        cors({
            origin: config.allowedOrigins,
            methods: ['GET', 'POST', 'DELETE', 'OPTIONS'],
            allowedHeaders: ['Content-Type', 'Authorization'],
            maxAge: 600,
        }),
    );

    // Small JSON bodies only.
    app.use(express.json({ limit: '8kb' }));

    // Health check (unauthenticated).
    app.get('/health', (_req, res) => res.json({ status: 'ok' }));

    const auth = requireAuth(config);

    // Everything under /blacklist requires a valid token.
    const router = express.Router();
    router.use(auth);

    // Validates + normalises a domain, replying 400 on failure.
    // Returns the canonical domain, or null when a response has already been sent.
    function parseDomain(raw, res) {
        try {
            return normalizeDomain(raw);
        } catch (err) {
            if (err instanceof InvalidDomainError) {
                res.status(400).json({ error: err.message });
                return null;
            }
            throw err;
        }
    }

    // Retrieve the current blacklist.
    router.get('/', async (_req, res, next) => {
        try {
            res.json({ domains: await store.list() });
        } catch (err) {
            next(err);
        }
    });

    // Check whether a given domain is blacklisted.
    // GET /blacklist/check?domain=example.com
    router.get('/check', async (req, res, next) => {
        const domain = parseDomain(req.query.domain, res);
        if (domain === null) return;
        try {
            res.json({ domain, blacklisted: await store.isBlacklisted(domain) });
        } catch (err) {
            next(err);
        }
    });

    // Add a domain to the blacklist.
    router.post('/', async (req, res, next) => {
        const domain = parseDomain(req.body?.domain, res);
        if (domain === null) return;
        try {
            const added = await store.add(domain);
            res.status(added ? 201 : 200).json({ domain, added });
        } catch (err) {
            next(err);
        }
    });

    // Remove a domain from the blacklist.
    router.delete('/:domain', async (req, res, next) => {
        const domain = parseDomain(req.params.domain, res);
        if (domain === null) return;
        try {
            const removed = await store.remove(domain);
            if (!removed) {
                return res.status(404).json({ error: 'domain not found', domain });
            }
            res.json({ domain, removed });
        } catch (err) {
            next(err);
        }
    });

    app.use('/blacklist', router);

    // 404 for anything else.
    app.use((_req, res) => res.status(404).json({ error: 'not found' }));

    // Centralised error handler so unexpected failures never leak internals.
    // eslint-disable-next-line no-unused-vars
    app.use((err, _req, res, _next) => {
        if (err?.type === 'entity.parse.failed' || err instanceof SyntaxError) {
            return res.status(400).json({ error: 'invalid JSON body' });
        }
        console.error('unhandled error:', err);
        res.status(500).json({ error: 'internal server error' });
    });

    return app;
}
