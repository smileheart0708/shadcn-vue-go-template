<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

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
  <Card class="w-full shadow-sm">
    <CardHeader>
      <CardTitle>404</CardTitle>
      <CardDescription>Page Not Found</CardDescription>
    </CardHeader>
    <CardContent>
      <p class="text-sm text-muted-foreground">
        The page you are looking for does not exist.
      </p>
    </CardContent>
    <CardFooter>
      <Button @click="handleBack">
        Go Back
      </Button>
    </CardFooter>
  </Card>
</template>
