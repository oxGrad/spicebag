<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/cv" class="text-blue-600 hover:underline text-sm">← CV Library</RouterLink>
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
      <ExportButton :file-path="`cv/${name}`" :theme="selectedTheme" />
    </div>
  </div>
  <h1 class="text-xl font-bold mb-4">{{ name.replace(/\.html$/, '') }}</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 80vh;">
    <iframe
      :key="refreshKey"
      :src="renderSrc"
      class="w-full h-full border-0"
      title="CV Preview"
    />
  </div>

  <div v-if="usages.length > 0" class="mt-4 bg-white rounded-lg shadow overflow-hidden">
    <div class="px-4 py-3 border-b flex items-center justify-between">
      <h2 class="font-semibold text-sm">Used in Applications</h2>
      <span class="text-xs text-gray-400">{{ usages.length }}</span>
    </div>
    <div
      v-for="app in usages"
      :key="app.ID"
      class="flex items-center justify-between px-4 py-2.5 hover:bg-gray-50 cursor-pointer border-b last:border-0"
      @click="$router.push(`/apps/${app.ID}`)"
    >
      <div class="flex items-center gap-3 min-w-0">
        <span class="font-medium text-sm truncate">{{ app.Company }}</span>
        <span class="text-gray-500 text-sm truncate">{{ app.Role }}</span>
      </div>
      <div class="flex items-center gap-3 shrink-0 ml-3">
        <span class="text-xs text-gray-400">{{ app.AppliedDate }}</span>
        <span class="px-2 py-0.5 rounded text-xs font-semibold" :class="badgeClass(app.CurrentStatus)">
          {{ app.CurrentStatus }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { CV_DEFAULT_KEY, getDefaultTheme } from '../theme-defaults.js'
import ExportButton from '../components/ExportButton.vue'

const route = useRoute()
const name = computed(() => route.params.name)
const themes = ref([])
const selectedTheme = ref(getDefaultTheme(CV_DEFAULT_KEY))
const refreshKey = ref(0)
const isSpinning = ref(false)
const usages = ref([])

const renderSrc = computed(() => {
  const t = selectedTheme.value ? `?theme=${encodeURIComponent(selectedTheme.value)}` : ''
  return `/render/cv/${name.value}${t}`
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

function badgeClass(status) {
  const map = {
    offer: 'bg-green-100 text-green-800',
    interview: 'bg-yellow-100 text-yellow-800',
    assessment: 'bg-yellow-100 text-yellow-800',
    rejected: 'bg-red-100 text-red-800',
    withdrawn: 'bg-red-100 text-red-800',
    ghosted: 'bg-red-100 text-red-800',
  }
  return map[status?.toLowerCase()] ?? 'bg-blue-100 text-blue-800'
}

onMounted(async () => {
  ;[themes.value, usages.value] = await Promise.all([
    api.themes.list(),
    api.cv.usages(name.value),
  ])
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
