<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-semibold text-gray-900">Memory</h1>
      <div class="flex gap-2">
        <button
          v-for="t in ['all', 'user', 'feedback', 'project', 'reference']"
          :key="t"
          @click="filter = t"
          :class="[
            'px-3 py-1 rounded-full text-xs font-medium transition-colors',
            filter === t
              ? 'bg-gray-900 text-white'
              : 'bg-white text-gray-600 border border-gray-200 hover:border-gray-400'
          ]"
        >{{ t }}</button>
      </div>
    </div>

    <div v-if="loading" class="text-sm text-gray-500">Loading…</div>
    <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
    <div v-else-if="filtered.length === 0" class="text-sm text-gray-400">No memories.</div>

    <div v-else class="flex gap-6">
      <div class="flex-1 min-w-0">
        <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 border-b border-gray-200">
              <tr>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Description</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Updated</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              <tr
                v-for="m in filtered"
                :key="m.name"
                @click="selected = m"
                :class="[
                  'cursor-pointer transition-colors',
                  selected?.name === m.name ? 'bg-blue-50' : 'hover:bg-gray-50'
                ]"
              >
                <td class="px-4 py-3 font-mono text-xs text-gray-900">{{ m.name }}</td>
                <td class="px-4 py-3">
                  <span :class="typeBadge(m.type)">{{ m.type }}</span>
                </td>
                <td class="px-4 py-3 text-gray-600 truncate max-w-xs">{{ m.description }}</td>
                <td class="px-4 py-3 text-gray-400 whitespace-nowrap">{{ formatDate(m.updated_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-if="selected" class="w-80 shrink-0">
        <div class="bg-white border border-gray-200 rounded-lg p-4 sticky top-8">
          <div class="flex items-start justify-between mb-3">
            <div>
              <p class="font-mono text-xs text-gray-900 font-medium">{{ selected.name }}</p>
              <span :class="typeBadge(selected.type)" class="mt-1 inline-block">{{ selected.type }}</span>
            </div>
            <button @click="selected = null" class="text-gray-400 hover:text-gray-600 text-lg leading-none">×</button>
          </div>
          <p class="text-xs text-gray-500 mb-3 italic">{{ selected.description }}</p>
          <p class="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">{{ selected.body }}</p>
          <p class="mt-4 text-xs text-gray-400">Updated {{ formatDate(selected.updated_at) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const memories = ref([])
const loading = ref(true)
const error = ref(null)
const selected = ref(null)
const filter = ref('all')

const filtered = computed(() =>
  filter.value === 'all' ? memories.value : memories.value.filter(m => m.type === filter.value)
)

onMounted(async () => {
  try {
    memories.value = await api.memory.list()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function typeBadge(type) {
  const map = {
    user:      'px-1.5 py-0.5 rounded text-xs bg-blue-100 text-blue-700',
    feedback:  'px-1.5 py-0.5 rounded text-xs bg-amber-100 text-amber-700',
    project:   'px-1.5 py-0.5 rounded text-xs bg-green-100 text-green-700',
    reference: 'px-1.5 py-0.5 rounded text-xs bg-purple-100 text-purple-700',
  }
  return map[type] ?? 'px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-600'
}

function formatDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
}
</script>
