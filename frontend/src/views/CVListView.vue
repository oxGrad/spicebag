<template>
  <h1 class="text-2xl font-bold mb-6">Curriculum Vitae</h1>
  <div class="bg-white rounded-lg shadow divide-y">
    <div v-if="files.length === 0" class="px-4 py-8 text-center text-gray-400">
      No CVs yet. Use <code>/seed-cv</code> in Claude Code to import one.
    </div>
    <div
      v-for="f in files"
      :key="f.Name"
      class="flex items-center justify-between px-4 py-3 hover:bg-gray-50 cursor-pointer"
      @click="$router.push(`/cv/${f.Name}`)"
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
onMounted(async () => { files.value = await api.cv.list() })
</script>
