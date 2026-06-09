import { test } from 'node:test';
import assert from 'node:assert/strict';
import jwt from 'jsonwebtoken';

import { createApp } from '../src/app.js';
import { BlacklistStore } from '../src/store.js';
import { normalizeDomain } from '../src/domain.js';

const config = {
  port: 0,
  jwtSecret: 'test-secret-at-least-16-chars',
  jwtIssuer: 'fency-auth',
  allowedOrigins: ['http://localhost:5173'],
};

function token(overrides = {}) {
  return jwt.sign({ username: 'alice', ...overrides }, config.jwtSecret, {
    algorithm: 'HS256',
    issuer: config.jwtIssuer,
    subject: 'u_1',
    expiresIn: '1h',
  });
}

// Spin up the app on an ephemeral port and return a base URL + closer.
async function startServer(store) {
  const app = createApp({ config, store });
  const server = app.listen(0);
  await new Promise((r) => server.once('listening', r));
  const { port } = server.address();
  return {
    url: `http://127.0.0.1:${port}`,
    close: () => new Promise((r) => server.close(r)),
  };
}

test('rejects requests without a token', async () => {
  const s = await startServer(new BlacklistStore());
  const res = await fetch(`${s.url}/blacklist`);
  assert.equal(res.status, 401);
  await s.close();
});

test('rejects a token signed with the wrong secret', async () => {
  const s = await startServer(new BlacklistStore());
  const bad = jwt.sign({ username: 'alice' }, 'totally-different-secret', {
    algorithm: 'HS256',
    issuer: config.jwtIssuer,
  });
  const res = await fetch(`${s.url}/blacklist`, {
    headers: { authorization: `Bearer ${bad}` },
  });
  assert.equal(res.status, 401);
  await s.close();
});

test('full add / list / check / remove lifecycle', async () => {
  const s = await startServer(new BlacklistStore());
  const auth = { authorization: `Bearer ${token()}` };

  // Add (note: URL form should normalise to bare domain).
  let res = await fetch(`${s.url}/blacklist`, {
    method: 'POST',
    headers: { ...auth, 'content-type': 'application/json' },
    body: JSON.stringify({ domain: 'https://www.Evil.com/login' }),
  });
  assert.equal(res.status, 201);
  assert.deepEqual(await res.json(), { domain: 'evil.com', added: true });

  // List.
  res = await fetch(`${s.url}/blacklist`, { headers: auth });
  assert.deepEqual((await res.json()).domains, ['evil.com']);

  // Check subdomain matches.
  res = await fetch(`${s.url}/blacklist/check?domain=mail.evil.com`, { headers: auth });
  assert.deepEqual(await res.json(), { domain: 'mail.evil.com', blacklisted: true });

  // Check unrelated domain.
  res = await fetch(`${s.url}/blacklist/check?domain=good.com`, { headers: auth });
  assert.equal((await res.json()).blacklisted, false);

  // Remove.
  res = await fetch(`${s.url}/blacklist/evil.com`, { method: 'DELETE', headers: auth });
  assert.equal(res.status, 200);

  // Removing again -> 404.
  res = await fetch(`${s.url}/blacklist/evil.com`, { method: 'DELETE', headers: auth });
  assert.equal(res.status, 404);

  await s.close();
});

test('rejects invalid domains', async () => {
  const s = await startServer(new BlacklistStore());
  const auth = { authorization: `Bearer ${token()}` };
  const res = await fetch(`${s.url}/blacklist`, {
    method: 'POST',
    headers: { ...auth, 'content-type': 'application/json' },
    body: JSON.stringify({ domain: 'not a domain!!' }),
  });
  assert.equal(res.status, 400);
  await s.close();
});

test('normalizeDomain unit cases', () => {
  assert.equal(normalizeDomain('https://www.Example.com/x'), 'example.com');
  assert.equal(normalizeDomain('  SUB.Example.CO.UK. '), 'sub.example.co.uk');
  assert.throws(() => normalizeDomain('http://localhost'), /invalid/);
  assert.throws(() => normalizeDomain('1.2.3.4'), /invalid/);
});
