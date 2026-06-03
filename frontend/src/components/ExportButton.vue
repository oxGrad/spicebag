<template>
  <div class="relative inline-flex flex-col items-end gap-1">
    <div class="inline-flex rounded overflow-hidden border border-blue-600">
      <button
        @click="doExport"
        :disabled="exporting"
        class="bg-blue-600 text-white px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
      >
        {{ exporting ? "Exporting…" : "Export PDF" }}
      </button>
      <button
        @click="open = !open"
        class="bg-blue-600 text-white px-2 py-1.5 text-sm hover:bg-blue-700 border-l border-blue-500 transition-colors"
        aria-label="Select export method"
      >
        ▾
      </button>
    </div>

    <div
      v-if="open"
      class="absolute top-full mt-1 right-0 z-10 bg-white border rounded shadow-lg min-w-[160px]"
    >
      <button
        v-for="opt in methods"
        :key="opt.value"
        @click="select(opt.value)"
        class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center gap-2"
        :class="{ 'font-medium': method === opt.value }"
      >
        <span class="w-3 text-blue-600">{{ method === opt.value ? "✓" : "" }}</span>
        <span>{{ opt.label }}</span>
        <span class="ml-auto text-xs text-gray-400">{{ opt.hint }}</span>
      </button>
    </div>

    <span v-if="err" class="text-xs text-red-500 max-w-[220px] text-right">{{ err }}</span>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from "vue";
import { api } from "../api.js";

const props = defineProps({
  filePath: { type: String, required: true },
  theme: { type: String, default: "" },
});

const STORAGE_KEY = "spicebag:export-method";

const methods = [
  { value: "auto",       label: "Auto",           hint: "Gotenberg → Chrome" },
  { value: "gotenberg",  label: "Gotenberg",       hint: "needs Gotenberg" },
  { value: "chrome",     label: "Chrome (local)",  hint: "needs Chrome" },
];

const method = ref(localStorage.getItem(STORAGE_KEY) ?? "chrome");
const open = ref(false);
const exporting = ref(false);
const err = ref("");

function select(value) {
  method.value = value;
  localStorage.setItem(STORAGE_KEY, value);
  open.value = false;
}

function closeOnOutsideClick(e) {
  if (!e.target.closest(".relative")) open.value = false;
}

onMounted(() => document.addEventListener("click", closeOnOutsideClick));
onBeforeUnmount(() => document.removeEventListener("click", closeOnOutsideClick));

async function doExport() {
  err.value = "";
  exporting.value = true;
  try {
    const res = await api.export(props.filePath, props.theme, method.value);
    if (!res.ok) {
      const text = await res.text();
      err.value = text || "Export failed";
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = props.filePath.split("/").pop().replace(".html", ".pdf");
    a.click();
    URL.revokeObjectURL(url);
  } catch (e) {
    err.value = e.message;
  } finally {
    exporting.value = false;
  }
}
</script>
