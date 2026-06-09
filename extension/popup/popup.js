// Popup controller: handles login, logout and shows blocking status.
import { AUTH_URL, BLACKLIST_URL } from '../src/config.js';
import { getSession, setSession, clearSession } from '../src/session.js';

const loginView = document.getElementById('login-view');
const statusView = document.getElementById('status-view');
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');
const loginButton = document.getElementById('login-button');
const statusUsername = document.getElementById('status-username');
const statusCount = document.getElementById('status-count');

function show(view) {
    loginView.classList.toggle('hidden', view !== 'login');
    statusView.classList.toggle('hidden', view !== 'status');
}

function showError(message) {
    loginError.textContent = message;
    loginError.classList.remove('hidden');
}

async function refreshStatus(session) {
    statusUsername.textContent = session.username || '';
    try {
        const res = await fetch(`${BLACKLIST_URL}/blacklist`, {
            headers: { Authorization: `Bearer ${session.token}` },
        });
        if (res.ok) {
            const body = await res.json();
            statusCount.textContent = String(body.domains?.length ?? 0);
        }
    } catch {
        // Leave the previous count so the background worker will retry syncing.
    }
}

async function render() {
    const session = await getSession();
    if (session) {
        show('status');
        await refreshStatus(session);
    } else {
        show('login');
    }
}

loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    loginError.classList.add('hidden');
    loginButton.disabled = true;

    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;

    try {
        const res = await fetch(`${AUTH_URL}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });

        if (!res.ok) {
            let message = 'Sign in failed.';
            try {
                const body = await res.json();
                if (body?.error) message = body.error;
            } catch {
                //
            }
            showError(message);
            return;
        }

        const data = await res.json();
        await setSession({
            token: data.token,
            username: data.username,
            expiresAt: data.expiresAt,
        });
        // Ask the background worker to sync blocking rules immediately.
        chrome.runtime.sendMessage({ type: 'sync' });
        await render();
    } catch {
        showError('Could not reach the authentication service.');
    } finally {
        loginButton.disabled = false;
    }
});

document.getElementById('logout-button').addEventListener('click', async () => {
    await clearSession();
    // Clearing the token triggers the background worker to drop blocking rules.
    await render();
});

document.getElementById('refresh-button').addEventListener('click', async () => {
    await chrome.runtime.sendMessage({ type: 'sync' });
    const session = await getSession();
    if (session) await refreshStatus(session);
});

render();
