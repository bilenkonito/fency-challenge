import { test } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';

import { DataClient } from '../src/dataClient.js';

const API_KEY = 'data-service-test-key-16chars';

// Minimal stub of the data-service that records requests and returns canned responses.
async function withStub(handler, fn) {
    const seen = [];
    const server = http.createServer((req, res) => {
        let body = '';
        req.on('data', (c) => (body += c));
        req.on('end', () => {
            seen.push({ method: req.method, url: req.url, auth: req.headers.authorization, body });
            handler(req, res, body);
        });
    });
    server.listen(0);
    await new Promise((r) => server.once('listening', r));
    const { port } = server.address();
    const client = new DataClient({ baseUrl: `http://127.0.0.1:${port}`, apiKey: API_KEY });
    try {
        await fn(client, seen);
    } finally {
        await new Promise((r) => server.close(r));
    }
}

test('sends the API key as a Bearer token', async () => {
    await withStub(
        (_req, res) => {
            res.writeHead(200, { 'content-type': 'application/json' });
            res.end(JSON.stringify({ domains: [] }));
        },
        async (client, seen) => {
            await client.list();
            assert.equal(seen[0].auth, `Bearer ${API_KEY}`);
        },
    );
});

test('add maps 201 -> true and 200 -> false', async () => {
    let status = 201;
    await withStub(
        (_req, res) => {
            res.writeHead(status, { 'content-type': 'application/json' });
            res.end(JSON.stringify({ added: status === 201 }));
        },
        async (client) => {
            assert.equal(await client.add('evil.com'), true);
            status = 200;
            assert.equal(await client.add('evil.com'), false);
        },
    );
});

test('remove maps 200 -> true and 404 -> false', async () => {
    let status = 200;
    await withStub(
        (_req, res) => {
            res.writeHead(status, { 'content-type': 'application/json' });
            res.end(JSON.stringify({ removed: status === 200 }));
        },
        async (client) => {
            assert.equal(await client.remove('evil.com'), true);
            status = 404;
            assert.equal(await client.remove('evil.com'), false);
        },
    );
});

test('list and isBlacklisted parse the response body', async () => {
    await withStub(
        (req, res) => {
            res.writeHead(200, { 'content-type': 'application/json' });
            if (req.url.startsWith('/blacklist/check')) {
                res.end(JSON.stringify({ blacklisted: true }));
            } else {
                res.end(JSON.stringify({ domains: ['a.com', 'b.com'] }));
            }
        },
        async (client) => {
            assert.deepEqual(await client.list(), ['a.com', 'b.com']);
            assert.equal(await client.isBlacklisted('x.com'), true);
        },
    );
});

test('throws on unexpected status', async () => {
    await withStub(
        (_req, res) => {
            res.writeHead(500);
            res.end('boom');
        },
        async (client) => {
            await assert.rejects(() => client.list(), /list failed: 500/);
        },
    );
});
