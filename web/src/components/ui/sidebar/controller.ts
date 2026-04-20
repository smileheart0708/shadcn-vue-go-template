import { defaultDocument, useMediaQuery } from '@vueuse/core'
import { computed, getCurrentInstance, ref, watch, type ComputedRef, type Ref } from 'vue'
import { SIDEBAR_COOKIE_MAX_AGE, SIDEBAR_COOKIE_NAME } from './utils'

export interface SidebarControllerProps {
  defaultOpen?: boolean | null
  open?: boolean | null
}

export interface SidebarController {
  state: ComputedRef<'expanded' | 'collapsed'>
  open: Ref<boolean>
  setOpen: (value: boolean) => void
  isMobile: Ref<boolean>
  openMobile: Ref<boolean>
  setOpenMobile: (value: boolean) => void
  toggleSidebar: () => void
}

export function useSidebarController(
  props: SidebarControllerProps,
  emitUpdateOpen: (value: boolean) => void,
): SidebarController {
  const isMobile = useMediaQuery('(max-width: 768px)')
  const openMobile = ref(false)
  const uncontrolledOpen = ref(readStoredSidebarOpen())
  const instance = getCurrentInstance()
  // Vue normalizes omitted Boolean props, so we read vnode props once here to
  // distinguish an omitted prop from an explicit `false`.
  const isOpenControlled = hasPassedProp(instance, 'open')
  const hasDefaultOpenProp = hasPassedProp(instance, 'defaultOpen')

  watch(
    () => props.defaultOpen,
    (defaultOpen) => {
      if (hasDefaultOpenProp && defaultOpen !== null && defaultOpen !== undefined) {
        uncontrolledOpen.value = defaultOpen
      }
    },
    { immediate: true, once: true },
  )

  const open = computed({
    get: () => (isOpenControlled ? props.open ?? uncontrolledOpen.value : uncontrolledOpen.value),
    set: (value: boolean) => {
      if (!isOpenControlled) {
        uncontrolledOpen.value = value
      }

      emitUpdateOpen(value)
    },
  })

  watch(
    open,
    (value) => {
      persistStoredSidebarOpen(value)
    },
    { immediate: true },
  )

  function setOpen(value: boolean) {
    open.value = value
  }

  function setOpenMobile(value: boolean) {
    openMobile.value = value
  }

  function toggleSidebar() {
    if (isMobile.value) {
      setOpenMobile(!openMobile.value)
      return
    }

    setOpen(!open.value)
  }

  const state = computed(() => (open.value ? 'expanded' : 'collapsed'))

  return { state, open, setOpen, isMobile, openMobile, setOpenMobile, toggleSidebar }
}

function hasPassedProp(instance: ReturnType<typeof getCurrentInstance>, name: 'open' | 'defaultOpen') {
  return Boolean(instance?.vnode.props && Object.prototype.hasOwnProperty.call(instance.vnode.props, name))
}

function persistStoredSidebarOpen(value: boolean) {
  if (defaultDocument === undefined) {
    return
  }

  const cookieValue = value ? 'true' : 'false'
  defaultDocument.cookie = `${SIDEBAR_COOKIE_NAME}=${cookieValue}; path=/; max-age=${String(SIDEBAR_COOKIE_MAX_AGE)}`
}

function readStoredSidebarOpen(): boolean {
  if (defaultDocument === undefined) {
    return true
  }

  return !defaultDocument.cookie.includes(`${SIDEBAR_COOKIE_NAME}=false`)
}
