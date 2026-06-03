<template>
  <div class="mb-4">
    <RouterLink to="/apps" class="text-blue-600 hover:underline text-sm">← Applications</RouterLink>
  </div>
  <div v-if="detail" class="mb-6">
    <h1 class="text-2xl font-bold">{{ detail.app.Company }}</h1>
    <p class="text-gray-500">{{ detail.app.Role }} · Applied {{ detail.app.AppliedDate }}</p>
  </div>
  <div v-if="detail" class="grid grid-cols-2 gap-6">
    <div class="space-y-6">
      <div class="bg-white rounded-lg shadow p-5">
        <h2 class="font-semibold mb-3">Status History</h2>
        <div class="space-y-2 mb-4">
          <div v-for="entry in detail.history" :key="entry.ID" class="flex gap-3 text-sm">
            <span class="text-gray-400 text-xs w-24 shrink-0">{{ entry.ChangedAt }}</span>
            <span class="px-2 py-0.5 rounded text-xs font-semibold" :class="badgeClass(entry.Status)">{{ entry.Status }}</span>
            <span v-if="entry.Notes" class="text-gray-500">{{ entry.Notes }}</span>
          </div>
        </div>
        <form class="flex gap-2 items-end" @submit.prevent="submitStatus">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">New Status</label>
            <select v-model="newStatus" class="border rounded px-2 py-1.5 text-sm">
              <option v-for="s in detail.valid_statuses" :key="s" :value="s">{{ s }}</option>
            </select>
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Notes (optional)</label>
            <input v-model="newNotes" type="text" class="border rounded px-2 py-1.5 text-sm w-40">
          </div>
          <button type="submit" class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Update</button>
        </form>
      </div>

      <div class="bg-white rounded-lg shadow p-5 space-y-4">
        <div v-if="detail.app.JobURL || detail.app.JobSummary">
          <h2 class="font-semibold mb-2">Job Post</h2>
          <a
            v-if="detail.app.JobURL"
            :href="detail.app.JobURL"
            target="_blank"
            rel="noopener"
            class="text-sm text-blue-600 hover:underline break-all block mb-2"
          >{{ detail.app.JobURL }}</a>
          <p v-if="detail.app.JobSummary" class="text-sm text-gray-600 leading-relaxed">{{ detail.app.JobSummary }}</p>
        </div>
        <div>
          <h2 class="font-semibold mb-1">Source</h2>
          <select
            v-model="selectedSource"
            @change="saveSource"
            class="border rounded px-2 py-1.5 text-sm w-full"
          >
            <option value="">— not set —</option>
            <option v-for="src in sources" :key="src.id" :value="src.name">{{ src.name }}</option>
          </select>
        </div>
        <div>
          <h2 class="font-semibold mb-1">Notes</h2>
          <p v-if="detail.app.Notes" class="text-sm text-gray-700">{{ detail.app.Notes }}</p>
          <p v-else class="text-gray-400 text-sm">No notes.</p>
        </div>
      </div>
    </div>

    <div class="space-y-6">
      <div class="bg-white rounded-lg shadow p-5">
        <h2 class="font-semibold mb-4">Documents</h2>

        <div class="space-y-3">
          <div class="flex items-center justify-between text-sm">
            <div>
              <p class="font-medium">Tailored CV</p>
              <p class="text-xs text-gray-400">Generated for this application</p>
            </div>
            <a
              :href="appCVUrl"
              target="_blank"
              class="text-blue-600 hover:underline text-xs"
            >Open →</a>
          </div>

          <div class="border-t pt-3 flex items-center justify-between text-sm">
            <div>
              <p class="font-medium">Cover Letter</p>
              <p class="text-xs text-gray-400">Generated for this application</p>
            </div>
            <a
              :href="appCLUrl"
              target="_blank"
              class="text-blue-600 hover:underline text-xs"
            >Open →</a>
          </div>

          <template v-if="detail.app.BaseCVUsed">
            <div class="border-t pt-3 flex items-center justify-between text-sm">
              <div>
                <p class="font-medium text-gray-500">Base CV</p>
                <p class="text-xs text-gray-400">{{ detail.app.BaseCVUsed.replace(/\.html$/, '') }}</p>
              </div>
              <a
                :href="`/render/cv/${detail.app.BaseCVUsed}`"
                target="_blank"
                class="text-gray-400 hover:text-blue-600 hover:underline text-xs"
              >Open →</a>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { CV_DEFAULT_KEY, getDefaultTheme } from '../theme-defaults.js'

const route = useRoute()
const detail = ref(null)
const newStatus = ref('')
const newNotes = ref('')
const sources = ref([])
const selectedSource = ref('')

const defaultTheme = getDefaultTheme(CV_DEFAULT_KEY)

const appCVUrl = computed(() => {
  if (!detail.value) return ''
  const t = defaultTheme ? `?theme=${encodeURIComponent(defaultTheme)}` : ''
  return `/render/app/${route.params.id}/cv${t}`
})

const appCLUrl = computed(() => {
  if (!detail.value) return ''
  const t = defaultTheme ? `?theme=${encodeURIComponent(defaultTheme)}` : ''
  return `/render/app/${route.params.id}/cl${t}`
})

onMounted(async () => {
  ;[detail.value, sources.value] = await Promise.all([
    api.apps.get(route.params.id),
    api.sources.list(),
  ])
  newStatus.value = detail.value.valid_statuses[0]
  selectedSource.value = detail.value.app.Source ?? ''
})

async function saveSource() {
  await api.apps.updateSource(route.params.id, selectedSource.value)
}

async function submitStatus() {
  const updated = await api.apps.updateStatus(route.params.id, newStatus.value, newNotes.value)
  detail.value.history = updated
  newNotes.value = ''
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
</script>
