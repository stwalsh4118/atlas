<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goHome = () => {
  router.push('/')
}
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-[60vh] text-center">
    <h1 class="text-9xl font-bold m-0 text-indigo-500">404</h1>
    <h2 class="text-4xl font-semibold my-4">Page Not Found</h2>
    <p class="text-gray-500 my-4 mb-8">The page you're looking for doesn't exist.</p>
    <button
      class="px-8 py-3 text-lg bg-indigo-600 text-white border-none rounded-lg cursor-pointer transition-colors hover:bg-indigo-700 focus:outline-none focus:ring-4 focus:ring-indigo-500/50"
      @click="goHome"
    >
      Go to Home
    </button>
  </div>
</template>


