import { readonly, ref } from 'vue'
import type { Router } from 'vue-router'

const SHOW_DELAY_MS = 120
const MIN_VISIBLE_MS = 80

const routeLoadingVisible = ref(false)
const routeNavigationPending = ref(false)
const requestLoadingCount = ref(0)
const routeLoadingErrorRevision = ref(0)
const navigationIds = new WeakMap<object, number>()

let installed = false
let requestTrackingInstalled = false
let activeNavigationId = 0
let visibleSince = 0
let showTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

function hasPendingWork() {
  return routeNavigationPending.value || requestLoadingCount.value > 0
}

function clearShowTimer() {
  if (showTimer === null) {
    return
  }

  clearTimeout(showTimer)
  showTimer = null
}

function clearHideTimer() {
  if (hideTimer === null) {
    return
  }

  clearTimeout(hideTimer)
  hideTimer = null
}

function showRouteLoadingIfPending() {
  showTimer = null

  if (!hasPendingWork()) {
    return
  }

  routeLoadingVisible.value = true
  visibleSince = Date.now()
}

function hideRouteLoadingIfIdle() {
  hideTimer = null

  if (hasPendingWork()) {
    return
  }

  routeLoadingVisible.value = false
  visibleSince = 0
}

function beginNavigation() {
  activeNavigationId += 1
  const navigationId = activeNavigationId

  routeNavigationPending.value = true
  syncLoadingState()

  return navigationId
}

function finishNavigation(navigationId: number) {
  if (navigationId !== activeNavigationId) {
    return
  }

  routeNavigationPending.value = false
  syncLoadingState()
}

function syncLoadingState() {
  if (hasPendingWork()) {
    clearHideTimer()

    if (routeLoadingVisible.value || showTimer !== null) {
      return
    }

    showTimer = setTimeout(showRouteLoadingIfPending, SHOW_DELAY_MS)
    return
  }

  clearShowTimer()

  if (!routeLoadingVisible.value) {
    return
  }

  const visibleForMs = visibleSince > 0 ? Date.now() - visibleSince : 0
  const remainingVisibleMs = Math.max(0, MIN_VISIBLE_MS - visibleForMs)

  clearHideTimer()
  hideTimer = setTimeout(hideRouteLoadingIfIdle, remainingVisibleMs)
}

function incrementRequestLoading() {
  requestLoadingCount.value += 1
  syncLoadingState()
}

function decrementRequestLoading() {
  requestLoadingCount.value = Math.max(0, requestLoadingCount.value - 1)
  syncLoadingState()
}

function signalRouteLoadingError() {
  routeLoadingErrorRevision.value += 1
}

function shouldTrackRequest(input: RequestInfo | URL) {
  try {
    const locationOrigin = typeof globalThis.location === 'undefined' ? 'http://localhost' : globalThis.location.origin
    const locationHref = typeof globalThis.location === 'undefined' ? `${locationOrigin}/` : globalThis.location.href
    const requestInput = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const requestUrl = typeof Request !== 'undefined' && input instanceof Request ? new URL(input.url) : new URL(requestInput, locationHref)
    return requestUrl.origin === locationOrigin && requestUrl.pathname.startsWith('/api/')
  } catch {
    return false
  }
}

export function installRouteLoading(router: Router) {
  if (installed) {
    return
  }

  installed = true

  router.beforeEach((to) => {
    const navigationId = beginNavigation()
    navigationIds.set(to, navigationId)
    return true
  })

  router.afterEach((to) => {
    const navigationId = navigationIds.get(to)
    if (navigationId === undefined) {
      return
    }

    navigationIds.delete(to)
    finishNavigation(navigationId)
  })

  router.onError((_, to) => {
    const navigationId = navigationIds.get(to)
    if (navigationId === undefined) {
      return
    }

    navigationIds.delete(to)

    if (navigationId === activeNavigationId) {
      finishNavigation(navigationId)
      signalRouteLoadingError()
    }
  })
}

export function installRequestLoadingTracking() {
  if (requestTrackingInstalled || typeof globalThis.fetch !== 'function') {
    return
  }

  requestTrackingInstalled = true

  const originalFetch = globalThis.fetch.bind(globalThis)

  const wrappedFetch: typeof globalThis.fetch = async (...args) => {
    if (!shouldTrackRequest(args[0])) {
      return originalFetch(...args)
    }

    incrementRequestLoading()

    try {
      return await originalFetch(...args)
    } finally {
      decrementRequestLoading()
    }
  }

  globalThis.fetch = wrappedFetch
}

export function useRouteLoadingState() {
  return readonly(routeLoadingVisible)
}

export function useRouteLoadingErrorRevision() {
  return readonly(routeLoadingErrorRevision)
}
