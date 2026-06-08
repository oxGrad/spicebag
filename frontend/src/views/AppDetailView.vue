<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/apps" class="text-blue-600 hover:underline text-sm">← Applications</RouterLink>
    <button
      v-if="detail"
      @click="deleteApp"
      class="text-xs text-red-400 hover:text-red-600 hover:bg-red-50 border border-red-200 hover:border-red-300 rounded px-2 py-1 transition-colors"
    >Delete application</button>
  </div>
  <div v-if="detail" class="mb-5">
    <h1 class="text-2xl font-bold">{{ detail.app.Company }}</h1>
    <div class="flex items-center gap-3 mt-1">
      <p class="text-gray-500 text-sm">{{ detail.app.Role }}</p>
      <span class="text-gray-300 text-sm">·</span>
      <div class="flex items-center gap-1.5">
        <template v-if="editingDate">
          <input
            v-model="dateInput"
            type="text"
            placeholder="YYYY-MM-DD"
            maxlength="10"
            class="border rounded px-2 py-0.5 text-sm w-32 focus:outline-none focus:ring-1 focus:ring-gray-400"
            @keydown.enter="saveAppliedDate"
            @keydown.escape="editingDate = false"
          >
          <button @click="saveAppliedDate" class="text-xs text-blue-600 hover:underline">Save</button>
          <button @click="editingDate = false" class="text-xs text-gray-400 hover:text-gray-600">Cancel</button>
        </template>
        <template v-else>
          <span class="text-sm" :class="detail.app.AppliedDate ? 'text-gray-500' : 'text-gray-300'">
            {{ detail.app.AppliedDate ? 'Applied ' + detail.app.AppliedDate : 'Applied date not set' }}
          </span>
          <button @click="startEditDate" class="text-xs text-gray-400 hover:text-gray-600">✎</button>
        </template>
      </div>
      <button
        @click="copyAppId"
        class="flex items-center gap-1 text-xs border rounded px-2 py-0.5 text-gray-400 hover:text-gray-600 hover:bg-gray-50 transition-colors"
        :title="'Copy application ID ' + appId"
      >
        {{ idCopied ? '✓ Copied' : 'ID ' + appId }}
      </button>
    </div>
  </div>

  <div v-if="detail" class="flex gap-0 border-b border-gray-200 mb-6">
    <button
      v-for="tab in appTabs"
      :key="tab.id"
      @click="activeSection = tab.id"
      class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
      :class="activeSection === tab.id
        ? 'border-gray-900 text-gray-900'
        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'"
    >
      {{ tab.label }}
      <span
        v-if="tab.id === 'questions' && questions.length > 0"
        class="ml-1.5 text-xs bg-gray-200 text-gray-600 rounded-full px-1.5 py-0.5 font-normal"
      >{{ questions.length }}</span>
    </button>
  </div>

  <div v-if="detail && activeSection === 'documents'" class="grid grid-cols-2 gap-6 mb-6">
    <!-- Left: Documents -->
    <div class="bg-white rounded-lg shadow divide-y">
      <div class="px-5 py-4">
        <h2 class="font-semibold">Documents</h2>
      </div>

      <div
        v-for="doc in docs"
        :key="doc.key"
        class="flex items-center transition-colors"
        :class="docIsActive(doc) ? 'bg-blue-50' : ''"
      >
        <button
          @click="selectDoc(doc)"
          class="flex-1 text-left px-5 py-3.5 flex items-center justify-between"
          :class="docIsActive(doc) ? 'hover:bg-blue-50/80' : 'hover:bg-gray-50'"
        >
          <div>
            <p class="text-sm font-medium" :class="docIsActive(doc) ? 'text-blue-700' : ''">{{ doc.label }}</p>
            <p class="text-xs text-gray-400 mt-0.5">{{ doc.sub }}</p>
          </div>
          <span v-if="docIsActive(doc) && !comparing" class="text-blue-400 text-xs leading-none">●</span>
        </button>

        <!-- Compare button: only on Base CV -->
        <button
          v-if="doc.key === 'base'"
          @click="startCompare"
          class="mr-4 px-2.5 py-1 text-xs border rounded shrink-0 transition-colors"
          :class="comparing
            ? 'border-gray-800 bg-gray-800 text-white'
            : 'border-gray-300 text-gray-500 hover:border-gray-500 hover:text-gray-700 hover:bg-gray-50'"
          title="Compare base CV with tailored CV"
        >Compare</button>
      </div>

      <!-- PDF files -->
      <template v-if="pdfs.length > 0">
        <div class="px-5 py-2 text-xs text-gray-400 uppercase tracking-wide bg-gray-50 border-t">Generated PDFs</div>
        <div
          v-for="p in pdfs"
          :key="p.filename"
          class="flex items-center transition-colors"
          :class="activeDoc?.key === 'pdf:' + p.filename ? 'bg-blue-50' : ''"
        >
          <button @click="selectPDF(p)" class="flex-1 text-left px-5 py-3 flex items-center gap-3"
            :class="activeDoc?.key === 'pdf:' + p.filename ? 'hover:bg-blue-50/80' : 'hover:bg-gray-50'"
          >
            <span class="text-xs font-semibold px-1.5 py-0.5 rounded bg-red-100 text-red-600 shrink-0">PDF</span>
            <p class="text-sm font-medium truncate min-w-0" :class="activeDoc?.key === 'pdf:' + p.filename ? 'text-blue-700' : ''">
              {{ p.filename.replace(/\.pdf$/, '') }}
            </p>
            <span v-if="activeDoc?.key === 'pdf:' + p.filename" class="text-blue-400 text-xs leading-none ml-auto shrink-0">●</span>
          </button>
        </div>
      </template>
    </div>

    <!-- Right: Status history + info -->
    <div class="space-y-5">
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
            <input v-model="newNotes" type="text" class="border rounded px-2 py-1.5 text-sm w-36">
          </div>
          <button type="submit" class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Update</button>
        </form>
      </div>

      <div class="bg-white rounded-lg shadow p-5 space-y-4">
        <div v-if="detail.app.JobURL || detail.app.JobSummary">
          <h2 class="font-semibold mb-2">Job Post</h2>
          <a v-if="detail.app.JobURL" :href="detail.app.JobURL" target="_blank" rel="noopener"
             class="text-sm text-blue-600 hover:underline break-all block mb-2">{{ detail.app.JobURL }}</a>
          <p v-if="detail.app.JobSummary" class="text-sm text-gray-600 leading-relaxed">{{ detail.app.JobSummary }}</p>
        </div>
        <div v-if="detail.app.TailoringNotes">
          <h2 class="font-semibold mb-3 flex items-center gap-2">
            Tailoring Assessment
            <span
              v-if="detail.app.MatchScore != null"
              class="text-xs font-semibold px-2 py-0.5 rounded"
              :class="matchBadgeClass(detail.app.MatchScore)"
            >{{ detail.app.MatchScore }}% match</span>
          </h2>
          <div class="space-y-2">
            <div v-if="tailoring.strengths.bullets.length || tailoring.strengths.prose" class="rounded-md bg-green-50 border border-green-100 px-3 py-2">
              <span class="text-xs font-semibold text-green-700 uppercase tracking-wide">Strengths</span>
              <ul v-if="tailoring.strengths.bullets.length" class="mt-1 space-y-0.5 list-disc list-inside">
                <li v-for="b in tailoring.strengths.bullets" :key="b" class="text-sm text-green-900 leading-relaxed">{{ b }}</li>
              </ul>
              <p v-if="tailoring.strengths.prose" class="text-sm text-green-900 mt-1 leading-relaxed">{{ tailoring.strengths.prose }}</p>
            </div>
            <div v-if="tailoring.gaps.bullets.length || tailoring.gaps.prose" class="rounded-md bg-red-50 border border-red-100 px-3 py-2">
              <span class="text-xs font-semibold text-red-700 uppercase tracking-wide">Gaps</span>
              <ul v-if="tailoring.gaps.bullets.length" class="mt-1 space-y-0.5 list-disc list-inside">
                <li v-for="b in tailoring.gaps.bullets" :key="b" class="text-sm text-red-900 leading-relaxed">{{ b }}</li>
              </ul>
              <p v-if="tailoring.gaps.prose" class="text-sm text-red-900 mt-1 leading-relaxed">{{ tailoring.gaps.prose }}</p>
            </div>
            <div v-if="tailoring.plan.bullets.length || tailoring.plan.prose" class="rounded-md bg-gray-50 border border-gray-200 px-3 py-2">
              <span class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Tailoring Plan</span>
              <ul v-if="tailoring.plan.bullets.length" class="mt-1 space-y-0.5 list-disc list-inside">
                <li v-for="b in tailoring.plan.bullets" :key="b" class="text-sm text-gray-700 leading-relaxed">{{ b }}</li>
              </ul>
              <p v-if="tailoring.plan.prose" class="text-sm text-gray-700 mt-1 leading-relaxed">{{ tailoring.plan.prose }}</p>
            </div>
            <p v-if="!tailoring.strengths.bullets.length && !tailoring.strengths.prose && !tailoring.gaps.bullets.length && !tailoring.gaps.prose && !tailoring.plan.bullets.length && !tailoring.plan.prose"
               class="text-sm text-gray-600 leading-relaxed whitespace-pre-line">{{ detail.app.TailoringNotes }}</p>
          </div>
        </div>
        <div>
          <h2 class="font-semibold mb-1">Source</h2>
          <select v-model="selectedSource" @change="saveSource" class="border rounded px-2 py-1.5 text-sm w-full">
            <option value="">- not set -</option>
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
  </div>

  <!-- Compare view (Documents tab) -->
  <template v-if="activeSection === 'documents'">
  <div v-if="comparing" class="bg-white rounded-lg shadow overflow-hidden">
    <div class="flex items-center justify-between px-5 py-3 border-b bg-gray-50">
      <div class="flex items-center gap-6 text-sm">
        <span class="flex items-center gap-1.5">
          <span class="w-3 h-3 rounded-sm bg-red-200 border border-red-400 inline-block"></span>
          <span class="text-gray-600">Removed in tailored</span>
        </span>
        <span class="flex items-center gap-1.5">
          <span class="w-3 h-3 rounded-sm bg-green-200 border border-green-400 inline-block"></span>
          <span class="text-gray-600">Added in tailored</span>
        </span>
      </div>
      <div class="flex items-center gap-3">
        <select v-model="compareTheme" class="border rounded px-2 py-1 text-xs bg-white">
          <option value="">No theme</option>
          <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
        </select>
        <button @click="comparing = false" class="text-gray-400 hover:text-gray-600 text-lg leading-none">×</button>
      </div>
    </div>
    <div class="grid grid-cols-2" style="height: 75vh;">
      <div class="flex flex-col border-r">
        <div class="text-xs font-medium text-gray-500 px-3 py-1.5 bg-gray-50 border-b">Base CV</div>
        <iframe
          ref="leftIframeRef"
          :src="compareBaseUrl"
          class="flex-1 border-0 w-full"
          title="Base CV"
          @load="onLeftLoad"
        />
      </div>
      <div class="flex flex-col">
        <div class="text-xs font-medium text-gray-500 px-3 py-1.5 bg-gray-50 border-b">Tailored CV</div>
        <iframe
          ref="rightIframeRef"
          :src="compareTailoredUrl"
          class="flex-1 border-0 w-full"
          title="Tailored CV"
          @load="onRightLoad"
        />
      </div>
    </div>
  </div>

  <!-- Single document preview -->
  <div v-else-if="activeDoc" class="bg-white rounded-lg shadow overflow-hidden">
    <div class="flex items-center justify-between px-5 py-3 border-b bg-gray-50">
      <div class="flex items-center gap-2">
        <span v-if="activeDoc.isPDF" class="text-xs font-semibold px-1.5 py-0.5 rounded bg-red-100 text-red-600">PDF</span>
        <span class="text-sm font-medium text-gray-700">{{ activeDoc.label }}</span>
        <button @click="triggerRefresh" title="Refresh" class="text-gray-400 hover:text-gray-600 text-sm leading-none">
          <span :class="{ spinning: isRefreshing }">↺</span>
        </button>
      </div>
      <div class="flex items-center gap-3">
        <template v-if="!activeDoc.isPDF">
          <span
            v-if="activePDF"
            class="text-xs text-green-600 font-medium truncate max-w-[180px]"
            :title="activePDF.filename"
          >✓ {{ activePDF.filename }}</span>
          <button
            v-else
            @click="generatePDF"
            :disabled="generatingPDF"
            class="px-2.5 py-1 text-xs border rounded hover:bg-gray-50 disabled:opacity-50 transition-colors text-gray-600"
          >{{ generatingPDF ? 'Generating…' : 'Generate PDF' }}</button>
          <select v-model="previewTheme" class="border rounded px-2 py-1 text-xs bg-white">
            <option value="">No theme</option>
            <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
          </select>
        </template>
        <button @click="activeDoc = null" class="text-gray-400 hover:text-gray-600 text-lg leading-none">×</button>
      </div>
    </div>
    <iframe
      :key="activeDoc.key + previewTheme + previewKey"
      :src="activeDocUrl"
      class="w-full border-0"
      style="height: 75vh;"
      :title="activeDoc.label"
    />
  </div>
  </template>

  <!-- Questions tab -->
  <div v-if="detail && activeSection === 'questions'" class="bg-white rounded-lg shadow overflow-hidden">
    <div class="px-5 py-4 border-b flex items-center justify-between">
      <div>
        <h2 class="font-semibold">Application Questions</h2>
        <p class="text-xs text-gray-400 mt-0.5 flex items-center gap-1.5">
          Run <code class="bg-gray-100 px-1 rounded">/answer-questions {{ appId }}</code> to generate answers
          <button
            @click="copyAnswerCmd"
            class="border rounded px-1.5 py-0.5 text-gray-400 hover:text-gray-600 hover:bg-gray-50 transition-colors leading-none"
            title="Copy command"
          >{{ answerCmdCopied ? '✓' : '⎘' }}</button>
        </p>
      </div>
    </div>

    <!-- Question list -->
    <div v-if="questions.length > 0" class="divide-y">
      <div v-for="q in questions" :key="q.id" class="p-5 space-y-3">

        <!-- Question header -->
        <div class="flex items-start justify-between gap-3">
          <p class="text-sm font-semibold text-gray-800 flex-1 leading-snug">{{ q.question }}</p>
          <button @click="deleteQuestion(q.id)" class="text-gray-300 hover:text-red-400 text-sm shrink-0 mt-0.5">✕</button>
        </div>

        <!-- Bullet points editor -->
        <div class="rounded-md bg-gray-50 px-3 py-2.5 space-y-1.5">
          <p class="text-xs font-medium text-gray-400">Your notes <span class="font-normal">(optional — helps craft better answers)</span></p>
          <div v-for="(bullet, bi) in bulletEdits[q.id]" :key="bi" class="flex items-center gap-2">
            <span class="text-gray-300 text-sm shrink-0">•</span>
            <input
              v-model="bulletEdits[q.id][bi]"
              @blur="saveBullets(q.id)"
              type="text"
              placeholder="e.g. Led the migration of 3 legacy services"
              class="flex-1 text-sm border rounded px-2 py-1 bg-white focus:outline-none focus:ring-1 focus:ring-blue-300"
            >
            <button @click="removeBullet(q.id, bi)" class="text-gray-300 hover:text-red-400 text-xs shrink-0">✕</button>
          </div>
          <button
            @click="addBullet(q.id)"
            class="text-xs text-blue-500 hover:text-blue-700 flex items-center gap-1 mt-0.5"
          >+ Add note</button>
        </div>

        <!-- Answer -->
        <div v-if="q.answer" class="rounded-md bg-slate-50 border border-slate-100 p-3.5">
          <div class="flex items-center justify-between mb-2">
            <span class="text-xs font-medium text-slate-400">Answer</span>
            <button
              @click="copyAnswer(q)"
              class="text-xs px-2 py-0.5 border rounded transition-colors"
              :class="copiedId === q.id ? 'border-green-400 text-green-600 bg-green-50' : 'border-slate-200 text-slate-400 hover:text-slate-600 hover:bg-white'"
            >{{ copiedId === q.id ? '✓ Copied' : 'Copy' }}</button>
          </div>
          <p class="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">{{ q.answer }}</p>
        </div>
        <p v-else class="text-xs text-gray-400 italic">No answer yet — run <code class="bg-gray-100 px-1 rounded">/answer-questions {{ appId }}</code> to generate</p>

      </div>
    </div>
    <div v-else class="px-5 py-6 text-center text-gray-400 text-sm">No questions added yet.</div>

    <!-- Add question form -->
    <div class="px-5 py-4 border-t bg-gray-50">
      <form @submit.prevent="addQuestion" class="flex gap-2">
        <textarea
          v-model="newQuestion"
          placeholder="Paste a question from the application form…"
          rows="2"
          class="flex-1 border rounded px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-blue-400"
        ></textarea>
        <button
          type="submit"
          :disabled="!newQuestion.trim()"
          class="self-end bg-blue-600 text-white rounded px-3 py-2 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Add</button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api.js'
import { CV_DEFAULT_KEY, CL_DEFAULT_KEY, getDefaultTheme } from '../theme-defaults.js'

const route = useRoute()
const router = useRouter()

const appTabs = [
  { id: 'documents', label: 'Documents' },
  { id: 'questions', label: 'Questions' },
]
const activeSection = ref('documents')

const detail = ref(null)
const newStatus = ref('')
const newNotes = ref('')
const sources = ref([])
const selectedSource = ref('')
const themes = ref([])
const previewTheme = ref(getDefaultTheme(CV_DEFAULT_KEY))
const activeDoc = ref(null)
const previewKey = ref(0)
const isRefreshing = ref(false)
const generatingPDF = ref(false)
const pdfs = ref([])
const questions = ref([])
const newQuestion = ref('')
const copiedId = ref(null)
const idCopied = ref(false)
const bulletEdits = ref({}) // questionId -> string[]
const answerCmdCopied = ref(false)
const editingDate = ref(false)
const dateInput = ref('')

const appId = computed(() => route.params.id)

const activePDF = computed(() =>
  activeDoc.value ? pdfs.value.find(p => p.doc_type === activeDoc.value.key) ?? null : null
)

const tailoring = computed(() => {
  const raw = detail.value?.app?.TailoringNotes || ''
  // Matches "Label:" or "Label (qualifier):" and captures everything until the next known heading or end
  const extract = (label) => {
    const re = new RegExp(`${label}(?:\\s*\\([^)]*\\))?:\\s*([\\s\\S]*?)(?=\\n(?:Strengths|Gaps|Tailoring plan)(?:\\s*\\([^)]*\\))?:|$)`, 'i')
    const m = raw.match(re)
    if (!m) return { bullets: [], prose: '' }
    const text = m[1].trim()
    const lines = text.split('\n').map(l => l.trim()).filter(Boolean)
    const bullets = lines.filter(l => l.startsWith('- ')).map(l => l.slice(2))
    const prose = lines.filter(l => !l.startsWith('- ')).join(' ')
    return { bullets, prose }
  }
  return {
    strengths: extract('Strengths'),
    gaps: extract('Gaps'),
    plan: extract('Tailoring plan'),
  }
})

function matchBadgeClass(score) {
  if (score >= 80) return 'bg-green-100 text-green-800'
  if (score >= 60) return 'bg-yellow-100 text-yellow-800'
  return 'bg-red-100 text-red-800'
}

function triggerRefresh() {
  previewKey.value++
  isRefreshing.value = true
  setTimeout(() => { isRefreshing.value = false }, 400)
}

// Compare mode
const comparing = ref(false)
const compareTheme = ref(getDefaultTheme(CV_DEFAULT_KEY))
const leftIframeRef = ref(null)
const rightIframeRef = ref(null)
let leftLoaded = false
let rightLoaded = false
let syncingScroll = false

function docIsActive(doc) {
  return (comparing.value && doc.key === 'base') || (!comparing.value && activeDoc.value?.key === doc.key)
}

function themeForDoc(doc) {
  return doc.key === 'cl' ? getDefaultTheme(CL_DEFAULT_KEY) : getDefaultTheme(CV_DEFAULT_KEY)
}

const docs = computed(() => {
  if (!detail.value) return []
  const id = route.params.id
  const items = [
    { key: 'cv',   label: 'Tailored CV',  sub: 'Generated for this application', baseUrl: `/render/app/${id}/cv` },
    { key: 'cl',   label: 'Cover Letter', sub: 'Generated for this application', baseUrl: `/render/app/${id}/cl` },
  ]
  if (detail.value.app.BaseCVUsed) {
    items.push({
      key: 'base',
      label: 'Base CV',
      sub: detail.value.app.BaseCVUsed.replace(/\.html$/, ''),
      baseUrl: `/render/cv/${detail.value.app.BaseCVUsed}`,
    })
  }
  return items
})

const activeDocUrl = computed(() => {
  if (!activeDoc.value) return ''
  const t = previewTheme.value ? `?theme=${encodeURIComponent(previewTheme.value)}` : ''
  return activeDoc.value.baseUrl + t
})

const compareBaseUrl = computed(() => {
  if (!detail.value?.app.BaseCVUsed) return ''
  const t = compareTheme.value ? `?theme=${encodeURIComponent(compareTheme.value)}` : ''
  return `/render/cv/${detail.value.app.BaseCVUsed}${t}`
})

const compareTailoredUrl = computed(() => {
  const t = compareTheme.value ? `?theme=${encodeURIComponent(compareTheme.value)}` : ''
  return `/render/app/${route.params.id}/cv${t}`
})

function selectDoc(doc) {
  comparing.value = false
  if (activeDoc.value?.key === doc.key) {
    activeDoc.value = null
  } else {
    activeDoc.value = doc
    previewTheme.value = themeForDoc(doc)
  }
}

function selectPDF(p) {
  comparing.value = false
  const key = 'pdf:' + p.filename
  if (activeDoc.value?.key === key) {
    activeDoc.value = null
  } else {
    activeDoc.value = {
      key,
      label: p.filename.replace(/\.pdf$/, ''),
      sub: 'PDF',
      baseUrl: `/render/app/${route.params.id}/pdf/${encodeURIComponent(p.filename)}`,
      isPDF: true,
    }
    previewTheme.value = ''
  }
}

async function generatePDF() {
  if (!activeDoc.value) return
  generatingPDF.value = true
  try {
    const res = await api.apps.exportPDF(route.params.id, activeDoc.value.key, previewTheme.value)
    const docType = activeDoc.value.key
    // remove any previous entry for this doc type then add the new one
    pdfs.value = pdfs.value.filter(p => p.doc_type !== docType)
    pdfs.value.push({ filename: res.filename, doc_type: docType })
  } catch (e) {
    alert('PDF generation failed: ' + e.message)
  } finally {
    generatingPDF.value = false
  }
}

function startCompare() {
  activeDoc.value = null
  comparing.value = true
  leftLoaded = false
  rightLoaded = false
  compareTheme.value = getDefaultTheme(CV_DEFAULT_KEY)
}

function onLeftLoad() {
  leftLoaded = true
  if (rightLoaded) onBothLoaded()
}

function onRightLoad() {
  rightLoaded = true
  if (leftLoaded) onBothLoaded()
}

function onBothLoaded() {
  setupScrollSync()
  injectDiff()
}

function setupScrollSync() {
  const left = leftIframeRef.value?.contentWindow
  const right = rightIframeRef.value?.contentWindow
  if (!left || !right) return

  left.addEventListener('scroll', () => {
    if (syncingScroll) return
    syncingScroll = true
    const ratio = left.scrollY / Math.max(1, left.document.body.scrollHeight - left.innerHeight)
    right.scrollTo(0, ratio * Math.max(1, right.document.body.scrollHeight - right.innerHeight))
    requestAnimationFrame(() => { syncingScroll = false })
  })

  right.addEventListener('scroll', () => {
    if (syncingScroll) return
    syncingScroll = true
    const ratio = right.scrollY / Math.max(1, right.document.body.scrollHeight - right.innerHeight)
    left.scrollTo(0, ratio * Math.max(1, left.document.body.scrollHeight - left.innerHeight))
    requestAnimationFrame(() => { syncingScroll = false })
  })
}

function injectDiff() {
  const leftDoc = leftIframeRef.value?.contentDocument
  const rightDoc = rightIframeRef.value?.contentDocument
  if (!leftDoc || !rightDoc) return

  const leftTexts = new Set([...leftDoc.querySelectorAll('li')].map(el => el.innerText.trim()))
  const rightTexts = new Set([...rightDoc.querySelectorAll('li')].map(el => el.innerText.trim()))

  // Tailored (right): green = added relative to base
  rightDoc.querySelectorAll('li').forEach(el => {
    if (!leftTexts.has(el.innerText.trim())) {
      Object.assign(el.style, {
        background: '#dcfce7',
        borderLeft: '3px solid #16a34a',
        paddingLeft: '6px',
        marginLeft: '-9px',
      })
    }
  })

  // Base (left): red = removed in tailored
  leftDoc.querySelectorAll('li').forEach(el => {
    if (!rightTexts.has(el.innerText.trim())) {
      Object.assign(el.style, {
        background: '#fee2e2',
        borderLeft: '3px solid #ef4444',
        paddingLeft: '6px',
        marginLeft: '-9px',
        textDecoration: 'line-through',
        opacity: '0.75',
      })
    }
  })
}

onMounted(async () => {
  try {
    ;[detail.value, sources.value, themes.value, pdfs.value, questions.value] = await Promise.all([
      api.apps.get(route.params.id),
      api.sources.list(),
      api.themes.list(),
      api.apps.pdfs(route.params.id),
      api.apps.questions(route.params.id),
    ])
  } catch (e) {
    console.error('AppDetailView load failed:', e)
    alert('Failed to load application: ' + e.message)
    return
  }
  initBulletEdits(questions.value)
  newStatus.value = detail.value.valid_statuses[0]
  selectedSource.value = detail.value.app.Source ?? ''
  if (docs.value.length) {
    activeDoc.value = docs.value[0]
    previewTheme.value = themeForDoc(docs.value[0])
  }
})

function copyAnswerCmd() {
  navigator.clipboard.writeText(`/answer-questions ${appId.value}`)
  answerCmdCopied.value = true
  setTimeout(() => { answerCmdCopied.value = false }, 2000)
}

function copyAppId() {
  navigator.clipboard.writeText(String(appId.value))
  idCopied.value = true
  setTimeout(() => { idCopied.value = false }, 2000)
}

function initBulletEdits(qs) {
  const edits = {}
  for (const q of qs) edits[q.id] = [...(q.user_bullets ?? [])]
  bulletEdits.value = edits
}

async function addQuestion() {
  if (!newQuestion.value.trim()) return
  const q = await api.apps.addQuestion(route.params.id, newQuestion.value.trim(), questions.value.length)
  questions.value.push(q)
  bulletEdits.value[q.id] = []
  newQuestion.value = ''
}

async function deleteQuestion(qid) {
  await api.apps.deleteQuestion(route.params.id, qid)
  questions.value = questions.value.filter(q => q.id !== qid)
  delete bulletEdits.value[qid]
}

function addBullet(qid) {
  bulletEdits.value[qid] = [...(bulletEdits.value[qid] ?? []), '']
}

function removeBullet(qid, index) {
  const bullets = [...bulletEdits.value[qid]]
  bullets.splice(index, 1)
  bulletEdits.value[qid] = bullets
  saveBullets(qid)
}

async function saveBullets(qid) {
  const bullets = (bulletEdits.value[qid] ?? []).filter(b => b.trim())
  await api.apps.updateBullets(route.params.id, qid, bullets)
  // sync back into questions so the skill sees current bullets
  const q = questions.value.find(q => q.id === qid)
  if (q) q.user_bullets = bullets
}

function copyAnswer(q) {
  navigator.clipboard.writeText(q.answer)
  copiedId.value = q.id
  setTimeout(() => { copiedId.value = null }, 2000)
}

function startEditDate() {
  dateInput.value = detail.value.app.AppliedDate ?? ''
  editingDate.value = true
}

async function saveAppliedDate() {
  await api.apps.updateAppliedDate(appId.value, dateInput.value.trim())
  detail.value.app.AppliedDate = dateInput.value.trim()
  editingDate.value = false
}

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
    offer:      'bg-green-100 text-green-800',
    interview:  'bg-yellow-100 text-yellow-800',
    assessment: 'bg-yellow-100 text-yellow-800',
    rejected:   'bg-red-100 text-red-800',
    withdrawn:  'bg-red-100 text-red-800',
    ghosted:    'bg-red-100 text-red-800',
    skipped:    'bg-gray-100 text-gray-500',
  }
  return map[status?.toLowerCase()] ?? 'bg-blue-100 text-blue-800'
}

async function deleteApp() {
  if (!confirm(`Delete ${detail.value.app.Company} — ${detail.value.app.Role}? This cannot be undone.`)) return
  await api.apps.delete(appId.value)
  router.push('/apps')
}
</script>

<style scoped>
@keyframes spin-once {
  from { transform: rotate(0deg); }
  to   { transform: rotate(-360deg); }
}
.spinning {
  display: inline-block;
  animation: spin-once 0.4s ease-out forwards;
}
</style>
