<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { ArrowRightIcon } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-background text-foreground">
    <div class="flex max-w-5xl flex-col items-center px-6 text-center">
      <a
        href="#"
        class="inline-flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
      >
        <span class="size-2 rounded-full bg-[#22c55e]" />
        <span>New Components: Field, Input Group, Item and more</span>
        <ArrowRightIcon class="size-4" />
      </a>

      <h1 class="mt-4 max-w-4xl text-balance text-[2rem] font-semibold tracking-[-0.04em] text-foreground sm:text-[2.5rem] lg:text-[3rem]">
        The Foundation for your Design System
      </h1>

      <p class="mt-3 max-w-3xl text-balance text-sm leading-relaxed text-muted-foreground sm:text-base sm:leading-relaxed">
        A set of beautifully designed components that you can customize, extend, and build on.
        Start here then make it your own. Open Source. Open Code.
      </p>

      <div class="mt-5 flex flex-wrap items-center justify-center gap-4">
        <Button
          as-child
          size="lg"
          class="h-10 rounded-xl px-5 text-sm"
        >
          <RouterLink to="/dashboard">Get Started</RouterLink>
        </Button>
        <Button
          as-child
          variant="ghost"
          size="lg"
          class="h-10 rounded-xl px-4 text-sm"
        >
          <a href="#components">View Components</a>
        </Button>
      </div>
    </div>
  </main>
</template>
