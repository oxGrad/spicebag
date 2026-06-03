<template>
  <h1 class="text-2xl font-bold mb-6">Sources</h1>
  <div class="max-w-md space-y-6">
    <div class="bg-white rounded-lg shadow divide-y">
      <div v-if="sources.length === 0" class="px-4 py-8 text-center text-gray-400 text-sm">No sources yet.</div>
      <div
        v-for="src in sources"
        :key="src.id"
        class="flex items-center justify-between px-4 py-3"
      >
        <span class="text-sm">{{ src.name }}</span>
        <button
          @click="remove(src.id)"
          class="text-xs text-red-400 hover:text-red-600"
        >Remove</button>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow p-4">
      <h2 class="font-semibold text-sm mb-3">Add Source</h2>
      <form class="flex gap-2" @submit.prevent="add">
        <input
          v-model="newName"
          type="text"
          placeholder="e.g. AngelList"
          class="flex-1 border rounded px-3 py-1.5 text-sm"
        >
        <button
          type="submit"
          :disabled="!newName.trim()"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Add</button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const sources = ref([])
const newName = ref('')

onMounted(async () => { sources.value = await api.sources.list() })

async function add() {
  if (!newName.value.trim()) return
  await api.sources.add(newName.value.trim())
  sources.value = await api.sources.list()
  newName.value = ''
}

async function remove(id) {
  await api.sources.delete(id)
  sources.value = sources.value.filter(s => s.id !== id)
}
</script>
