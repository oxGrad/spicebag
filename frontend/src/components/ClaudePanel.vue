<template>
  <Teleport to="body">
    <div
      class="fixed inset-y-0 right-0 flex flex-col bg-gray-950 border-l border-white/10 shadow-2xl transition-transform duration-200"
      :class="open ? 'translate-x-0 ease-out' : 'translate-x-full ease-in'"
      style="width: 540px; z-index: 40;"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-2.5 border-b border-white/10 shrink-0 select-none">
        <div class="flex items-center gap-2.5">
          <span class="font-mono text-xs text-gray-500 bg-white/5 px-1.5 py-0.5 rounded">$_</span>
          <span class="text-sm text-white font-medium">Claude Code</span>
          <span class="text-xs text-gray-600">~/.config/spicebag</span>
        </div>
        <div class="flex items-center gap-2">
          <span v-if="status === 'connecting'" class="text-xs text-yellow-500/80">connecting</span>
          <template v-else-if="status === 'exited'">
            <span class="text-xs text-red-400/80">session ended</span>
            <button
              @click="restart"
              class="text-xs px-2 py-0.5 border border-white/15 rounded text-gray-400 hover:text-white hover:border-white/30 transition-colors"
            >Restart</button>
          </template>
          <button
            @click="$emit('close')"
            class="text-gray-600 hover:text-gray-300 text-xl leading-none ml-1 transition-colors"
            title="Close (Ctrl+`)"
          >×</button>
        </div>
      </div>

      <!-- Terminal -->
      <div ref="termContainer" class="flex-1 min-h-0 overflow-hidden p-1.5" />
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const props = defineProps({ open: Boolean })
const emit = defineEmits(['close'])

const termContainer = ref(null)
const status = ref('disconnected')

// Session ID persists for the lifetime of this component instance.
// Closing the panel disconnects the WebSocket but the server keeps the session alive.
const sessionId = ref(crypto.randomUUID())

let term = null
let fitAddon = null
let ws = null
let resizeObserver = null
let everOpened = false

function initTerminal() {
  if (term) return
  term = new Terminal({
    cursorBlink: true,
    fontFamily: '"Cascadia Code", "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace',
    fontSize: 13,
    lineHeight: 1.45,
    theme: {
      background:          '#030712',
      foreground:          '#d1d5db',
      cursor:              '#6b7280',
      cursorAccent:        '#030712',
      selectionBackground: '#1f2937',
      black:   '#1f2937', brightBlack:   '#4b5563',
      red:     '#f87171', brightRed:     '#fca5a5',
      green:   '#4ade80', brightGreen:   '#86efac',
      yellow:  '#fbbf24', brightYellow:  '#fde68a',
      blue:    '#60a5fa', brightBlue:    '#93c5fd',
      magenta: '#c084fc', brightMagenta: '#d8b4fe',
      cyan:    '#22d3ee', brightCyan:    '#67e8f9',
      white:   '#d1d5db', brightWhite:   '#f9fafb',
    },
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termContainer.value)
  fitAddon.fit()

  term.onData(data => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(new TextEncoder().encode(data))
    }
  })

  resizeObserver = new ResizeObserver(() => {
    fitAddon.fit()
    sendResize()
  })
  resizeObserver.observe(termContainer.value)
}

function sendResize() {
  if (ws?.readyState === WebSocket.OPEN && term) {
    ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
  }
}

function connect() {
  if (ws) { ws.close(); ws = null }
  status.value = 'connecting'
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${proto}//${location.host}/ws/claude?session=${sessionId.value}`)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    status.value = 'connected'
    sendResize()
  }

  ws.onmessage = e => {
    if (!term) return
    if (e.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(e.data))
    } else {
      try {
        const msg = JSON.parse(e.data)
        if (msg.type === 'exit') {
          status.value = 'exited'
          term.write('\r\n\x1b[2m\r\n  session ended  \r\n\x1b[0m\r\n')
        } else if (msg.type === 'error') {
          term.write(`\r\n\x1b[31m${msg.detail}\x1b[0m\r\n`)
          status.value = 'exited'
        }
      } catch {}
    }
  }

  ws.onclose = () => {
    if (status.value === 'connected') status.value = 'disconnected'
  }
}

function disconnect() {
  if (ws) { ws.close(); ws = null }
}

async function restart() {
  // Tell server to kill the current session, then start fresh.
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'kill' }))
  }
  disconnect()
  sessionId.value = crypto.randomUUID()
  term?.clear()
  await nextTick()
  connect()
}

watch(() => props.open, async val => {
  if (val) {
    if (!everOpened) {
      everOpened = true
      await nextTick()
      initTerminal()
    }
    // Reconnect if not already connected (panel was closed and reopened)
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      connect()
    }
    setTimeout(() => term?.focus(), 210)
  } else {
    // Close WebSocket — server keeps the session alive and buffers output.
    disconnect()
    term?.blur()
  }
})

onBeforeUnmount(() => {
  disconnect()
  resizeObserver?.disconnect()
  term?.dispose()
})
</script>
