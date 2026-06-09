// Authentication state shared across the app via a module-level reactive store.
//
// The token is kept in sessionStorage rather than localStorage.
// It is cleared when the tab closes, which slightly limits exposure.

import { reactive, computed, readonly } from 'vue';
import { login as apiLogin, type LoginResponse } from '../api/client';

const STORAGE_KEY = 'fency.session';

interface Session {
    token: string;
    username: string;
    expiresAt: string;
}

function loadSession(): Session | null {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    try {
        const s = JSON.parse(raw) as Session;
        // Drop expired sessions on load.
        if (new Date(s.expiresAt).getTime() <= Date.now()) return null;
        return s;
    } catch {
        return null;
    }
}

const state = reactive<{ session: Session | null; }>({
    session: loadSession(),
});

export function useAuth() {
    const isAuthenticated = computed(() => state.session !== null);
    const username = computed(() => state.session?.username ?? '');

    async function login(user: string, password: string): Promise<void> {
        const res: LoginResponse = await apiLogin(user, password);
        const session: Session = {
            token: res.token,
            username: res.username,
            expiresAt: res.expiresAt,
        };
        state.session = session;
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    }

    function logout(): void {
        state.session = null;
        sessionStorage.removeItem(STORAGE_KEY);
    }

    function token(): string {
        return state.session?.token ?? '';
    }

    return {
        isAuthenticated,
        username,
        session: readonly(state),
        login,
        logout,
        token,
    };
}
