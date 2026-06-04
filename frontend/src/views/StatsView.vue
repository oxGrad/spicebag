<template>
  <h1 class="text-2xl font-bold mb-6">Experience Stats</h1>
  <div v-if="rows.length > 0" class="bg-white rounded-lg shadow overflow-hidden mb-4">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
        <tr>
          <th class="text-left px-4 py-3">Role Type</th>
          <th class="text-right px-4 py-3">Years</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-for="row in rows" :key="row.role" class="hover:bg-gray-50">
          <td class="px-4 py-3 font-medium">{{ row.role }}</td>
          <td class="px-4 py-3 text-right text-gray-600">{{ row.years }}</td>
        </tr>
      </tbody>
      <tfoot class="bg-gray-50 border-t border-gray-200">
        <tr>
          <td class="px-4 py-3 font-semibold">Total</td>
          <td class="px-4 py-3 text-right font-semibold">{{ total }}</td>
        </tr>
      </tfoot>
    </table>
  </div>
  <div v-else-if="loaded" class="bg-white rounded-lg shadow px-4 py-8 text-center text-gray-400 text-sm">
    No experience data. Use the <code>add_experience</code> MCP tool to sync from your CV.
  </div>
  <p class="text-xs text-gray-400 mt-2">Experience is managed via the <code>add_experience</code> MCP tool in Claude Code.</p>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const stats = ref(null)
const loaded = ref(false)

onMounted(async () => {
  stats.value = await api.stats.get()
  loaded.value = true
})

const rows = computed(() => {
  if (!stats.value?.ByRole) return []
  return Object.entries(stats.value.ByRole)
    .map(([role, yrs]) => ({ role, years: yrs.toFixed(1) }))
    .sort((a, b) => b.years - a.years)
})

const total = computed(() => stats.value?.Total?.toFixed(1) ?? '0.0')
</script>
