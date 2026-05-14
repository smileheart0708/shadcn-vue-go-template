import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { listAuditLogs, type AuditLogEntry } from '@/lib/api/system-logs'
import { getAuditTotalPages } from '@/utils/system-logs/audit'

interface UseAuditLogsOptions {
  activeTab: Ref<'console' | 'audit'>
  canReadAuditLogs: ComputedRef<boolean>
}

export function useAuditLogs(options: UseAuditLogsOptions) {
  const { t } = useI18n()

  const entries = ref<AuditLogEntry[]>([])
  const loading = ref(true)
  const refreshing = ref(false)
  const loadFailed = ref(false)
  const page = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  const totalPages = computed(() => getAuditTotalPages(total.value, pageSize.value))

  watch(
    () => options.activeTab.value,
    (tab) => {
      if (tab === 'audit' && options.canReadAuditLogs.value && entries.value.length === 0 && !loadFailed.value) {
        void loadAuditLogs()
      }
    },
    { immediate: true },
  )

  async function loadAuditLogs(loadOptions: { background?: boolean } = {}) {
    if (!options.canReadAuditLogs.value) {
      return
    }

    if (loadOptions.background === true) {
      refreshing.value = true
    } else {
      loading.value = true
    }

    try {
      const response = await listAuditLogs(page.value, pageSize.value)
      entries.value = response.items
      page.value = response.page
      pageSize.value = response.pageSize
      total.value = response.total
      loadFailed.value = false
    } catch (error) {
      loadFailed.value = true
      toast.error(getAPIErrorMessage(t, error, 'systemLogs.audit.feedback.loadFailed'))
    } finally {
      loading.value = false
      refreshing.value = false
    }
  }

  function goToPreviousPage() {
    if (page.value <= 1 || refreshing.value) {
      return
    }

    page.value -= 1
    void loadAuditLogs({ background: true })
  }

  function goToNextPage() {
    if (page.value >= totalPages.value || refreshing.value) {
      return
    }

    page.value += 1
    void loadAuditLogs({ background: true })
  }

  return {
    entries,
    loading,
    refreshing,
    loadFailed,
    page,
    total,
    totalPages,
    loadAuditLogs,
    goToPreviousPage,
    goToNextPage,
  }
}
