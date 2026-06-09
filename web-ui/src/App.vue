<script setup lang="ts">
import { useAuth } from './composables/useAuth';
import LoginForm from './components/LoginForm.vue';
import BlacklistManager from './components/BlacklistManager.vue';

const { isAuthenticated } = useAuth();
</script>

<template>
    <BlacklistManager v-if="isAuthenticated" />
    <LoginForm v-else />
</template>
