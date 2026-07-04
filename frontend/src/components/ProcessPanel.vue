<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import PanelFrame from './PanelFrame.vue'
import type { ProcInfo, ProcessDetail } from '../types'
import { fmtBytes, fmtDuration } from '../utils'

type SortKey = 'pid' | 'name' | 'user' | 'state' | 'cpu' | 'rss' | 'threads' | 'nice'
type DisplayRow = ProcInfo & { depth: number }

const props = defineProps<{
  procs: ProcInfo[]; selectedPid: number | null
  detail: ProcessDetail | null; detailLoading: boolean
}>()
const emit = defineEmits<{
  select: [process: ProcInfo]
  kill: [process: ProcInfo, signal: number]
  nice: [process: ProcInfo, value: number]
}>()

const search = ref('')
const searchEl = ref<HTMLInputElement>()
const treeMode = ref(false)
const sortKey = ref<SortKey>('cpu')
const descending = ref(true)

function compare(a: ProcInfo, b: ProcInfo): number {
  const key = sortKey.value
  const av = a[key]
  const bv = b[key]
  let result = typeof av === 'string' && typeof bv === 'string' ? av.localeCompare(bv) : Number(av) - Number(bv)
  return descending.value ? -result : result
}

const rows = computed<DisplayRow[]>(() => {
  const query = search.value.trim().toLowerCase()
  if (!treeMode.value) {
    return props.procs
      .filter(p => !query || `${p.pid} ${p.name} ${p.user}`.toLowerCase().includes(query))
      .slice().sort(compare).map(p => ({ ...p, depth: 0 }))
  }
  const byPID = new Map(props.procs.map(p => [p.pid, p]))
  const visible = new Set<number>()
  if (query) {
    for (const process of props.procs) {
      if (!`${process.pid} ${process.name} ${process.user}`.toLowerCase().includes(query)) continue
      let current: ProcInfo | undefined = process
      const guard = new Set<number>()
      while (current && !guard.has(current.pid)) {
        guard.add(current.pid)
        visible.add(current.pid)
        current = byPID.get(current.ppid)
      }
    }
  }
  const children = new Map<number, ProcInfo[]>()
  for (const process of props.procs) {
    const list = children.get(process.ppid) || []
    list.push(process)
    children.set(process.ppid, list)
  }
  for (const list of children.values()) list.sort(compare)
  const roots = props.procs.filter(p => p.ppid === 0 || !byPID.has(p.ppid)).sort(compare)
  const result: DisplayRow[] = []
  const visited = new Set<number>()
  const walk = (process: ProcInfo, depth: number) => {
    if (visited.has(process.pid)) return
    visited.add(process.pid)
    if (!query || visible.has(process.pid)) result.push({ ...process, depth })
    for (const child of children.get(process.pid) || []) walk(child, depth + 1)
  }
  roots.forEach(root => walk(root, 0))
  props.procs.forEach(process => walk(process, 0))
  return result
})

const selected = computed(() => props.procs.find(p => p.pid === props.selectedPid) || null)

function setSort(key: SortKey) {
  if (sortKey.value === key) descending.value = !descending.value
  else { sortKey.value = key; descending.value = key !== 'pid' && key !== 'name' && key !== 'user' }
}
function sortMark(key: SortKey) { return sortKey.value === key ? (descending.value ? '▼' : '▲') : '' }
function stateLabel(state: string) {
  return ({ R: 'run', S: 'sleep', D: 'wait', Z: 'zombie', T: 'stop', I: 'idle' } as Record<string, string>)[state] || state
}
function selectAt(index: number) {
  const row = rows.value[Math.max(0, Math.min(rows.value.length - 1, index))]
  if (row) emit('select', row)
}
function move(delta: number) {
  const index = rows.value.findIndex(p => p.pid === props.selectedPid)
  selectAt(index < 0 ? 0 : index + delta)
}
function onKey(event: KeyboardEvent) {
  const inputFocused = document.activeElement instanceof HTMLInputElement
  if (event.key.toLowerCase() === 'f' && !inputFocused) {
    event.preventDefault(); searchEl.value?.focus(); return
  }
  if (inputFocused) {
    if (event.key === 'Escape') { search.value = ''; searchEl.value?.blur() }
    return
  }
  if (event.key === 'ArrowDown' || event.key.toLowerCase() === 'j') { event.preventDefault(); move(1) }
  else if (event.key === 'ArrowUp' || event.key.toLowerCase() === 'k') { event.preventDefault(); move(-1) }
  else if (event.key.toLowerCase() === 't') treeMode.value = !treeMode.value
}

watch(rows, value => {
  if (value.length && !value.some(p => p.pid === props.selectedPid)) emit('select', value[0])
}, { immediate: true })
onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <PanelFrame title="processes" tone="cyan" :meta="`${rows.length}/${procs.length}`" class="process-panel">
    <div class="proc-tools">
      <button :class="{ active: !treeMode }" @click="treeMode = false">flat</button>
      <button :class="{ active: treeMode }" @click="treeMode = true">tree</button>
      <input ref="searchEl" v-model="search" placeholder="filter pid / name / user [F]" spellcheck="false">
    </div>
    <div class="table-scroll">
      <table>
        <thead><tr>
          <th class="num" @click="setSort('pid')">PID {{ sortMark('pid') }}</th>
          <th @click="setSort('name')">COMMAND {{ sortMark('name') }}</th>
          <th @click="setSort('user')">USER {{ sortMark('user') }}</th>
          <th @click="setSort('state')">S {{ sortMark('state') }}</th>
          <th class="num" @click="setSort('threads')">THR {{ sortMark('threads') }}</th>
          <th class="num" @click="setSort('rss')">MEM {{ sortMark('rss') }}</th>
          <th class="num" @click="setSort('cpu')">CPU% {{ sortMark('cpu') }}</th>
        </tr></thead>
        <tbody>
          <tr v-for="process in rows" :key="`${process.pid}-${process.startTime}`"
              :class="{ selected: process.pid === selectedPid }" @click="emit('select', process)">
            <td class="num muted">{{ process.pid }}</td>
            <td class="command" :style="{ paddingLeft: `${8 + process.depth * 13}px` }">
              <span v-if="treeMode" class="branch">{{ process.depth ? '└' : '─' }}</span>{{ process.name }}
            </td>
            <td class="user">{{ process.user || process.uid }}</td>
            <td><span class="state" :class="`state-${process.state.toLowerCase()}`">{{ stateLabel(process.state) }}</span></td>
            <td class="num muted">{{ process.threads }}</td>
            <td class="num">{{ fmtBytes(process.rss) }}</td>
            <td class="num cpu" :class="{ hot: process.cpu >= 50 }">{{ process.cpu.toFixed(1) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="proc-detail" :class="{ loading: detailLoading }">
      <template v-if="selected">
        <div class="detail-top">
          <b>{{ selected.name }}</b><span>pid {{ selected.pid }}</span><span>ppid {{ selected.ppid }}</span>
          <span>nice {{ selected.nice }}</span><span v-if="detail">up {{ fmtDuration(detail.elapsed) }}</span>
        </div>
        <div v-if="detail" class="detail-grid">
          <span class="commandline" :title="detail.command">{{ detail.command }}</span>
          <span>I/O <b>R {{ fmtBytes(detail.readBytes) }} · W {{ fmtBytes(detail.writeBytes) }}</b></span>
          <span>FD <b>{{ detail.fdCount < 0 ? 'N/A' : detail.fdCount }}</b></span>
          <span>CTX <b>{{ detail.voluntarySwitches + detail.involuntarySwitches }}</b></span>
        </div>
        <div class="actions">
          <button @click="emit('nice', selected, selected.nice + 1)" :disabled="selected.nice >= 19">nice +</button>
          <button @click="emit('nice', selected, selected.nice - 1)" :disabled="selected.nice <= -20">nice −</button>
          <button class="warn" @click="emit('kill', selected, 15)">term</button>
          <button class="danger" @click="emit('kill', selected, 9)">kill</button>
        </div>
      </template>
      <span v-else class="muted">Select a process with mouse or J/K</span>
    </div>
  </PanelFrame>
</template>

<style scoped>
.process-panel { display: grid; grid-template-rows: auto 1fr auto; height: 100%; }
.proc-tools { display: flex; gap: 5px; align-items: center; padding-bottom: 6px; border-bottom: 1px solid var(--line); }
button, input { font: inherit; }
.proc-tools button, .actions button { border: 1px solid var(--line); border-radius: 3px; background: transparent; color: var(--muted); font-size: 10.5px; padding: 3px 7px; cursor: pointer; }
.proc-tools button.active { border-color: var(--cyan); color: var(--cyan); }
.proc-tools input { flex: 1; min-width: 80px; border: 0; border-bottom: 1px solid var(--line); outline: 0; background: transparent; color: var(--text); padding: 4px 7px; font-size: 11.5px; }
.proc-tools input:focus { border-color: var(--cyan); }
.table-scroll { min-height: 0; overflow: auto; }
table { width: 100%; border-collapse: collapse; table-layout: fixed; font-size: 11.25px; }
th { position: sticky; top: 0; z-index: 2; height: 27px; background: var(--panel); color: var(--muted); text-align: left; font-size: 9.75px; font-weight: 600; cursor: pointer; }
th:nth-child(1) { width: 61px; } th:nth-child(2) { width: 31%; } th:nth-child(3) { width: 76px; }
th:nth-child(4) { width: 48px; } th:nth-child(5) { width: 39px; } th:nth-child(6) { width: 67px; } th:nth-child(7) { width: 51px; }
td { height: 25px; border-bottom: 1px solid rgba(122,130,160,.12); color: var(--text-soft); overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
tr { cursor: default; }
tbody tr:hover { background: rgba(123,223,242,.045); }
tbody tr.selected { background: rgba(123,223,242,.12); box-shadow: inset 2px 0 var(--cyan); }
.num { text-align: right; padding-right: 7px; font-variant-numeric: tabular-nums; }
.command { color: var(--text); }
.branch { color: var(--violet); margin-right: 5px; }
.user { color: var(--cyan); }
.cpu { color: var(--green); font-weight: 700; }.cpu.hot { color: var(--rose); }
.state { color: var(--muted); }.state-r { color: var(--green); }.state-d,.state-z { color: var(--rose); }.state-t { color: var(--amber); }
.muted { color: var(--muted); }
.proc-detail { min-height: 75px; padding: 6px 2px 0; border-top: 1px solid var(--line); font-size: 10.75px; }
.proc-detail.loading { opacity: .65; }
.detail-top { display: flex; gap: 9px; align-items: baseline; color: var(--muted); }.detail-top b { color: var(--cyan); font-size: 12.5px; }
.detail-grid { display: grid; grid-template-columns: minmax(120px, 1fr) auto auto auto; gap: 8px; margin-top: 5px; color: var(--muted); }
.detail-grid b { color: var(--text); font-weight: 500; }.commandline { overflow: hidden; color: var(--text-soft); text-overflow: ellipsis; }
.actions { display: flex; justify-content: flex-end; gap: 5px; margin-top: 5px; }
.actions button:hover { color: var(--cyan); border-color: var(--cyan); }.actions .warn:hover { color: var(--amber); border-color: var(--amber); }.actions .danger:hover { color: var(--rose); border-color: var(--rose); }
.actions button:disabled { opacity: .3; }
@media (max-width: 1050px) { .process-panel { min-height: 500px; } }
</style>
