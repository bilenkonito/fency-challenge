// Thin wrapper around chrome.storage.local for auth session.
// Token is stored in the extension's own storage area for isolation.
import { STORAGE } from './config.js';

export async function getSession() {
    const data = await chrome.storage.local.get([
        STORAGE.token,
        STORAGE.username,
        STORAGE.expiresAt,
    ]);
    const token = data[STORAGE.token];
    const expiresAt = data[STORAGE.expiresAt];

    if (!token) return null;
    if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
        await clearSession();
        return null;
    }
    return { token, username: data[STORAGE.username], expiresAt };
}

export async function setSession({ token, username, expiresAt }) {
    await chrome.storage.local.set({
        [STORAGE.token]: token,
        [STORAGE.username]: username,
        [STORAGE.expiresAt]: expiresAt,
    });
}

export async function clearSession() {
    await chrome.storage.local.remove([
        STORAGE.token,
        STORAGE.username,
        STORAGE.expiresAt,
    ]);
}
