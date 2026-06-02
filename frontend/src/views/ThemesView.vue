<template>
  <h1 class="text-2xl font-bold mb-6">Themes</h1>
  <div class="grid grid-cols-2 gap-6">
    <div>
      <h2 class="font-semibold mb-3">Available Themes</h2>
      <div class="bg-white rounded-lg shadow divide-y mb-6">
        <div v-if="themes.length === 0" class="px-4 py-8 text-center text-gray-400">No themes yet.</div>
        <div
          v-for="t in themes"
          :key="t"
          class="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
        >
          <span class="font-medium">{{ t }}</span>
          <button @click="previewTheme = t" class="text-sm text-blue-600 hover:underline">Preview →</button>
        </div>
      </div>

      <h2 class="font-semibold mb-3">Upload New Theme</h2>
      <div class="bg-white rounded-lg shadow p-4 flex gap-3 items-end">
        <div class="flex flex-col gap-1">
          <label class="text-xs text-gray-500">CSS File</label>
          <input type="file" accept=".css" @change="onFileChange" class="text-sm">
        </div>
        <button
          @click="upload"
          :disabled="!selectedFile"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Upload</button>
      </div>
    </div>

    <div v-if="previewTheme">
      <h2 class="font-semibold mb-3">Preview: {{ previewTheme }}</h2>
      <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 60vh;">
        <iframe
          :src="`/render/cv/base.html?theme=${encodeURIComponent(previewTheme)}`"
          class="w-full h-full border-0"
          title="Theme Preview"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const themes       = ref([])
const previewTheme = ref('')
const selectedFile = ref(null)

onMounted(async () => { themes.value = await api.themes.list() })

function onFileChange(e) { selectedFile.value = e.target.files[0] ?? null }

async function upload() {
  if (!selectedFile.value) return
  await api.themes.upload(selectedFile.value)
  themes.value = await api.themes.list()
  selectedFile.value = null
}
</script>
