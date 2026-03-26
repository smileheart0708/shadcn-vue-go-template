import { onBeforeUnmount, onMounted, watch } from 'vue'
import { acquirePollingTask, type PollingTaskHandle, type PollingTaskOptions } from '@/lib/polling/manager'

interface UsePollingTaskOptions<TData> extends PollingTaskOptions<TData> {
  autoStart?: boolean
}

export function usePollingTask<TData>(options: UsePollingTaskOptions<TData>): PollingTaskHandle {
  const task = acquirePollingTask(options)
  const autoStart = options.autoStart ?? true

  if (typeof options.enabled === 'function') {
    watch(
      () => options.enabled?.() ?? true,
      (enabled) => {
        if (!task.running.value) {
          return
        }

        if (enabled) {
          task.resume()
          return
        }

        task.pause()
      },
    )
  }

  onMounted(() => {
    if (!autoStart) {
      return
    }

    task.start()
  })

  onBeforeUnmount(() => {
    task.release()
  })

  return task
}
