<template>
  <!-- Header + filter -->
  <div class="flex items-start justify-between mb-6 gap-4">
    <div>
      <h1 class="text-2xl font-bold">Overview</h1>
      <p class="text-sm text-gray-400 mt-0.5">{{ rangeLabel }}</p>
    </div>

    <div class="flex flex-col items-end gap-2 shrink-0">
      <!-- Preset pills -->
      <div class="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
        <button
          v-for="p in presets"
          :key="p.id"
          @click="preset = p.id"
          class="px-3 py-1.5 rounded-md text-sm font-medium transition-all"
          :class="preset === p.id
            ? 'bg-white shadow-sm text-gray-900'
            : 'text-gray-500 hover:text-gray-700'"
        >{{ p.label }}</button>
      </div>

      <!-- Custom range inputs (shown only when custom preset active) -->
      <div v-if="preset === 'custom'" class="flex items-center gap-2">
        <input
          v-model="customFrom"
          type="month"
          class="border rounded px-2 py-1 text-sm bg-white focus:outline-none focus:ring-1 focus:ring-gray-400"
          :max="customTo || undefined"
        >
        <span class="text-gray-400 text-sm">—</span>
        <input
          v-model="customTo"
          type="month"
          class="border rounded px-2 py-1 text-sm bg-white focus:outline-none focus:ring-1 focus:ring-gray-400"
          :min="customFrom || undefined"
        >
      </div>
    </div>
  </div>

  <!-- Loading skeleton -->
  <div v-if="loading" class="space-y-8 animate-pulse">
    <div class="bg-white rounded-lg shadow p-5 h-24 flex gap-6 items-end">
      <div class="w-16 h-8 bg-gray-100 rounded"></div>
      <div class="w-16 h-6 bg-gray-100 rounded"></div>
      <div class="w-16 h-6 bg-gray-100 rounded"></div>
      <div class="w-16 h-5 bg-gray-100 rounded"></div>
    </div>
    <div class="bg-white rounded-lg shadow p-5 h-52 bg-gray-50"></div>
    <div class="bg-white rounded-lg shadow h-40 bg-gray-50"></div>
  </div>

  <!-- Error state -->
  <div v-else-if="error" class="bg-white rounded-lg shadow p-10 text-center">
    <p class="text-gray-500 mb-3">{{ error }}</p>
    <button @click="load" class="text-sm text-blue-600 hover:underline">Retry</button>
  </div>

  <div v-else-if="data" class="space-y-8">
    <!-- Pipeline funnel -->
    <div class="bg-white rounded-lg shadow p-5">
      <p class="text-xs font-medium text-gray-400 mb-4">Application pipeline</p>
      <div class="grid grid-cols-4 divide-x divide-gray-100">
        <div class="pr-5">
          <p class="text-2xl font-semibold tabular-nums">{{ funnel.total }}</p>
          <p class="text-sm text-gray-500 mt-0.5">Applied</p>
          <p v-if="preset === 'all'" class="text-xs text-gray-400 mt-1">{{ thisMonth }} this month</p>
        </div>
        <div class="px-5">
          <p class="text-2xl font-semibold tabular-nums" :class="funnel.total === 0 ? 'text-gray-200' : ''">{{ funnel.responded }}</p>
          <p class="text-sm text-gray-500 mt-0.5">Responded</p>
          <p class="text-xs text-gray-400 mt-1">{{ funnel.respondedPct }}% of applied</p>
        </div>
        <div class="px-5">
          <p class="text-2xl font-semibold tabular-nums" :class="funnel.total === 0 ? 'text-gray-200' : ''">{{ funnel.interviewed }}</p>
          <p class="text-sm text-gray-500 mt-0.5">Interviewed</p>
          <p class="text-xs text-gray-400 mt-1">{{ funnel.interviewedPct }}% of applied</p>
        </div>
        <div class="pl-5">
          <p class="text-2xl font-semibold tabular-nums"
            :class="funnel.offered > 0 ? 'text-green-600' : funnel.total === 0 ? 'text-gray-200' : ''">
            {{ funnel.offered }}
          </p>
          <p class="text-sm text-gray-500 mt-0.5">Offered</p>
          <p class="text-xs text-gray-400 mt-1">{{ funnel.offeredPct }}% of applied</p>
        </div>
      </div>
    </div>

    <!-- Applications per month -->
    <div class="bg-white rounded-lg shadow p-5">
      <h2 class="font-semibold mb-4">Applications by month</h2>
      <div v-if="data.per_month.length === 0" class="text-gray-400 text-sm">No data for this period.</div>
      <div v-else class="space-y-2">
        <RouterLink
          v-for="m in data.per_month"
          :key="m.month"
          :to="`/apps?month=${m.month}`"
          class="flex items-center gap-3 group"
        >
          <span class="text-xs text-gray-500 w-16 shrink-0 group-hover:text-blue-500 transition-colors">{{ m.month }}</span>
          <div class="flex-1 bg-gray-100 rounded-full h-4 overflow-hidden">
            <div
              class="bg-blue-400 h-full rounded-full transition-all group-hover:bg-blue-500"
              :style="{ width: barWidth(m.count) }"
            ></div>
          </div>
          <span class="text-sm font-medium w-6 text-right text-gray-600 group-hover:text-blue-600">{{ m.count }}</span>
        </RouterLink>
      </div>
    </div>

    <!-- Source performance -->
    <div class="bg-white rounded-lg shadow overflow-hidden">
      <div class="px-5 py-4 border-b">
        <h2 class="font-semibold">Source performance</h2>
        <p class="text-xs text-gray-400 mt-0.5">Drop rate: % of applications that never moved past "applied"</p>
      </div>
      <div v-if="data.source_stats.length === 0" class="px-5 py-8 text-center text-gray-400 text-sm">
        No applications yet. Set a source on your applications to see performance here.
      </div>
      <table v-else class="w-full text-sm">
        <thead class="bg-gray-50 text-gray-600 text-xs">
          <tr>
            <th class="text-left px-5 py-3">Source</th>
            <th class="text-right px-5 py-3">Applied</th>
            <th class="text-right px-5 py-3">Responded</th>
            <th class="text-right px-5 py-3">Interviewed</th>
            <th class="text-right px-5 py-3">Offered</th>
            <th class="text-right px-5 py-3">Drop rate</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <RouterLink
            v-for="s in data.source_stats"
            :key="s.source"
            :to="`/apps?source=${encodeURIComponent(s.source)}`"
            custom
            v-slot="{ navigate }"
          >
            <tr class="hover:bg-gray-50 cursor-pointer" @click="navigate">
              <td class="px-5 py-3 font-medium">{{ s.source }}</td>
              <td class="px-5 py-3 text-right text-gray-600">{{ s.total }}</td>
              <td class="px-5 py-3 text-right">
                <span :class="rateClass(s.responded / s.total)">{{ pct(s.responded, s.total) }}</span>
                <span v-if="s.responded > 0" class="ml-1.5 text-xs text-gray-400">{{ rateLabel(s.responded / s.total) }}</span>
              </td>
              <td class="px-5 py-3 text-right text-gray-600">{{ pct(s.interviewed, s.total) }}</td>
              <td class="px-5 py-3 text-right font-medium" :class="s.offered > 0 ? 'text-green-600' : 'text-gray-300'">
                {{ s.offered > 0 ? pct(s.offered, s.total) : '—' }}
              </td>
              <td class="px-5 py-3 text-right">
                <span :class="dropClass(s.drop_rate)">{{ s.drop_rate.toFixed(0) }}%</span>
                <span class="ml-1.5 text-xs text-gray-400">{{ dropLabel(s.drop_rate) }}</span>
              </td>
            </tr>
          </RouterLink>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api } from '../api.js'

const data    = ref(null)
const loading = ref(true)
const error   = ref(null)

// --- Filter state ---
const presets = [
  { id: 'all',    label: 'All time' },
  { id: '6m',     label: '6 months' },
  { id: '3m',     label: '3 months' },
  { id: '1m',     label: 'This month' },
  { id: 'custom', label: 'Custom' },
]
const preset     = ref('all')
const customFrom = ref('')
const customTo   = ref('')

function ymNow() {
  return new Date().toISOString().slice(0, 7)
}

function ymOffset(monthsAgo) {
  const d = new Date()
  d.setDate(1)
  d.setMonth(d.getMonth() - monthsAgo)
  return d.toISOString().slice(0, 7)
}

const filterRange = computed(() => {
  switch (preset.value) {
    case 'all': return { from: '', to: '' }
    case '6m':  return { from: ymOffset(5), to: ymNow() }
    case '3m':  return { from: ymOffset(2), to: ymNow() }
    case '1m':  return { from: ymNow(), to: ymNow() }
    case 'custom': return { from: customFrom.value, to: customTo.value }
    default:    return { from: '', to: '' }
  }
})

function formatMonthShort(ym) {
  const [y, m] = ym.split('-')
  return new Date(+y, +m - 1).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

function formatMonthAbbr(ym) {
  const [y, m] = ym.split('-')
  return new Date(+y, +m - 1).toLocaleDateString('en-US', { month: 'short' })
}

const rangeLabel = computed(() => {
  const { from, to } = filterRange.value
  if (!from && !to) return 'All time'
  if (from && to && from === to) return formatMonthShort(from)
  if (from && to) {
    const fromYear = from.slice(0, 4)
    const toYear   = to.slice(0, 4)
    if (fromYear === toYear) {
      return `${formatMonthAbbr(from)} — ${formatMonthShort(to)}`
    }
    return `${formatMonthShort(from)} — ${formatMonthShort(to)}`
  }
  if (from) return `from ${formatMonthShort(from)}`
  return `up to ${formatMonthShort(to)}`
})

// --- Derived data ---
const thisMonth = computed(() => {
  if (!data.value) return 0
  const ym = ymNow()
  return data.value.per_month.find(m => m.month === ym)?.count ?? 0
})

const funnel = computed(() => {
  const ss = data.value?.source_stats ?? []
  const total      = ss.reduce((s, r) => s + r.total, 0)
  const responded  = ss.reduce((s, r) => s + r.responded, 0)
  const interviewed = ss.reduce((s, r) => s + r.interviewed, 0)
  const offered    = ss.reduce((s, r) => s + r.offered, 0)
  const p = (n) => total ? Math.round((n / total) * 100) : 0
  return { total, responded, interviewed, offered,
    respondedPct: p(responded), interviewedPct: p(interviewed), offeredPct: p(offered) }
})

const maxCount = computed(() =>
  data.value ? Math.max(...data.value.per_month.map(m => m.count), 1) : 1
)

function barWidth(count) {
  return `${(count / maxCount.value) * 100}%`
}

function pct(n, total) {
  return total ? `${Math.round((n / total) * 100)}%` : '—'
}

function rateClass(ratio) {
  if (ratio >= 0.3) return 'text-green-600 font-medium'
  if (ratio >= 0.1) return 'text-yellow-600'
  return 'text-red-500'
}

function rateLabel(ratio) {
  if (!ratio) return ''
  if (ratio >= 0.3) return 'strong'
  if (ratio >= 0.1) return 'weak'
  return 'low'
}

function dropClass(rate) {
  if (rate >= 80) return 'text-red-500 font-medium'
  if (rate >= 50) return 'text-yellow-600'
  return 'text-green-600'
}

function dropLabel(rate) {
  if (rate >= 80) return 'very high'
  if (rate >= 50) return 'high'
  return 'good'
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const { from, to } = filterRange.value
    data.value = await api.analytics.get(from, to)
  } catch {
    error.value = 'Failed to load analytics. Check that the server is running.'
  } finally {
    loading.value = false
  }
}

watch(filterRange, load)
onMounted(load)
</script>
