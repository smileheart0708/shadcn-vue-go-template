<script setup lang="ts">
import { computed } from 'vue'
import { IconCreditCard, IconDotsVertical, IconLogout, IconNotification, IconUserCircle } from '@tabler/icons-vue'
import { Globe } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/components/ui/sidebar'
import { getAvatarFallbackText } from '@/lib/avatar'
import { getUserRoleBadgeVariant, getUserRoleLabelKey } from '@/lib/auth/roles'
import type { RoleKey } from '@/lib/api/auth'
import { useI18n } from 'vue-i18n'
import { supportedLocales, localeNames, type AppLocale } from '@/plugins/i18n'
import { useLocaleStore } from '@/stores/locale'

const props = defineProps<{
  user: {
    username: string
    email: string | null
    avatarUrl: string | null
    role: RoleKey
  }
}>()

const emit = defineEmits<{ logout: [] }>()

const { isMobile } = useSidebar()
const { t } = useI18n()
const localeStore = useLocaleStore()
const avatarFallbackText = computed(() => getAvatarFallbackText(props.user.username))
const roleLabel = computed(() => t(getUserRoleLabelKey(props.user.role)))
const roleBadgeVariant = computed(() => getUserRoleBadgeVariant(props.user.role))

function switchLanguage(newLocale: AppLocale) {
  localeStore.setLocale(newLocale)
}
</script>

<template>
  <SidebarMenu>
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <SidebarMenuButton
            size="lg"
            class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
          >
            <Avatar class="size-8 rounded-lg">
              <AvatarImage
                v-if="user.avatarUrl"
                :src="user.avatarUrl"
                :alt="user.username"
              />
              <AvatarFallback class="rounded-lg">{{ avatarFallbackText }}</AvatarFallback>
            </Avatar>
            <div class="grid flex-1 text-start text-sm/tight">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium">{{ user.username }}</span>
                <Badge
                  :variant="roleBadgeVariant"
                  class="px-1.5 py-0 text-[10px]"
                >
                  {{ roleLabel }}
                </Badge>
              </div>
              <span class="truncate text-xs text-muted-foreground">
                {{ user.email ?? ' ' }}
              </span>
            </div>
            <IconDotsVertical class="ms-auto size-4" />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          class="w-(--reka-dropdown-menu-trigger-width) min-w-56 rounded-lg"
          :side="isMobile ? 'bottom' : 'right'"
          :side-offset="4"
          align="end"
        >
          <DropdownMenuLabel class="p-0 font-normal">
            <div class="flex items-center gap-2 px-1 py-1.5 text-start text-sm">
              <Avatar class="size-8 rounded-lg">
                <AvatarImage
                  v-if="user.avatarUrl"
                  :src="user.avatarUrl"
                  :alt="user.username"
                />
                <AvatarFallback class="rounded-lg">{{ avatarFallbackText }}</AvatarFallback>
              </Avatar>
              <div class="grid flex-1 text-start text-sm/tight">
                <div class="flex items-center gap-2">
                  <span class="truncate font-medium">{{ user.username }}</span>
                  <Badge
                    :variant="roleBadgeVariant"
                    class="px-1.5 py-0 text-[10px]"
                  >
                    {{ roleLabel }}
                  </Badge>
                </div>
                <span class="truncate text-xs text-muted-foreground">
                  {{ user.email ?? ' ' }}
                </span>
              </div>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem>
              <IconUserCircle />
              {{ t('nav.user.account') }}
            </DropdownMenuItem>
            <DropdownMenuItem>
              <IconCreditCard />
              {{ t('nav.user.billing') }}
            </DropdownMenuItem>
            <DropdownMenuItem>
              <IconNotification />
              {{ t('nav.user.notifications') }}
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Globe />
                {{ t('nav.user.language') }}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup
                  :model-value="localeStore.locale"
                  @update:model-value="switchLanguage($event as AppLocale)"
                >
                  <DropdownMenuRadioItem
                    v-for="lang in supportedLocales"
                    :key="lang"
                    :value="lang"
                  >
                    {{ localeNames[lang] }}
                  </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="emit('logout')">
            <IconLogout />
            {{ t('nav.user.logout') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
