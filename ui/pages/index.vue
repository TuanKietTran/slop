<script setup lang="ts">
// Live task board: renders the pipeline DAG, one node per stage, colored by
// status. Subscribes to the control-plane SSE stream for updates.
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { ref, onMounted, computed } from 'vue'

const STAGES = ['Provision', 'Investigate', 'GenTests', 'Verify', 'SelfReview', 'BigReview', 'Notify']

const STATUS_COLORS: Record<string, string> = {
  pending: '#94a3b8',
  running: '#3b82f6',
  completed: '#22c55e',
  failed: '#ef4444',
}

const stageStatus = ref<Record<string, string>>(
  Object.fromEntries(STAGES.map((s) => [s, 'pending'])),
)

const nodes = computed(() =>
  STAGES.map((s, i) => ({
    id: s,
    position: { x: i * 200, y: 100 },
    data: { label: s },
    style: {
      background: STATUS_COLORS[stageStatus.value[s]] ?? STATUS_COLORS.pending,
      color: '#fff',
      border: '1px solid #334155',
      borderRadius: '8px',
      padding: '10px 18px',
      fontWeight: '600',
    },
  })),
)

const edges = computed(() =>
  STAGES.slice(1).map((s, i) => ({
    id: `${STAGES[i]}-${s}`,
    source: STAGES[i],
    target: s,
    animated: stageStatus.value[STAGES[i]] === 'running',
  })),
)

const tasks = ref<any[]>([])
const logLines = ref<string[]>([])

onMounted(async () => {
  // Load initial task list.
  try {
    const data = await $fetch<any[]>('/api/tasks')
    tasks.value = data
  } catch {}

  // Subscribe to SSE stream for live stage updates.
  const es = new EventSource('/api/stream')
  es.onmessage = (e) => {
    try {
      const { stage, status } = JSON.parse(e.data)
      if (stage && status) {
        stageStatus.value[stage] = status
        logLines.value.unshift(`[${new Date().toISOString()}] ${stage} → ${status}`)
        if (logLines.value.length > 50) logLines.value.pop()
      }
    } catch {}
  }
  es.onerror = () => {
    logLines.value.unshift(`[${new Date().toISOString()}] SSE connection error`)
  }
})
</script>

<template>
  <div class="flex flex-col h-screen bg-slate-900 text-white">
    <header class="px-6 py-4 border-b border-slate-700 flex items-center gap-4">
      <h1 class="text-xl font-bold">Test Agents Control Plane</h1>
      <div class="flex gap-3 ml-auto text-sm">
        <span v-for="(color, label) in { Pending: '#94a3b8', Running: '#3b82f6', Done: '#22c55e', Failed: '#ef4444' }" :key="label" class="flex items-center gap-1">
          <span class="inline-block w-3 h-3 rounded-full" :style="{ background: color }" />
          {{ label }}
        </span>
      </div>
    </header>

    <div class="flex flex-1 overflow-hidden">
      <!-- DAG panel -->
      <div class="flex-1 relative">
        <VueFlow
          :nodes="nodes"
          :edges="edges"
          fit-view-on-init
          class="bg-slate-900"
        />
      </div>

      <!-- Side panel: tasks + logs -->
      <div class="w-80 border-l border-slate-700 flex flex-col overflow-hidden">
        <div class="p-4 border-b border-slate-700">
          <h2 class="font-semibold mb-2 text-slate-300">Tasks ({{ tasks.length }})</h2>
          <div v-if="tasks.length === 0" class="text-slate-500 text-sm">No tasks yet</div>
          <div
            v-for="t in tasks"
            :key="t.id"
            class="mb-2 p-2 rounded bg-slate-800 text-xs"
          >
            <div class="font-mono truncate text-slate-400">{{ t.id }}</div>
            <div class="truncate text-slate-200">{{ t.target_site }}</div>
            <div class="flex gap-2 mt-1">
              <span class="px-1 rounded text-xs" :style="{ background: STATUS_COLORS[t.status] ?? '#475569' }">
                {{ t.status }}
              </span>
              <span class="text-slate-500">{{ t.stage }}</span>
            </div>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto p-4">
          <h2 class="font-semibold mb-2 text-slate-300">Live Log</h2>
          <div
            v-for="(line, i) in logLines"
            :key="i"
            class="text-xs font-mono text-slate-400 mb-1"
          >
            {{ line }}
          </div>
          <div v-if="logLines.length === 0" class="text-slate-500 text-xs">Waiting for events…</div>
        </div>
      </div>
    </div>
  </div>
</template>
