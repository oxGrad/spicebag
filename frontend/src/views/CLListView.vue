<template>
  <h1 class="text-2xl font-bold mb-6">Cover Letters</h1>
  <div class="bg-white rounded-lg shadow divide-y">
    <div v-if="files.length === 0" class="px-4 py-8 text-center text-gray-400">
      No cover letters yet. Use <code>/apply</code> in Claude Code to create one.
    </div>
    <div
      v-for="f in files"
      :key="f.Name"
      class="flex items-center justify-between px-4 py-3 hover:bg-gray-50 cursor-pointer"
      @click="$router.push(`/cl/${f.Name}`)"
    >
      <span class="font-medium">{{ f.Name.replace(/\.html$/, '') }}</span>
      <span class="text-xs text-gray-400">{{ new Date(f.ModifiedAt).toLocaleDateString() }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const files = ref([])
onMounted(async () => { files.value = await api.cl.list() })
</script>
