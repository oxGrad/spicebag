<template>
  <h1 class="text-2xl font-bold mb-6">Overview</h1>

  <div v-if="data" class="space-y-8">
    <!-- Summary cards -->
    <div class="grid grid-cols-3 gap-4">
      <div class="bg-white rounded-lg shadow p-5">
        <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Total Applied</p>
        <p class="text-3xl font-bold">{{ totalApplied }}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-5">
        <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">This Month</p>
        <p class="text-3xl font-bold">{{ thisMonth }}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-5">
        <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Response Rate</p>
        <p class="text-3xl font-bold">{{ overallResponseRate }}%</p>
      </div>
    </div>

    <!-- Applications per month -->
    <div class="bg-white rounded-lg shadow p-5">
      <h2 class="font-semibold mb-4">Applications by Month</h2>
      <div v-if="data.per_month.length === 0" class="text-gray-400 text-sm">No data yet.</div>
      <div v-else class="space-y-2">
        <div v-for="m in data.per_month" :key="m.month" class="flex items-center gap-3">
          <span class="text-xs text-gray-500 w-16 shrink-0">{{ m.month }}</span>
          <div class="flex-1 bg-gray-100 rounded-full h-5 overflow-hidden">
            <div
              class="bg-blue-500 h-full rounded-full transition-all"
              :style="{ width: barWidth(m.count) }"
            ></div>
          </div>
          <span class="text-sm font-medium w-6 text-right">{{ m.count }}</span>
        </div>
      </div>
    </div>

    <!-- Source performance -->
    <div class="bg-white rounded-lg shadow overflow-hidden">
      <div class="px-5 py-4 border-b">
        <h2 class="font-semibold">Source Performance</h2>
        <p class="text-xs text-gray-400 mt-0.5">Drop rate = % of applications that never moved past "applied"</p>
      </div>
      <div v-if="data.source_stats.length === 0" class="px-5 py-8 text-center text-gray-400 text-sm">
        No applications with sources yet. Set a source on your applications to see stats here.
      </div>
      <table v-else class="w-full text-sm">
        <thead class="bg-gray-50 text-gray-500 uppercase text-xs">
          <tr>
            <th class="text-left px-5 py-3">Source</th>
            <th class="text-right px-5 py-3">Applied</th>
            <th class="text-right px-5 py-3">Responded</th>
            <th class="text-right px-5 py-3">Interviewed</th>
            <th class="text-right px-5 py-3">Offered</th>
            <th class="text-right px-5 py-3">Drop Rate</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="s in data.source_stats" :key="s.source" class="hover:bg-gray-50">
            <td class="px-5 py-3 font-medium">{{ s.source }}</td>
            <td class="px-5 py-3 text-right">{{ s.total }}</td>
            <td class="px-5 py-3 text-right">
              <span :class="rateClass(s.responded / s.total)">{{ pct(s.responded, s.total) }}</span>
            </td>
            <td class="px-5 py-3 text-right">{{ pct(s.interviewed, s.total) }}</td>
            <td class="px-5 py-3 text-right text-green-600 font-medium">{{ pct(s.offered, s.total) }}</td>
            <td class="px-5 py-3 text-right">
              <span :class="dropClass(s.drop_rate)">{{ s.drop_rate.toFixed(0) }}%</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const data = ref(null)

const totalApplied = computed(() =>
  data.value?.per_month.reduce((s, m) => s + m.count, 0) ?? 0
)

const thisMonth = computed(() => {
  if (!data.value) return 0
  const ym = new Date().toISOString().slice(0, 7)
  return data.value.per_month.find(m => m.month === ym)?.count ?? 0
})

const overallResponseRate = computed(() => {
  if (!data.value?.source_stats.length) return 0
  const total = data.value.source_stats.reduce((s, r) => s + r.total, 0)
  const responded = data.value.source_stats.reduce((s, r) => s + r.responded, 0)
  return total ? ((responded / total) * 100).toFixed(0) : 0
})

const maxCount = computed(() =>
  data.value ? Math.max(...data.value.per_month.map(m => m.count), 1) : 1
)

function barWidth(count) {
  return `${(count / maxCount.value) * 100}%`
}

function pct(n, total) {
  return total ? `${((n / total) * 100).toFixed(0)}%` : '—'
}

function rateClass(ratio) {
  if (ratio >= 0.3) return 'text-green-600 font-medium'
  if (ratio >= 0.1) return 'text-yellow-600'
  return 'text-red-500'
}

function dropClass(rate) {
  if (rate >= 80) return 'text-red-500 font-medium'
  if (rate >= 50) return 'text-yellow-600'
  return 'text-green-600'
}

onMounted(async () => { data.value = await api.analytics.get() })
</script>
