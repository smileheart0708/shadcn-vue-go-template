<script setup lang="ts">
import type { HTMLAttributes } from "vue"

import { cn } from "@/lib/utils"
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

const props = defineProps<{
  class?: HTMLAttributes["class"]
}>()
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <form>
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <img src="/logo.svg" alt="Logo" class="size-8" />
          <h1 class="text-xl font-bold">
            Welcome back
          </h1>
        </div>
        <Field>
          <FieldLabel for="email">
            Email
          </FieldLabel>
          <Input
            id="email"
            type="email"
            placeholder="m@example.com"
            required
          />
        </Field>
        <Field>
          <Button type="submit">
            Login
          </Button>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
