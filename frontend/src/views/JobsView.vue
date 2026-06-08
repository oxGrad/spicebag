<template>
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold">Scraped Jobs</h1>
    <label class="flex items-center gap-2 text-sm text-gray-500">
      <input type="checkbox" v-model="showDismissed" @change="load" />
      Show dismissed
    </label>
  </div>

  <div v-if="failedCompanies.length" class="mb-4 rounded-md bg-amber-50 border border-amber-200 px-4 py-2 text-sm text-amber-800">
    ⚠ {{ failedCompanies.length }} of {{ companies.length }} companies failed to scrape —
    <RouterLink to="/settings" class="underline">see Settings</RouterLink>.
  </div>

  <p class="text-xs text-gray-400 mb-4">
    Run <code>/scrape-jobs</code> in Claude Code to refresh this list.
  </p>

  <div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 text-xs">
        <tr>
          <th class="text-left px-4 py-3">Company</th>
          <th class="text-left px-4 py-3">Role</th>
          <th class="text-left px-4 py-3">Location</th>
          <th class="text-left px-4 py-3">Why matched</th>
          <th class="text-left px-4 py-3">Found</th>
          <th class="text-left px-4 py-3">Action</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="jobs.length === 0">
          <td colspan="6" class="px-4 py-8 text-center text-gray-400">
            No scraped jobs. Run <code>/scrape-jobs</code> in Claude Code.
          </td>
        </tr>
        <tr v-for="job in jobs" :key="job.id" class="hover:bg-gray-50 align-top">
          <td class="px-4 py-3 font-medium">{{ job.company_name }}</td>
          <td class="px-4 py-3">
            <a :href="job.url" target="_blank" rel="noopener" class="text-blue-600 hover:underline">{{ job.title }}</a>
          </td>
          <td class="px-4 py-3 text-gray-500">{{ job.location || '—' }}</td>
          <td class="px-4 py-3 text-gray-500 text-xs">{{ job.match_reason || '—' }}</td>
          <td class="px-4 py-3 text-gray-400 text-xs">{{ (job.scraped_at || '').slice(0, 10) }}</td>
          <td class="px-4 py-3">
            <div class="flex items-center gap-2">
              <button @click="toggleApply(job)" class="text-xs border rounded px-2 py-1 text-gray-600 hover:bg-gray-50">
                {{ openApplyId === job.id ? 'Hide' : 'Apply' }}
              </button>
              <button v-if="job.status !== 'dismissed'" @click="setStatus(job, 'dismissed')"
                class="text-xs text-gray-400 hover:text-red-600">Dismiss</button>
              <button v-else @click="setStatus(job, 'new')"
                class="text-xs text-gray-400 hover:text-blue-600">Restore</button>
            </div>
            <div v-if="openApplyId === job.id" class="mt-2 bg-gray-900 text-gray-100 rounded px-3 py-2 text-xs font-mono flex items-center justify-between gap-3">
              <span class="truncate">/apply {{ job.url }}</span>
              <button @click="copyCmd(job)" class="shrink-0 text-gray-300 hover:text-white">
                {{ copiedId === job.id ? '✓ Copied' : 'Copy' }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const jobs = ref([])
const companies = ref([])
const showDismissed = ref(false)
const openApplyId = ref(null)
const copiedId = ref(null)
const failedCompanies = ref([])

async function load() {
  jobs.value = await api.scrape.jobs(showDismissed.value ? 'dismissed' : 'new')
  companies.value = await api.scrape.companies()
  failedCompanies.value = companies.value.filter(c => c.last_scrape_status === 'error')
}

function toggleApply(job) {
  openApplyId.value = openApplyId.value === job.id ? null : job.id
}

async function copyCmd(job) {
  await navigator.clipboard.writeText(`/apply ${job.url}`)
  copiedId.value = job.id
  setTimeout(() => { copiedId.value = null }, 1500)
}

async function setStatus(job, status) {
  await api.scrape.setJobStatus(job.id, status)
  await load()
}

onMounted(load)
</script>
