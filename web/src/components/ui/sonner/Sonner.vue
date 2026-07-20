<script lang="ts" setup>
import type { ToasterProps } from 'vue-sonner'
import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
  XIcon,
} from '@lucide/vue'
import { Toaster as Sonner } from 'vue-sonner'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<ToasterProps>(), {
  closeButton: true,
  closeButtonPosition: 'top-right',
  position: 'top-center',
  richColors: true,
})
</script>

<template>
  <Sonner
    :class="cn('group', props.class)"
    :style="{
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': 'var(--radius)',
    }"
    v-bind="props"
  >
    <template #success-icon>
      <CircleCheckIcon class="block-4 inline-4" />
    </template>
    <template #info-icon>
      <InfoIcon class="block-4 inline-4" />
    </template>
    <template #warning-icon>
      <TriangleAlertIcon class="block-4 inline-4" />
    </template>
    <template #error-icon>
      <OctagonXIcon class="block-4 inline-4" />
    </template>
    <template #loading-icon>
      <div>
        <Loader2Icon class="animate-spin block-4 inline-4" />
      </div>
    </template>
    <template #close-icon>
      <XIcon class="block-4 inline-4" />
    </template>
  </Sonner>
</template>
