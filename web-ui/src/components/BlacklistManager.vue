<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useAuth } from '../composables/useAuth';
import { getBlacklist, addDomain, removeDomain, ApiError } from '../api/client';

const { username, logout, token } = useAuth();

const domains = ref<string[]>([]);
const newDomain = ref('');
const loading = ref(false);
const busy = ref(false);
const error = ref('');
const success = ref('');

function flash(kind: 'error' | 'success', message: string) {
    if (kind === 'error') {
        error.value = message;
        success.value = '';
    } else {
        success.value = message;
        error.value = '';
    }
}

function handle(e: unknown, fallback: string) {
    // A 401 means the session is no longer valid — send the user back to login.
    if (e instanceof ApiError && e.status === 401) {
        logout();
        return;
    }
    flash('error', e instanceof ApiError ? e.message : fallback);
}

async function refresh() {
    loading.value = true;
    error.value = '';
    try {
        domains.value = await getBlacklist(token());
    } catch (e) {
        handle(e, 'Failed to load the blacklist.');
    } finally {
        loading.value = false;
    }
}

async function onAdd() {
    const value = newDomain.value.trim();
    if (!value) return;
    busy.value = true;
    try {
        await addDomain(token(), value);
        newDomain.value = '';
        flash('success', `Added ${value}`);
        await refresh();
    } catch (e) {
        handle(e, 'Failed to add the domain.');
    } finally {
        busy.value = false;
    }
}

async function onRemove(domain: string) {
    busy.value = true;
    try {
        await removeDomain(token(), domain);
        flash('success', `Removed ${domain}`);
        await refresh();
    } catch (e) {
        handle(e, 'Failed to remove the domain.');
    } finally {
        busy.value = false;
    }
}

onMounted(refresh);
</script>

<template>
    <div class="min-h-screen bg-slate-100">
        <header class="border-b border-slate-200 bg-white">
            <div class="mx-auto flex max-w-2xl items-center justify-between px-4 py-4">
                <div>
                    <h1 class="text-lg font-bold text-slate-800">Fency — Blacklist</h1>
                    <p class="text-xs text-slate-500">Signed in as {{ username }}</p>
                </div>
                <button class="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
                    @click="logout">
                    Sign out
                </button>
            </div>
        </header>

        <main class="mx-auto max-w-2xl space-y-6 p-4">
            <form class="flex gap-2" @submit.prevent="onAdd">
                <input v-model="newDomain" type="text" placeholder="example.com" aria-label="Domain to block"
                    class="flex-1 rounded-md border border-slate-300 px-3 py-2 text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
                <button type="submit" :disabled="busy"
                    class="rounded-md bg-indigo-600 px-4 py-2 font-medium text-white hover:bg-indigo-700 disabled:opacity-60">
                    Add
                </button>
            </form>

            <p v-if="error" class="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">
                {{ error }}
            </p>
            <p v-if="success" class="rounded-md bg-green-50 px-3 py-2 text-sm text-green-700" role="status">
                {{ success }}
            </p>

            <section class="rounded-xl bg-white shadow-sm">
                <div v-if="loading" class="p-6 text-center text-slate-500">Loading…</div>
                <div v-else-if="domains.length === 0" class="p-6 text-center text-slate-500">
                    No blocked domains yet. Add one above.
                </div>
                <ul v-else class="divide-y divide-slate-100">
                    <li v-for="domain in domains" :key="domain" class="flex items-center justify-between px-4 py-3">
                        <span class="font-mono text-sm text-slate-800">{{ domain }}</span>
                        <button :disabled="busy"
                            class="rounded-md px-2 py-1 text-sm text-red-600 hover:bg-red-50 disabled:opacity-60"
                            @click="onRemove(domain)">
                            Remove
                        </button>
                    </li>
                </ul>
            </section>
        </main>
    </div>
</template>
