// Typed API client for the auth and blacklist backends.

const AUTH_URL = import.meta.env.VITE_AUTH_URL ?? 'http://localhost:8080';
const BLACKLIST_URL = import.meta.env.VITE_BLACKLIST_URL ?? 'http://localhost:3000';

export interface LoginResponse {
    token: string;
    expiresAt: string;
    username: string;
}

/** Error carrying the HTTP status so callers can react to it if needed. */
export class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
    }
}

async function parseError(res: Response): Promise<never> {
    let message = `request failed (${res.status})`;
    try {
        const body = await res.json();
        if (body?.error) message = body.error;
    } catch {
        // Non-JSON body; keep the default message.
    }
    throw new ApiError(message, res.status);
}

export async function login(username: string, password: string): Promise<LoginResponse> {
    const res = await fetch(`${AUTH_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
    });
    if (!res.ok) return parseError(res);
    return res.json();
}

function authHeaders(token: string): HeadersInit {
    return { Authorization: `Bearer ${token}` };
}

export async function getBlacklist(token: string): Promise<string[]> {
    const res = await fetch(`${BLACKLIST_URL}/blacklist`, {
        headers: authHeaders(token),
    });
    if (!res.ok) return parseError(res);
    const body = await res.json();
    return body.domains as string[];
}

export async function addDomain(token: string, domain: string): Promise<void> {
    const res = await fetch(`${BLACKLIST_URL}/blacklist`, {
        method: 'POST',
        headers: { ...authHeaders(token), 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain }),
    });
    if (!res.ok) return parseError(res);
}

export async function removeDomain(token: string, domain: string): Promise<void> {
    const res = await fetch(`${BLACKLIST_URL}/blacklist/${encodeURIComponent(domain)}`, {
        method: 'DELETE',
        headers: authHeaders(token),
    });
    if (!res.ok) return parseError(res);
}
