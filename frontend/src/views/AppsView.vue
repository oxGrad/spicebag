<template>
  <h1 class="text-2xl font-bold mb-6">Applications</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
        <tr>
          <th class="text-left px-4 py-3">Company</th>
          <th class="text-left px-4 py-3">Role</th>
          <th class="text-left px-4 py-3">Applied</th>
          <th class="text-left px-4 py-3">Source</th>
          <th class="text-left px-4 py-3">Status</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="apps.length === 0">
          <td colspan="5" class="px-4 py-8 text-center text-gray-400">
            No applications yet. Use <code>/apply</code> in Claude Code to create one.
          </td>
        </tr>
        <tr
          v-for="app in apps"
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
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const apps = ref([])

onMounted(async () => {
  apps.value = await api.apps.list()
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
