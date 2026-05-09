<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <main class="flex items-center justify-center bg-background px-4 py-10 min-block-screen sm:px-6">
    <div class="inline-full max-inline-md">
      <RouterView />
    </div>
  </main>
</template>
