<template>
  <h1 class="text-2xl font-bold mb-6">Experience Stats</h1>
  <div v-if="stats" class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
        <tr>
          <th class="text-left px-4 py-3">Role Type</th>
          <th class="text-left px-4 py-3">Years</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="!stats.Entries || stats.Entries.length === 0">
          <td colspan="2" class="px-4 py-8 text-center text-gray-400">
            No experience data. Use the <code>add_experience</code> MCP tool in Claude Code to add entries.
          </td>
        </tr>
        <tr v-for="entry in stats.Entries" :key="entry.RoleType" class="hover:bg-gray-50">
          <td class="px-4 py-3 font-medium">{{ entry.RoleType }}</td>
          <td class="px-4 py-3 text-gray-600">{{ entry.Years }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <p class="text-xs text-gray-400 mt-2">Experience is managed via the <code>add_experience</code> MCP tool in Claude Code.</p>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const stats = ref(null)
onMounted(async () => { stats.value = await api.stats.get() })
</script>
