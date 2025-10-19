<script setup lang="ts">
import { ref } from 'vue'

const version = ref('1.0.0')
</script>

<template>
  <div class="max-w-4xl mx-auto text-left">
    <h1 class="text-5xl font-bold mb-6 text-center text-indigo-400">About Atlas</h1>
    <p class="text-lg text-gray-300 mb-8 text-center">
      Atlas is a Property Boundary Viewer built with Vue 3, TypeScript, and Vite.
    </p>

    <div class="mt-12">
      <h2 class="text-3xl font-semibold mb-6 text-indigo-300">Tech Stack</h2>
      <ul class="space-y-4">
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">Frontend:</strong>
          <span class="text-gray-300"> Vue 3.5.22 + TypeScript 5.9.3 + Tailwind CSS</span>
        </li>
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">Build Tool:</strong>
          <span class="text-gray-300"> Vite 7.1.10</span>
        </li>
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">State Management:</strong>
          <span class="text-gray-300"> Pinia 3.0.3</span>
        </li>
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">Routing:</strong>
          <span class="text-gray-300"> Vue Router 4.6.3</span>
        </li>
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">Testing:</strong>
          <span class="text-gray-300"> Vitest 3.2.4</span>
        </li>
        <li class="py-4 border-b border-gray-800">
          <strong class="text-indigo-400">Backend:</strong>
          <span class="text-gray-300"> Go 1.25 API</span>
        </li>
        <li class="py-4">
          <strong class="text-indigo-400">Database:</strong>
          <span class="text-gray-300"> PostgreSQL 18 + PostGIS 3.5</span>
        </li>
      </ul>
    </div>

    <p class="mt-12 text-center text-gray-500 text-sm">Version: {{ version }}</p>
  </div>
</template>


