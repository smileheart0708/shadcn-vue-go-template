<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppShellNavDocuments from '@/components/app-shell/AppShellNavDocuments.vue'
import AppShellNavMain from '@/components/app-shell/AppShellNavMain.vue'
import AppShellNavSecondary from '@/components/app-shell/AppShellNavSecondary.vue'
import AppShellNavUser from '@/components/app-shell/AppShellNavUser.vue'
import { useAppShellNavigation } from '@/components/app-shell/navigation'
import { useAuth } from '@/composables/auth/useAuth'
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader } from '@/components/ui/sidebar'

const router = useRouter()
const auth = useAuth()
const navigation = useAppShellNavigation()

const currentUser = computed(() => ({
  name: auth.state.user?.name ?? navigation.user.name,
  email: auth.state.user?.email ?? navigation.user.email,
  avatar: navigation.user.avatar,
}))

async function handleLogout() {
  auth.logout()
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
