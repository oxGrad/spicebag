<template>
  <h1 class="text-2xl font-bold mb-6">Settings</h1>

  <div class="flex gap-0 border-b border-gray-200 mb-6">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      @click="activeTab = tab.id"
      class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
      :class="activeTab === tab.id
        ? 'border-gray-900 text-gray-900'
        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'"
    >{{ tab.label }}</button>
  </div>

  <!-- Themes tab -->
  <div v-if="activeTab === 'themes'" class="grid grid-cols-2 gap-6">
    <div>
      <h2 class="font-semibold mb-3">Available Themes</h2>
      <div class="bg-white rounded-lg shadow divide-y mb-6">
        <div v-if="themes.length === 0" class="px-4 py-8 text-center text-gray-400">No themes yet.</div>
        <div
          v-for="t in themes"
          :key="t"
          class="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
        >
          <div class="flex items-center gap-2 min-w-0">
            <span class="font-medium truncate">{{ t }}</span>
            <span
              v-if="defaultCV === t"
              class="shrink-0 text-xs px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 font-medium"
            >CV</span>
            <span
              v-if="defaultCL === t"
              class="shrink-0 text-xs px-1.5 py-0.5 rounded bg-violet-100 text-violet-700 font-medium"
            >CL</span>
          </div>
          <div class="flex items-center gap-2 shrink-0 ml-3">
            <div class="relative" @click.stop>
              <button
                @click="toggleDropdown(t)"
                class="text-xs border rounded px-2 py-1 hover:bg-gray-50 flex items-center gap-1"
              >Set default <span class="text-gray-400">▾</span></button>
              <div
                v-if="openDropdown === t"
                class="absolute right-0 top-full mt-1 z-10 bg-white border rounded shadow-lg min-w-[160px]"
              >
                <button @click="setDefault(t, 'cv')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center justify-between">
                  <span>CV default</span>
                  <span v-if="defaultCV === t" class="text-blue-600 text-xs">✓</span>
                </button>
                <button @click="setDefault(t, 'cl')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center justify-between">
                  <span>CL default</span>
                  <span v-if="defaultCL === t" class="text-violet-600 text-xs">✓</span>
                </button>
                <div class="border-t my-1"></div>
                <button @click="setDefault(t, 'both')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50">
                  Both
                </button>
              </div>
            </div>
            <button @click="previewTheme = t; openDropdown = null" class="text-sm text-blue-600 hover:underline">Preview →</button>
          </div>
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

  <!-- Sources tab -->
  <div v-if="activeTab === 'sources'" class="max-w-md space-y-6">
    <div class="bg-white rounded-lg shadow divide-y">
      <div v-if="sources.length === 0" class="px-4 py-8 text-center text-gray-400 text-sm">No sources yet.</div>
      <div
        v-for="src in sources"
        :key="src.id"
        class="flex items-center justify-between px-4 py-3"
      >
        <span class="text-sm">{{ src.name }}</span>
        <button
          @click="removeSource(src.id)"
          class="text-xs text-red-400 hover:text-red-600"
        >Remove</button>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow p-4">
      <h2 class="font-semibold text-sm mb-3">Add Source</h2>
      <form class="flex gap-2" @submit.prevent="addSource">
        <input
          v-model="newSourceName"
          type="text"
          placeholder="e.g. AngelList"
          class="flex-1 border rounded px-3 py-1.5 text-sm"
        >
        <button
          type="submit"
          :disabled="!newSourceName.trim()"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Add</button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { CV_DEFAULT_KEY, CL_DEFAULT_KEY } from '../theme-defaults.js'

const route = useRoute()

const tabs = [
  { id: 'themes', label: 'Themes' },
  { id: 'sources', label: 'Sources' },
]
const activeTab = ref(route.query.tab === 'sources' ? 'sources' : 'themes')

// Themes state
const themes       = ref([])
const previewTheme = ref('')
const selectedFile = ref(null)
const openDropdown = ref(null)
const defaultCV    = ref(localStorage.getItem(CV_DEFAULT_KEY) ?? '')
const defaultCL    = ref(localStorage.getItem(CL_DEFAULT_KEY) ?? '')

// Sources state
const sources     = ref([])
const newSourceName = ref('')

onMounted(async () => {
  themes.value  = await api.themes.list()
  sources.value = await api.sources.list()
})

function toggleDropdown(name) {
  openDropdown.value = openDropdown.value === name ? null : name
}

function setDefault(name, target) {
  if (target === 'cv' || target === 'both') {
    const next = defaultCV.value === name ? '' : name
    defaultCV.value = next
    next ? localStorage.setItem(CV_DEFAULT_KEY, next) : localStorage.removeItem(CV_DEFAULT_KEY)
  }
  if (target === 'cl' || target === 'both') {
    const next = defaultCL.value === name ? '' : name
    defaultCL.value = next
    next ? localStorage.setItem(CL_DEFAULT_KEY, next) : localStorage.removeItem(CL_DEFAULT_KEY)
  }
  openDropdown.value = null
}

function closeDropdown(e) {
  if (!e.target.closest('.relative')) openDropdown.value = null
}

onMounted(() => document.addEventListener('click', closeDropdown))
onBeforeUnmount(() => document.removeEventListener('click', closeDropdown))

function onFileChange(e) { selectedFile.value = e.target.files[0] ?? null }

async function upload() {
  if (!selectedFile.value) return
  await api.themes.upload(selectedFile.value)
  themes.value = await api.themes.list()
  selectedFile.value = null
}

async function addSource() {
  if (!newSourceName.value.trim()) return
  await api.sources.add(newSourceName.value.trim())
  sources.value = await api.sources.list()
  newSourceName.value = ''
}

async function removeSource(id) {
  await api.sources.delete(id)
  sources.value = sources.value.filter(s => s.id !== id)
}
</script>
