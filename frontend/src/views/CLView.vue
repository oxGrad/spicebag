<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/cl" class="text-blue-600 hover:underline text-sm">← Cover Letters</RouterLink>
    <div class="flex gap-2 items-center">
      <select v-model="selectedTheme" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
      </select>
      <button
        @click="refresh"
        title="Refresh preview (R)"
        class="border rounded px-2 py-1.5 text-sm bg-white hover:bg-gray-50"
      ><span :class="{ spinning: isSpinning }">↺</span></button>
      <ExportButton :file-path="`cover-letters/${name}`" :theme="selectedTheme" />
    </div>
  </div>
  <h1 class="text-xl font-bold mb-4">{{ name.replace(/\.html$/, '') }}</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 80vh;">
    <iframe :key="refreshKey" :src="renderSrc" class="w-full h-full border-0" title="Cover Letter Preview" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { CL_DEFAULT_KEY, getDefaultTheme } from '../theme-defaults.js'
import ExportButton from '../components/ExportButton.vue'

const route = useRoute()
const name = computed(() => route.params.name)
const themes = ref([])
const selectedTheme = ref(getDefaultTheme(CL_DEFAULT_KEY))
const refreshKey = ref(0)
const isSpinning = ref(false)

const renderSrc = computed(() => {
  const t = selectedTheme.value ? `?theme=${encodeURIComponent(selectedTheme.value)}` : ''
  return `/render/cl/${name.value}${t}`
})

function refresh() {
  refreshKey.value++
  isSpinning.value = true
  setTimeout(() => { isSpinning.value = false }, 400)
}

function onKeyDown(e) {
  if (e.key === 'r' && !['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) {
    refresh()
  }
}

onMounted(async () => {
  themes.value = await api.themes.list()
  document.addEventListener('keydown', onKeyDown)
})
onBeforeUnmount(() => document.removeEventListener('keydown', onKeyDown))
</script>

<style scoped>
@keyframes spin-once {
  from { transform: rotate(0deg); }
  to   { transform: rotate(-360deg); }
}
.spinning {
  display: inline-block;
  animation: spin-once 0.4s ease-out forwards;
}
</style>
