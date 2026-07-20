<script setup lang="ts">
import { computed } from 'vue'
import type { HTMLAttributes } from 'vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { getAvatarFallbackText } from '@/lib/avatar'

const props = defineProps<{
  username: string
  avatarUrl: string | null
  class?: HTMLAttributes['class']
  fallbackClass?: HTMLAttributes['class']
}>()

const fallbackText = computed(() => getAvatarFallbackText(props.username))
</script>

<template>
  <Avatar :class="props.class">
    <AvatarImage
      v-if="props.avatarUrl"
      :src="props.avatarUrl"
      :alt="props.username"
    />
    <AvatarFallback :class="props.fallbackClass">{{
      fallbackText
    }}</AvatarFallback>
  </Avatar>
</template>
