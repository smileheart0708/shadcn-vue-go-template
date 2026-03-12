<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-background px-4 py-10 sm:px-6">
    <div class="w-full max-w-md">
      <RouterView />
    </div>
  </main>
</template>
