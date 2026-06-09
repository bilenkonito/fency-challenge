<script setup lang="ts">
import { ref } from 'vue';
import { useAuth } from '../composables/useAuth';
import { ApiError } from '../api/client';

const { login } = useAuth();

const username = ref('');
const password = ref('');
const error = ref('');
const loading = ref(false);

async function onSubmit() {
    error.value = '';
    loading.value = true;
    try {
        await login(username.value, password.value);
    } catch (e) {
        error.value = e instanceof ApiError ? e.message : 'Unable to sign in. Please try again.';
    } finally {
        loading.value = false;
    }
}
</script>

<template>
    <div class="flex min-h-screen items-center justify-center bg-slate-100 p-4">
        <form class="w-full max-w-sm space-y-5 rounded-xl bg-white p-8 shadow-md" @submit.prevent="onSubmit">
            <div class="text-center">
                <h1 class="text-2xl font-bold text-slate-800">Fency</h1>
                <p class="text-sm text-slate-500">Sign in to manage blocked domains</p>
            </div>

            <div class="space-y-1">
                <label for="username" class="block text-sm font-medium text-slate-700">Username</label>
                <input id="username" v-model="username" type="text" autocomplete="username" required
                    class="w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
            </div>

            <div class="space-y-1">
                <label for="password" class="block text-sm font-medium text-slate-700">Password</label>
                <input id="password" v-model="password" type="password" autocomplete="current-password" required
                    class="w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
            </div>

            <p v-if="error" class="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">
                {{ error }}
            </p>

            <button type="submit" :disabled="loading"
                class="w-full rounded-md bg-indigo-600 px-4 py-2 font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-60">
                {{ loading ? 'Signing in…' : 'Sign in' }}
            </button>
        </form>
    </div>
</template>
