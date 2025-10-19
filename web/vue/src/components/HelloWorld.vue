<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ msg: string }>()

const count = ref(0)
</script>

<template>
  <h1 class="text-5xl font-bold mb-8">{{ msg }}</h1>

  <div class="p-8 mb-8">
    <button
      type="button"
      class="px-6 py-3 bg-indigo-600 text-white rounded-lg font-medium transition-colors hover:bg-indigo-700 focus:outline-none focus:ring-4 focus:ring-indigo-500/50"
      @click="count++"
    >
      count is {{ count }}
    </button>
    <p class="mt-6 text-gray-400">
      Edit
      <code class="bg-gray-800 px-2 py-1 rounded text-indigo-400">components/HelloWorld.vue</code>
      to test HMR
    </p>
  </div>

  <div class="space-y-4 text-gray-300">
    <p>
      Check out
      <a
        href="https://vuejs.org/guide/quick-start.html#local"
        target="_blank"
        rel="noopener"
        class="text-indigo-400 hover:text-indigo-300 transition-colors"
      >
        create-vue</a
      >, the official Vue + Vite starter
    </p>
    <p>
      Learn more about IDE Support for Vue in the
      <a
        href="https://vuejs.org/guide/scaling-up/tooling.html#ide-support"
        target="_blank"
        rel="noopener"
        class="text-indigo-400 hover:text-indigo-300 transition-colors"
      >
        Vue Docs Scaling up Guide</a
      >.
    </p>
    <p class="text-gray-500">Click on the Vite and Vue logos to learn more</p>
  </div>
</template>
