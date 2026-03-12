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
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <Card class="mx-auto mt-20 max-w-md">
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
        <Button @click="router.back()">
          Go Back
        </Button>
      </CardFooter>
    </Card>
  </div>
</template>
