<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-gray-950 text-gray-100">
    <header class="bg-gray-900 border-b border-gray-800">
      <nav class="max-w-7xl mx-auto px-8 py-4 flex justify-between items-center">
        <div>
          <h1 class="text-2xl font-bold text-indigo-500">Atlas</h1>
        </div>
        <div class="flex gap-8">
          <RouterLink
            to="/"
            class="text-white font-medium px-4 py-2 rounded transition-all hover:bg-gray-800 hover:text-indigo-400 router-link-active:text-indigo-500 router-link-active:bg-indigo-500/10"
          >
            Home
          </RouterLink>
          <RouterLink
            to="/about"
            class="text-white font-medium px-4 py-2 rounded transition-all hover:bg-gray-800 hover:text-indigo-400 router-link-active:text-indigo-500 router-link-active:bg-indigo-500/10"
          >
            About
          </RouterLink>
        </div>
      </nav>
    </header>

    <main class="flex-1 max-w-7xl mx-auto px-8 py-8 w-full">
      <RouterView />
    </main>

    <footer class="bg-gray-900 border-t border-gray-800 py-4 text-center text-gray-500 text-sm">
      <p>&copy; 2025 Atlas - Property Boundary Viewer</p>
    </footer>
  </div>
</template>
