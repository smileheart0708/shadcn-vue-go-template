<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { IconUnlink } from '@tabler/icons-vue'

const { t } = useI18n()
const router = useRouter()

function handleBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }

  void router.push({ name: 'dashboard' })
}
</script>

<template>
  <div class="flex h-[calc(100vh-200px)] items-center justify-center p-4">
    <Empty class="max-w-md">
      <EmptyMedia>
        <IconUnlink class="size-16 text-muted-foreground" />
      </EmptyMedia>
      <EmptyHeader>
        <EmptyTitle>404</EmptyTitle>
        <EmptyDescription> {{ t('notFound.description') }} </EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button @click="handleBack"> {{ t('common.action.back') }} </Button>
      </EmptyContent>
    </Empty>
  </div>
</template>
