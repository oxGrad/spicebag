<template>
  <div class="flex min-h-screen bg-gray-50 text-gray-900">
    <nav class="w-52 bg-gray-900 flex flex-col gap-0.5 py-4 px-2 fixed h-full shrink-0">
      <div class="flex items-center gap-2 px-2.5 py-1.5 mb-3 select-none">
        <span class="text-base leading-none">🌶️</span>
        <span class="text-sm font-semibold text-white tracking-tight">Spice Bag</span>
      </div>
      <RouterLink to="/"
        exact-active-class="bg-white/10 text-white font-medium"
        active-class=""
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Overview</RouterLink>
      <RouterLink to="/apps"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Applications</RouterLink>
      <RouterLink to="/cv"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >CVs</RouterLink>
      <RouterLink to="/stats"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Experience</RouterLink>
      <RouterLink to="/memory"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Memory</RouterLink>
      <RouterLink to="/settings"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Settings</RouterLink>

      <div class="mt-auto pt-3 border-t border-white/10">
        <button
          @click="showClaude = !showClaude"
          class="w-full text-left px-2.5 py-2 rounded-md text-sm transition-colors flex items-center gap-2"
          :class="showClaude
            ? 'bg-white/10 text-white font-medium'
            : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'"
          title="Toggle Claude Code (Ctrl+`)"
        >
          <span class="font-mono text-xs opacity-70">$_</span>
          Claude Code
        </button>
      </div>
    </nav>
    <main class="ml-52 flex-1 p-8 min-w-0">
      <RouterView />
    </main>
  </div>

  <ClaudePanel :open="showClaude" @close="showClaude = false" />
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import ClaudePanel from './components/ClaudePanel.vue'

const showClaude = ref(false)

function onKeyDown(e) {
  if (e.ctrlKey && e.key === '`') {
    e.preventDefault()
    showClaude.value = !showClaude.value
  }
}

onMounted(() => document.addEventListener('keydown', onKeyDown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeyDown))
</script>
