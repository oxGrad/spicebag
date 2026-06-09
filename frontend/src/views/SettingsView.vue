<template>
  <h1 class="text-2xl font-bold mb-6">Settings</h1>

  <div class="flex gap-0 border-b border-gray-200 mb-6">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      @click="activeTab = tab.id"
      class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
      :class="activeTab === tab.id
        ? 'border-gray-900 text-gray-900'
        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'"
    >{{ tab.label }}</button>
  </div>

  <!-- Themes tab -->
  <div v-if="activeTab === 'themes'" class="grid grid-cols-2 gap-6">
    <div>
      <h2 class="font-semibold mb-3">Available Themes</h2>
      <div class="bg-white rounded-lg shadow divide-y mb-6">
        <div v-if="themes.length === 0" class="px-4 py-8 text-center text-gray-400">No themes yet.</div>
        <div
          v-for="t in themes"
          :key="t"
          class="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
        >
          <div class="flex items-center gap-2 min-w-0">
            <span class="font-medium truncate">{{ t }}</span>
            <span
              v-if="defaultCV === t"
              class="shrink-0 text-xs px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 font-medium"
            >CV</span>
            <span
              v-if="defaultCL === t"
              class="shrink-0 text-xs px-1.5 py-0.5 rounded bg-violet-100 text-violet-700 font-medium"
            >CL</span>
          </div>
          <div class="flex items-center gap-2 shrink-0 ml-3">
            <div class="relative" @click.stop>
              <button
                @click="toggleDropdown(t)"
                class="text-xs border rounded px-2 py-1 hover:bg-gray-50 flex items-center gap-1"
              >Set default <span class="text-gray-400">▾</span></button>
              <div
                v-if="openDropdown === t"
                class="absolute right-0 top-full mt-1 z-10 bg-white border rounded shadow-lg min-w-[160px]"
              >
                <button @click="setDefault(t, 'cv')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center justify-between">
                  <span>CV default</span>
                  <span v-if="defaultCV === t" class="text-blue-600 text-xs">✓</span>
                </button>
                <button @click="setDefault(t, 'cl')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center justify-between">
                  <span>CL default</span>
                  <span v-if="defaultCL === t" class="text-violet-600 text-xs">✓</span>
                </button>
                <div class="border-t my-1"></div>
                <button @click="setDefault(t, 'both')" class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50">
                  Both
                </button>
              </div>
            </div>
            <button @click="previewTheme = t; openDropdown = null" class="text-sm text-blue-600 hover:underline">Preview →</button>
          </div>
        </div>
      </div>

      <h2 class="font-semibold mb-3">Upload New Theme</h2>
      <div class="bg-white rounded-lg shadow p-4 flex gap-3 items-end">
        <div class="flex flex-col gap-1">
          <label class="text-xs text-gray-500">CSS File</label>
          <input type="file" accept=".css" @change="onFileChange" class="text-sm">
        </div>
        <button
          @click="upload"
          :disabled="!selectedFile"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Upload</button>
      </div>
    </div>

    <div v-if="previewTheme">
      <h2 class="font-semibold mb-3">Preview: {{ previewTheme }}</h2>
      <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 60vh;">
        <iframe
          :src="`/render/cv/base.html?theme=${encodeURIComponent(previewTheme)}`"
          class="w-full h-full border-0"
          title="Theme Preview"
        />
      </div>
    </div>
  </div>

  <!-- Job Scraping tab -->
  <div v-if="activeTab === 'scraping'" class="max-w-2xl">
    <section class="bg-white rounded-lg shadow p-5 mb-6">
      <h2 class="font-semibold mb-3">Job Boards</h2>
      <p class="text-xs text-gray-500 mb-3">Enable boards to fetch jobs from. Run <code>/scrape-jobs</code> in Claude Code to pull listings.</p>
      <ul class="divide-y divide-gray-100">
        <li v-for="b in boards" :key="b.id" class="flex items-center justify-between py-3 text-sm">
          <div>
            <span class="font-medium">{{ b.label }}</span>
            <div class="text-xs mt-0.5" :class="b.last_scrape_status === 'error' ? 'text-red-600' : 'text-gray-400'">
              <template v-if="b.last_scrape_status === 'ok'">✅ {{ b.last_job_count }} jobs · {{ (b.last_scraped_at||'').slice(0,16) }}</template>
              <template v-else-if="b.last_scrape_status === 'error'">🔴 {{ b.last_scrape_error }}</template>
              <template v-else>Not scraped yet</template>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" :checked="b.enabled" @change="toggleBoard(b)" class="sr-only peer">
            <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full"></div>
          </label>
        </li>
      </ul>
    </section>

    <section class="bg-white rounded-lg shadow p-5 mb-6">
      <h2 class="font-semibold mb-3">Scrape Companies</h2>
      <form @submit.prevent="addCompany" class="flex gap-2 mb-3">
        <input v-model="newCompanyURL" type="text" placeholder="Paste a careers URL (Greenhouse, Lever, Ashby, …)"
          class="flex-1 border rounded px-2 py-1.5 text-sm" />
        <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
      </form>
      <p v-if="companyError" class="text-xs text-red-600 mb-2">{{ companyError }}</p>
      <ul class="divide-y divide-gray-100">
        <li v-for="c in companies" :key="c.id" class="flex items-center justify-between py-2 text-sm">
          <div>
            <span class="font-medium">{{ c.name }}</span>
            <span class="ml-2 text-xs bg-gray-100 text-gray-600 rounded px-1.5 py-0.5">{{ c.ats_platform }}</span>
            <div class="text-xs mt-0.5" :class="c.last_scrape_status === 'error' ? 'text-red-600' : 'text-gray-400'">
              <template v-if="c.last_scrape_status === 'ok'">✅ {{ c.last_job_count }} jobs · {{ (c.last_scraped_at||'').slice(0,16) }}</template>
              <template v-else-if="c.last_scrape_status === 'error'">🔴 {{ c.last_scrape_error }} · {{ (c.last_scraped_at||'').slice(0,16) }}</template>
              <template v-else>Not scraped yet</template>
            </div>
          </div>
          <button @click="deleteCompany(c.id)" class="text-xs text-gray-400 hover:text-red-600">Delete</button>
        </li>
      </ul>
    </section>

    <section class="bg-white rounded-lg shadow p-5 mb-6">
      <h2 class="font-semibold mb-3">Target Roles</h2>
      <form @submit.prevent="addRole" class="flex gap-2 mb-3">
        <input v-model="newRole" type="text" placeholder="e.g. SRE, DevOps, Platform Engineer"
          class="flex-1 border rounded px-2 py-1.5 text-sm" />
        <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
      </form>
      <div class="flex flex-wrap gap-2">
        <span v-for="r in roles" :key="r.id" class="flex items-center gap-1 bg-gray-100 rounded-full px-2.5 py-1 text-xs">
          {{ r.keyword }}
          <button @click="deleteRole(r.id)" class="text-gray-400 hover:text-red-600">×</button>
        </span>
      </div>
    </section>

    <section class="bg-white rounded-lg shadow p-5 mb-6">
      <h2 class="font-semibold mb-3">Target Skills</h2>
      <p class="text-xs text-gray-500 mb-3">Jobs mentioning these skills in their title qualify even if the role title doesn't match. Multiple hits raise the skill score.</p>
      <form @submit.prevent="addSkill" class="flex gap-2 mb-3">
        <input v-model="newSkill" type="text" placeholder="e.g. Go, Rust, Kubernetes"
          class="flex-1 border rounded px-2 py-1.5 text-sm" />
        <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
      </form>
      <div class="flex flex-wrap gap-2">
        <span v-for="sk in skills" :key="sk.id" class="flex items-center gap-1 bg-indigo-50 border border-indigo-200 rounded-full px-2.5 py-1 text-xs text-indigo-700">
          {{ sk.keyword }}
          <button @click="deleteSkill(sk.id)" class="text-indigo-400 hover:text-red-600">×</button>
        </span>
      </div>
    </section>

    <section class="bg-white rounded-lg shadow p-5 mb-6">
      <h2 class="font-semibold mb-3">Location Preferences</h2>
      <label class="block text-xs text-gray-500 mb-1">Home timezone</label>
      <input v-model="homeTimezone" type="text" placeholder="UTC+7" class="border rounded px-2 py-1.5 text-sm w-32 mb-3" />
      <label class="block text-xs text-gray-500 mb-1">Acceptance notes (Claude reads this)</label>
      <textarea v-model="locationNotes" rows="4" class="w-full border rounded px-2 py-1.5 text-sm"
        placeholder="Accept: anywhere/worldwide; any role whose required timezone window includes UTC+7; APAC, Asia, Southeast Asia, Indonesia. Reject: US-only, EMEA-only, Americas-only."></textarea>
      <button @click="savePrefs" class="mt-2 bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Save preferences</button>
      <span v-if="prefsSaved" class="ml-2 text-xs text-green-600">Saved</span>
    </section>
  </div>

  <!-- Sources tab -->
  <div v-if="activeTab === 'sources'" class="max-w-md space-y-6">
    <div class="bg-white rounded-lg shadow divide-y">
      <div v-if="sources.length === 0" class="px-4 py-8 text-center text-gray-400 text-sm">No sources yet.</div>
      <div
        v-for="src in sources"
        :key="src.id"
        class="flex items-center justify-between px-4 py-3"
      >
        <span class="text-sm">{{ src.name }}</span>
        <div class="flex items-center gap-2">
          <template v-if="confirmDeleteId === src.id">
            <span class="text-xs text-gray-500">Remove "{{ src.name }}"?</span>
            <button @click="removeSource(src.id)" class="text-xs text-red-500 hover:text-red-700 font-medium">Yes</button>
            <button @click="confirmDeleteId = null" class="text-xs text-gray-400 hover:text-gray-600">Cancel</button>
          </template>
          <button
            v-else
            @click="confirmDeleteId = src.id"
            class="text-xs text-red-400 hover:text-red-600"
          >Remove</button>
        </div>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow p-4">
      <h2 class="font-semibold text-sm mb-3">Add Source</h2>
      <form class="flex gap-2" @submit.prevent="addSource">
        <input
          v-model="newSourceName"
          type="text"
          placeholder="e.g. AngelList"
          class="flex-1 border rounded px-3 py-1.5 text-sm"
        >
        <button
          type="submit"
          :disabled="!newSourceName.trim()"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Add</button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { CV_DEFAULT_KEY, CL_DEFAULT_KEY } from '../theme-defaults.js'

const route = useRoute()

const tabs = [
  { id: 'themes', label: 'Themes' },
  { id: 'sources', label: 'Sources' },
  { id: 'scraping', label: 'Job Scraping' },
]
const activeTab = ref(route.query.tab === 'sources' ? 'sources' : route.query.tab === 'scraping' ? 'scraping' : 'themes')

// Themes state
const themes       = ref([])
const previewTheme = ref('')
const selectedFile = ref(null)
const openDropdown = ref(null)
const defaultCV    = ref(localStorage.getItem(CV_DEFAULT_KEY) ?? '')
const defaultCL    = ref(localStorage.getItem(CL_DEFAULT_KEY) ?? '')

// Sources state
const sources        = ref([])
const newSourceName  = ref('')
const confirmDeleteId = ref(null)

// Scraping state
const boards        = ref([])
const companies     = ref([])
const roles         = ref([])
const skills        = ref([])
const newCompanyURL = ref('')
const newRole       = ref('')
const newSkill      = ref('')
const companyError  = ref('')
const homeTimezone  = ref('')
const locationNotes = ref('')
const prefsSaved    = ref(false)

async function loadScrape() {
  boards.value = await api.boards.list()
  companies.value = await api.scrape.companies()
  roles.value = await api.scrape.roles()
  skills.value = await api.scrape.skills()
  const p = await api.scrape.prefs()
  homeTimezone.value = p.home_timezone
  locationNotes.value = p.location_notes
}
async function toggleBoard(b) {
  await api.boards.toggle(b.id, !b.enabled)
  boards.value = await api.boards.list()
}
async function addCompany() {
  companyError.value = ''
  try {
    await api.scrape.addCompany(newCompanyURL.value)
    newCompanyURL.value = ''
    await loadScrape()
  } catch (e) {
    companyError.value = 'Could not add — unsupported or invalid careers URL.'
  }
}
async function deleteCompany(id) { await api.scrape.deleteCompany(id); await loadScrape() }
async function addRole() {
  if (!newRole.value.trim()) return
  await api.scrape.addRole(newRole.value.trim()); newRole.value = ''; await loadScrape()
}
async function deleteRole(id) { await api.scrape.deleteRole(id); await loadScrape() }
async function addSkill() {
  if (!newSkill.value.trim()) return
  await api.scrape.addSkill(newSkill.value.trim()); newSkill.value = ''; await loadScrape()
}
async function deleteSkill(id) { await api.scrape.deleteSkill(id); await loadScrape() }
async function savePrefs() {
  await api.scrape.updatePrefs(homeTimezone.value, locationNotes.value)
  prefsSaved.value = true
  setTimeout(() => { prefsSaved.value = false }, 1500)
}

onMounted(async () => {
  themes.value  = await api.themes.list()
  sources.value = await api.sources.list()
  await loadScrape()
})

function toggleDropdown(name) {
  openDropdown.value = openDropdown.value === name ? null : name
}

function setDefault(name, target) {
  if (target === 'cv' || target === 'both') {
    const next = defaultCV.value === name ? '' : name
    defaultCV.value = next
    next ? localStorage.setItem(CV_DEFAULT_KEY, next) : localStorage.removeItem(CV_DEFAULT_KEY)
  }
  if (target === 'cl' || target === 'both') {
    const next = defaultCL.value === name ? '' : name
    defaultCL.value = next
    next ? localStorage.setItem(CL_DEFAULT_KEY, next) : localStorage.removeItem(CL_DEFAULT_KEY)
  }
  openDropdown.value = null
}

function closeDropdown(e) {
  if (!e.target.closest('.relative')) openDropdown.value = null
}

onMounted(() => document.addEventListener('click', closeDropdown))
onBeforeUnmount(() => document.removeEventListener('click', closeDropdown))

function onFileChange(e) { selectedFile.value = e.target.files[0] ?? null }

async function upload() {
  if (!selectedFile.value) return
  await api.themes.upload(selectedFile.value)
  themes.value = await api.themes.list()
  selectedFile.value = null
}

async function addSource() {
  if (!newSourceName.value.trim()) return
  await api.sources.add(newSourceName.value.trim())
  sources.value = await api.sources.list()
  newSourceName.value = ''
}

async function removeSource(id) {
  await api.sources.delete(id)
  sources.value = sources.value.filter(s => s.id !== id)
  confirmDeleteId.value = null
}
</script>
