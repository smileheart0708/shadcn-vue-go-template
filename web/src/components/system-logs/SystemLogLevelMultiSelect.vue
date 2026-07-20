<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SYSTEM_LOG_LEVEL_VALUES,
  type SystemLogLevel,
} from '@/lib/api/system-logs'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: readonly SystemLogLevel[]
    class?: HTMLAttributes['class']
  }>(),
  {
    modelValue: () => [...SYSTEM_LOG_LEVEL_VALUES],
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: SystemLogLevel[]]
}>()

const { t } = useI18n()

const selectedLevels = computed(() => new Set(props.modelValue))
const orderedSelectedLevels = computed(() =>
  SYSTEM_LOG_LEVEL_VALUES.filter((level) => selectedLevels.value.has(level)),
)
const triggerLabel = computed(() => {
  if (orderedSelectedLevels.value.length === SYSTEM_LOG_LEVEL_VALUES.length) {
    return t('systemLogs.filters.level.allSelected')
  }
  if (orderedSelectedLevels.value.length === 0) {
    return t('systemLogs.filters.level.noneSelected')
  }
  return orderedSelectedLevels.value
    .map((level) => t(`systemLogs.filters.level.${level}`))
    .join(', ')
})

function updateLevel(level: SystemLogLevel, selected: boolean) {
  const nextLevels = new Set(props.modelValue)
  if (selected) {
    nextLevels.add(level)
  } else {
    nextLevels.delete(level)
  }

  emit(
    'update:modelValue',
    SYSTEM_LOG_LEVEL_VALUES.filter((candidate) => nextLevels.has(candidate)),
  )
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        variant="outline"
        :class="
          cn('justify-between gap-2 inline-full md:inline-auto', props.class)
        "
      >
        <span class="truncate">{{ triggerLabel }}</span>
        <ChevronDown class="shrink-0 opacity-60 block-4 inline-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      align="end"
      class="inline-auto"
    >
      <DropdownMenuCheckboxItem
        v-for="level in SYSTEM_LOG_LEVEL_VALUES"
        :key="level"
        :model-value="selectedLevels.has(level)"
        @select.prevent
        @update:model-value="updateLevel(level, $event === true)"
      >
        {{ t(`systemLogs.filters.level.${level}`) }}
      </DropdownMenuCheckboxItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
