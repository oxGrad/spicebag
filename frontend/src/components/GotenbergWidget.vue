<template>
  <div class="flex gap-2 items-center">
    <button
      v-if="!running"
      @click="start"
      :disabled="starting"
      class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50 disabled:opacity-50"
    >
      {{ starting ? 'Starting…' : 'Start Gotenberg' }}
    </button>
    <button
      v-if="running"
      @click="exportPDF"
      class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700"
    >
      Export PDF
    </button>
    <button
      v-if="running"
      @click="stop"
      class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50 text-gray-500"
    >
      Stop
    </button>
    <span v-if="err" class="text-xs text-red-500">{{ err }}</span>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  filePath: { type: String, required: true },
  theme:    { type: String, default: '' },
})

const running  = ref(false)
const starting = ref(false)
const err      = ref('')

onMounted(async () => {
  const s = await api.gotenberg.status()
  running.value = s.running
})

async function start() {
  starting.value = true
  err.value = ''
  try {
    const s = await api.gotenberg.start()
    running.value = s.running
    if (s.error) err.value = s.error
  } catch (e) {
    err.value = e.message
  }
  starting.value = false
}

async function stop() {
  const s = await api.gotenberg.stop()
  running.value = s.running
}

async function exportPDF() {
  err.value = ''
  try {
    const res = await api.export(props.filePath, props.theme)
    if (!res.ok) { err.value = 'Export failed'; return }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = props.filePath.split('/').pop().replace('.html', '.pdf')
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    err.value = e.message
  }
}
</script>
