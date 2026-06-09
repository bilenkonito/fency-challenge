# Mini Fency

A small full-stack application that simulates the core behaviour of Fency: a
user logs in, manages a list of risky domains, and a browser extension prevents
access to those domains during navigation.

It is composed of five parts plus container tooling:

| Component           | Tech                                    | Role                                            |
| ------------------- | --------------------------------------- | ----------------------------------------------- |
| `auth-service`      | Go (net/http, JWT, bcrypt)              | User login, issues & validates JWTs             |
| `blacklist-service` | Node.js (Express)                       | CRUD + check API for the domain blacklist       |
| `data-service`      | Go (net/http, ODBC, SQLite by default)  | Internal persistence for users + blacklist      |
| `web-ui`            | Vue 3 (Composition API) + TS + Tailwind | Log in and manage the blacklist                 |
| `extension`         | Chromium Manifest V3                    | Logs in, blocks navigation to blacklisted sites |

All persistence is owned by the **`data-service`**: a private, never-publicly-
exposed Go service that the auth and blacklist services call over an internal
Docker network. Its storage backend is configured purely through an **ODBC**
connection string and defaults to a local **SQLite** database for development.

---

## Architecture

```
   Browser clients (untrusted) ......................................
        ┌──────────────────┐                 ┌─────────────────────┐
        │     Web UI        │                 │  Browser Extension  │
        │ Vue 3 + TS + TW   │                 │  Manifest V3 / DNR  │
        └───────┬──────────┘                 └──────────┬──────────┘
                │   login + manage (Bearer JWT)         │
   ════════════ │ ═══════════ public network ══════════ │ ═══════════
                ▼                                        ▼
        ┌──────────────────┐  shared JWT  ┌────────────────────────┐
        │  auth-service     │  secret(HMAC)│   blacklist-service     │
        │  Go · issues JWT  │◄────────────►│  Node · validates JWT   │
        └────────┬─────────┘              └────────────┬───────────┘
                 │     Bearer API key (DATA_SERVICE_API_KEY)        │
   ══════════════ │ ═════════ private internal network ═══════════ │ ═
                  └───────────────┬───────────────────────────────-┘
                                  ▼
                       ┌────────────────────────┐
                       │      data-service       │   no published ports;
                       │  Go · ODBC · SQLite     │   reachable only by the
                       │  (users + blacklist)    │   DNS name "data-service"
                       └────────────────────────┘
                                  │ ODBC DSN
                                  ▼
                         ┌──────────────────┐
                         │  SQLite (default) │  or any ODBC database
                         └──────────────────┘
```

**Trust boundary & token flow.** The Go auth service is the only component that
verifies user credentials. It signs a JWT (HS256) with a secret that is *shared*
with the blacklist service via the `JWT_SECRET` environment variable. The
blacklist service therefore validates tokens **locally** (no network call back
to auth) by verifying the signature, algorithm (`HS256` only) and issuer. The
web UI and the extension are untrusted clients: they only ever hold a token and
send it as a `Bearer` header.

**Data tier & service-to-service auth.** Neither the auth nor the blacklist
service stores data itself; both delegate persistence to the internal
`data-service`. That service:

- is **never exposed publicly** in `docker-compose.yml` it publishes no ports
  and sits only on an `internal: true` Docker network, so it is reachable solely
  by its Docker DNS name `data-service` from sibling backend services;
- protects **every** endpoint with a **Bearer API key** (`DATA_SERVICE_API_KEY`,
  shared as a secret) compared in constant time;
- is **storage-agnostic via ODBC** the backend is chosen entirely by the
  `ODBC_DSN`, defaulting to a local SQLite file persisted on a Docker volume.

So there are two distinct credentials in play: the **user JWT** (clients →
auth/blacklist) and the **service API key** (auth/blacklist → data-service).

**Why a shared secret (symmetric) instead of asymmetric keys?** For two
first-party services a shared HMAC secret is simple and sufficient. In a larger
system where many independent services validate tokens, asymmetric signing
(RS256/EdDSA, auth holds the private key, others hold the public key) would be
preferable so the secret never leaves the issuer. This is noted under
trade-offs.

### Service layering

- **auth-service** is split into clear layers: `internal/handlers` (HTTP
  transport/routing), `internal/auth` (business logic: credential checks, token
  issue/validate), and `internal/users` (data handling, behind a `Repository`
  interface implemented by `HTTPRepository`).
  `internal/config` and `internal/httputil` hold configuration and shared HTTP
  helpers (JSON + CORS).
- **blacklist-service** separates `app.js` (routes/middleware), `dataClient.js`
  (data-service adapter), `domain.js` (validation/normalisation), `auth.js` (JWT
  middleware) and `config.js`. `store.js` keeps an in-memory store used by tests.
- **data-service** mirrors the same layering: `internal/handlers` (transport),
  `internal/store` (SQL persistence over ODBC), `internal/apikey` (Bearer-key
  middleware), plus `internal/config` and `internal/httputil`.

---

## Prerequisites

- Docker + Docker Compose (for the backends), **or** Go 1.24+ and Node.js 20+ to
  run them directly.
- Node.js 20+ for the web UI.
- A Chromium-based browser (Chrome, Edge, Brave…) for the extension.

---

## 1. Run the backend services (Docker Compose)

```bash
# From the repository root:
cp .env.example .env
# Edit .env and set strong secrets, e.g.:
#   JWT_SECRET=$(openssl rand -hex 32)
#   DATA_SERVICE_API_KEY=$(openssl rand -hex 32)

docker compose up --build
```

This starts:

- `auth-service` on **http://localhost:8080**
- `blacklist-service` on **http://localhost:3000**
- `data-service` **internal only**, no published port (reachable only as
  `http://data-service:9090` from the other services)

Health checks: `curl localhost:8080/health` and `curl localhost:3000/health`.
The blacklist data and the seeded user persist in a SQLite database on the
`data-db` Docker volume.

> The Go services vendor their dependencies, so their images build with no Go
> module network access. The `data-service` image additionally bundles unixODBC
> and the SQLite ODBC driver.

### Run the backends without Docker (optional)

The data-service uses ODBC, so install the unixODBC runtime and the SQLite ODBC
driver first (Debian/Ubuntu): `sudo apt-get install -y unixodbc libsqliteodbc`.

```bash
export JWT_SECRET=a-long-dev-secret-value
export DATA_SERVICE_API_KEY=a-long-dev-api-key-value

# Terminal 1 data-service (SQLite via ODBC)
cd data-service
ODBC_DSN="Driver=SQLite3;Database=$PWD/fency.db;" go run .

# Terminal 2 auth service
cd auth-service
DATA_SERVICE_URL=http://localhost:9090 go run .

# Terminal 3 blacklist service
cd blacklist-service
npm install
DATA_SERVICE_URL=http://localhost:9090 npm start
```

Use the **same** `JWT_SECRET` for auth + blacklist, and the **same**
`DATA_SERVICE_API_KEY` for all three.

---

## 2. Run the web UI

```bash
cd web-ui
npm install
npm run dev
```

Open **http://localhost:5173**. Log in with the test credentials below, then add
and remove domains. The UI talks to the backends at their default URLs; override
with `web-ui/.env` (`VITE_AUTH_URL`, `VITE_BLACKLIST_URL`) if needed (see
`web-ui/.env.example`).

---

## 3. Install and test the browser extension

1. Make sure the backend services are running (step 1).
2. Open `chrome://extensions` in a Chromium browser.
3. Enable **Developer mode** (top-right toggle).
4. Click **Load unpacked** and select the `extension/` directory.
5. Click the Fency icon in the toolbar and **sign in** with the test
   credentials. The badge shows how many domains are currently blocked.
6. Add a domain to your blacklist (via the web UI or the extension stays in
   sync automatically within ~1 minute; use **Refresh now** in the popup to sync
   immediately).
7. Try to visit that domain (e.g. add `example.com`, then navigate to
   `http://example.com`). The extension redirects you to a **"Access blocked"**
   page naming the domain.

> The default backend URLs are in `extension/src/config.js`. If you change the
> backend ports, update them there before loading the extension.

### How blocking works (Manifest V3)

The background **service worker** fetches the blacklist from the service and
converts it into a single **`declarativeNetRequest`** dynamic rule that
redirects matching top-level navigations (and their subdomains) to the
extension's blocked page. This is the MV3-recommended approach: it works even
when the service worker is asleep, and it avoids sending your full browsing
history to the backend (the extension only *pulls* the list, it does not phone
home on every page load). The list re-syncs on login, on browser startup, on a
short alarm interval, and on demand from the popup.

---

## Test credentials

| Username | Password       |
| -------- | -------------- |
| `admin`  | `ChangeMe123!` |

These are seeded by the **data-service** from `SEED_USER` / `SEED_USER_PASSWORD`
(see `.env.example`) on first start, only when the database is empty. The
password is **bcrypt-hashed** before being stored (no plaintext credential is
persisted). Change them for any non-local use.

The blacklist is seeded with `malware-example.com` and `phishing-example.net`
(`SEED_DOMAINS`).

---

## Testing

```bash
# data-service: store + handler tests against real SQLite-over-ODBC, plus
# API-key enforcement. Requires unixODBC + the SQLite ODBC driver (the store
# tests skip automatically if the driver is absent).
cd data-service && CGO_ENABLED=1 go test ./...

# auth-service: auth/business logic and token validation
cd auth-service && go test ./...

# blacklist-service: API lifecycle, auth middleware, domain validation,
# and the data-service client (against a stub server)
cd blacklist-service && npm test

# Web UI type-check + production build
cd web-ui && npm run build
```

CI (`.github/workflows/ci.yml`) runs all of the above on every push and PR.

---

## Security considerations

This is a security product, so the implementation deliberately demonstrates a
number of practices (without aiming for enterprise completeness):

- **Password handling.** Passwords are hashed with **bcrypt** (cost 10) and never
  stored or logged in plaintext. Login returns a generic "invalid username or
  password" and runs a dummy bcrypt comparison for unknown users to reduce
  user-enumeration via timing.
- **Token generation & validation.** Short-lived **HS256 JWTs** with `iss`,
  `sub`, `iat` and `exp` claims. Both services pin the algorithm to `HS256` and
  verify the issuer, which prevents `alg: none` / algorithm-confusion attacks
  and cross-issuer token reuse. Default TTL is 60 minutes.
- **No hardcoded secrets.** Both `JWT_SECRET` and `DATA_SERVICE_API_KEY` have
  **no insecure default** (the services refuse to start without them) and
  require ≥16 chars. Secrets are injected via environment / `.env`, which is
  git-ignored (`.env.example` is the committed template).
- **Internal-only data tier.** The `data-service` is never published to the host
  or internet: it sits on an `internal: true` Docker network with no port
  mapping, reachable only by sibling services via the `data-service` DNS name.
  Every endpoint requires a **Bearer API key** (constant-time compared); the
  database is a private dependency, not a public API.
- **Network segmentation.** Two Docker networks separate the public edge
  (auth/blacklist) from the private data tier; only the two edge services bridge
  both, and only they hold the data-service API key.
- **Input validation.** Domains are normalised to a canonical lowercase
  registrable form (URLs and `www.`/trailing-dot variants are accepted and
  reduced) and validated against a strict hostname pattern; IPs, schemes, paths
  and junk are rejected. JSON bodies are size-limited and unknown fields are
  rejected on login.
- **CORS.** Configured as an explicit allow-list of origins (default the Vite
  dev server) rather than `*`, with only the methods/headers actually used.
- **Trusted vs untrusted data.** Services treat all client input as untrusted
  and validate it; the blacklist middleware attaches only a minimal, verified
  view of the user (`username`, `subject`) to the request. The extension's
  blocked page renders the (untrusted) domain via `textContent`, never
  `innerHTML`, to avoid DOM injection.
- **Extension token safety.** The token lives in the extension's isolated
  `chrome.storage.local` (not reachable by web pages), is only ever sent in the
  `Authorization` header to the configured backends, and is cleared on logout or
  when the backend rejects it (401). Expired sessions are dropped on read.
- **Container hardening.** All images run as **non-root** (distroless `nonroot`
  for the auth service, a dedicated `appuser` for the data-service, the built-in
  `node` user for Node) and ship only what they need.

---

## Main technical decisions

- **Dedicated data-service for persistence.** Instead of each service embedding
  its own database access, a single internal Go service owns all storage. This
  centralises the data tier behind one trust boundary (private network + API
  key), keeps the edge services storage-agnostic, and means the storage backend
  can change in one place.
- **ODBC for storage configurability.** The data-service talks to its database
  through `database/sql` over an ODBC driver, so the backend is selected purely
  by the `ODBC_DSN`. SQLite is the zero-setup default for development; pointing
  the DSN at PostgreSQL/SQL Server/etc. requires no code change. Only portable
  SQL and `?` placeholders are used, and timestamps are generated in Go.
- **Shared HMAC secret** between auth and blacklist so the blacklist service can
  validate tokens offline (fast, no coupling to auth availability).
- **The auth/blacklist data layers stay behind abstractions** (`Repository` in
  Go, the `store` interface in Node), so swapping the in-memory implementations
  for data-service clients touched no routing or business logic.
- **`declarativeNetRequest`** for blocking rather than live per-navigation API
  calls (the recommended MV3 mechanism, which is more resilient).
- **Subdomain matching**: a rule for `evil.com` also blocks `mail.evil.com`.
- **Vendored Go modules** for reproducible, network-free image builds.

---

## Known limitations & trade-offs

- **Persistence is now durable but minimal.** Users and the blacklist live in
  the data-service's database (SQLite by default, persisted on a Docker volume),
  so they survive restarts. It is still a simple schema with no migrations
  framework, no user-management UI, and SQLite's single-writer limit (the pool is
  capped at one connection); a production deployment would use a managed database
  via the ODBC DSN and add proper migrations.
- **ODBC driver dependency.** Running the data-service outside Docker requires
  unixODBC and a driver for the chosen backend installed on the host; the Docker
  image bundles the SQLite driver so `docker compose up` needs nothing extra.
- **Service API key is a shared static secret.** Simple and effective for
  first-party services, but there is no per-service key, rotation, or mTLS;
  those would be the next step for a real internal mesh.
- **Token storage in clients.** Any token held in browser-reachable JS (web UI
  `sessionStorage`) is exposed to XSS. A hardened design would use HttpOnly,
  `Secure`, `SameSite` cookies with CSRF protection for the web app. The
  extension's storage is isolated from web pages, which is safer, but the token
  is still recoverable by someone with local access to the browser's profile
  directory (this has recently become a very prolific attack vector).
- **Symmetric (HS256) tokens.** Fine for two first-party services; asymmetric
  signing (RS256/EdDSA) would be better at larger scale so the secret never
  leaves the issuer. There is also no refresh-token / revocation flow (tokens
  simply expire).
- **No HTTPS / rate limiting in the demo.** Local development uses plain HTTP and
  there is no brute-force throttling on `/login`; both would be required in
  production (TLS termination, rate limits, account lockout).
- **Blacklist sync latency.** The extension refreshes on an interval (~1 min) and
  on demand, so a freshly added domain may take up to that long to block unless
  you hit **Refresh now**.
- **Extension host permission.** A navigation blocker needs broad host access
  (`<all_urls>`) to redirect arbitrary blocked sites; this is inherent to the
  feature and is the same model ad/content blockers use.

