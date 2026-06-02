<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/cv" class="text-blue-600 hover:underline text-sm">← CV Library</RouterLink>
    <div class="flex gap-2 items-center">
      <select v-model="selectedTheme" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
      </select>
      <GotenbergWidget :file-path="`cv/${name}`" :theme="selectedTheme" />
    </div>
  </div>
  <h1 class="text-xl font-bold mb-4">{{ name }}</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 80vh;">
    <iframe
      :src="renderSrc"
      class="w-full h-full border-0"
      title="CV Preview"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import GotenbergWidget from '../components/GotenbergWidget.vue'

const route = useRoute()
const name = computed(() => route.params.name)
const themes = ref([])
const selectedTheme = ref('')

const renderSrc = computed(() => {
  const t = selectedTheme.value ? `?theme=${encodeURIComponent(selectedTheme.value)}` : ''
  return `/render/cv/${name.value}${t}`
})

onMounted(async () => {
  themes.value = await api.themes.list()
})
</script>
