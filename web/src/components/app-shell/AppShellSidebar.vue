<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppShellNavDocuments from '@/components/app-shell/AppShellNavDocuments.vue'
import AppShellNavMain from '@/components/app-shell/AppShellNavMain.vue'
import AppShellNavSecondary from '@/components/app-shell/AppShellNavSecondary.vue'
import AppShellNavUser from '@/components/app-shell/AppShellNavUser.vue'
import { useAppShellNavigation } from '@/components/app-shell/navigation'
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader } from '@/components/ui/sidebar'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const navigation = useAppShellNavigation()

const currentUser = computed(() => ({
  username: authStore.user?.username ?? 'admin',
  email: authStore.user?.email ?? null,
  avatarUrl: authStore.user?.avatarUrl ?? null,
  role: authStore.user?.role ?? 0,
}))

async function handleLogout() {
  authStore.logout()
  await router.push({ name: 'login' })
}
</script>

<template>
  <Sidebar collapsible="offcanvas">
    <SidebarHeader>
      <div class="flex items-center gap-2 px-2">
        <img
          src="/logo.svg"
          class="size-5"
          alt="Logo"
        />
        <span class="text-base font-semibold">web</span>
      </div>
    </SidebarHeader>
    <SidebarContent>
      <AppShellNavMain :items="navigation.main" />
      <AppShellNavDocuments :items="navigation.documents" />
      <AppShellNavSecondary
        :items="navigation.secondary"
        class="mt-auto"
      />
    </SidebarContent>
    <SidebarFooter>
      <AppShellNavUser
        :user="currentUser"
        @logout="handleLogout"
      />
    </SidebarFooter>
  </Sidebar>
</template>
