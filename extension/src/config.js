// Backend endpoints.
export const AUTH_URL = 'http://localhost:8080';
export const BLACKLIST_URL = 'http://localhost:3000';

// How often (in minutes) the extension re-syncs the blacklist from the service.
export const SYNC_INTERVAL_MINUTES = 1;

// Keys used in chrome.storage.local.
export const STORAGE = {
    token: 'token',
    username: 'username',
    expiresAt: 'expiresAt',
};
