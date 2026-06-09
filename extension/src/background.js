// Background service worker (MV3).
import { BLACKLIST_URL, SYNC_INTERVAL_MINUTES, STORAGE } from './config.js';
import { getSession, clearSession } from './session.js';

// A single dynamic rule covers every blocked domain.
// It redirects top-level navigations whose host is a subdomain of a blocked domain to the blocked page,
// passing the host so the page can show which domain was blocked.
const RULE_ID = 1;

const BLOCKED_PAGE = chrome.runtime.getURL('blocked/blocked.html');

/** Fetch the current blacklist from the service using the stored token. */
async function fetchBlacklist(token) {
    const res = await fetch(`${BLACKLIST_URL}/blacklist`, {
        headers: { Authorization: `Bearer ${token}` },
    });
    if (res.status === 401) {
        // Token rejected, clear it so the popup prompts for login again.
        await clearSession();
        return null;
    }
    if (!res.ok) {
        throw new Error(`blacklist fetch failed: ${res.status}`);
    }
    const body = await res.json();
    return Array.isArray(body.domains) ? body.domains : [];
}

/** Replace the dynamic DNR rules to match the supplied domains. */
async function applyRules(domains) {
    const existing = await chrome.declarativeNetRequest.getDynamicRules();
    const removeRuleIds = existing.map((r) => r.id);

    const addRules =
        domains.length === 0
            ? []
            : [
                {
                    id: RULE_ID,
                    priority: 1,
                    action: {
                        type: 'redirect',
                        // \\1 = captured host from regexFilter below.
                        redirect: { regexSubstitution: `${BLOCKED_PAGE}?domain=\\1` },
                    },
                    condition: {
                        // Capture the host portion of the URL.
                        regexFilter: '^https?://([^/?#]+)',
                        // Restrict to the blocked domains and their subdomains.
                        requestDomains: domains,
                        resourceTypes: ['main_frame'],
                    },
                },
            ];

    await chrome.declarativeNetRequest.updateDynamicRules({ removeRuleIds, addRules });
}

/** Full sync: read session, fetch list, apply rules. Safe to call often. */
async function sync() {
    const session = await getSession();
    if (!session) {
        // Not logged in: remove all blocking rules.
        await applyRules([]);
        await setBadge(false);
        return;
    }
    try {
        const domains = await fetchBlacklist(session.token);
        if (domains === null) {
            await applyRules([]);
            await setBadge(false);
            return;
        }
        await applyRules(domains);
        await setBadge(true, domains.length);
    } catch (err) {
        console.warn('blacklist sync failed:', err);
    }
}

async function setBadge(active, count = 0) {
    if (active) {
        await chrome.action.setBadgeBackgroundColor({ color: '#4f46e5' });
        await chrome.action.setBadgeText({ text: count > 0 ? String(count) : '' });
    } else {
        await chrome.action.setBadgeText({ text: '' });
    }
}

// Re-sync when the stored session changes.
chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && STORAGE.token in changes) {
        sync();
    }
});

// Periodic refresh so web-UI changes reach the extension.
chrome.runtime.onInstalled.addListener(() => {
    chrome.alarms.create('sync', { periodInMinutes: SYNC_INTERVAL_MINUTES });
    sync();
});

chrome.runtime.onStartup.addListener(() => {
    sync();
});

chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'sync') sync();
});

// Allow the popup to request an immediate sync.
chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type === 'sync') {
        sync().then(() => sendResponse({ ok: true }));
        return true; // keep the message channel open for the async response
    }
    return false;
});
