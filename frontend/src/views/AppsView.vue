<template>
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold">Applications</h1>
    <button
      v-if="activeFilter"
      @click="clearFilter"
      class="flex items-center gap-1.5 text-xs border rounded-full px-3 py-1 text-gray-500 hover:text-gray-700 hover:bg-gray-50 transition-colors"
    >
      {{ activeFilter }} ×
    </button>
  </div>
  <div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 text-xs">
        <tr>
          <th class="text-left px-4 py-3">Company</th>
          <th class="text-left px-4 py-3">Role</th>
          <th class="text-left px-4 py-3">Applied</th>
          <th class="text-left px-4 py-3">Source</th>
          <th class="text-left px-4 py-3">Status</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="filtered.length === 0">
          <td colspan="5" class="px-4 py-8 text-center text-gray-400">
            <span v-if="activeFilter">No applications match "{{ activeFilter }}".</span>
            <span v-else>No applications yet. Use <code>/apply</code> in Claude Code to create one.</span>
          </td>
        </tr>
        <tr
          v-for="app in filtered"
          :key="app.ID"
          class="hover:bg-gray-50 cursor-pointer"
          @click="$router.push(`/apps/${app.ID}`)"
        >
          <td class="px-4 py-3 font-medium">{{ app.Company }}</td>
          <td class="px-4 py-3 text-gray-600">{{ app.Role }}</td>
          <td class="px-4 py-3 text-gray-500">{{ app.AppliedDate }}</td>
          <td class="px-4 py-3 text-gray-500 text-xs">{{ app.Source || '—' }}</td>
          <td class="px-4 py-3">
            <span class="px-2 py-0.5 rounded text-xs font-semibold" :class="badgeClass(app.CurrentStatus)">
              {{ app.CurrentStatus }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api.js'

const route = useRoute()
const router = useRouter()
const allApps = ref([])

const activeFilter = computed(() => {
  if (route.query.month) return `Month: ${route.query.month}`
  if (route.query.source) return `Source: ${route.query.source}`
  return null
})

const filtered = computed(() => {
  if (route.query.month) {
    return allApps.value.filter(a => a.AppliedDate?.startsWith(route.query.month))
  }
  if (route.query.source) {
    const src = route.query.source
    return allApps.value.filter(a => (a.Source || 'Unknown') === src)
  }
  return allApps.value
})

function clearFilter() {
  router.replace({ path: '/apps' })
}

onMounted(async () => {
  allApps.value = await api.apps.list()
})

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
