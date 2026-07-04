<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ForkDemo, GetProcessDetail, GetSnapshot, KillProcess, SetPriority, SpawnProcess } from '../wailsjs/go/main/App'
import CpuPanel from './components/CpuPanel.vue'
import ProcessPanel from './components/ProcessPanel.vue'
import ResourcePanels from './components/ResourcePanels.vue'
import SpawnDialog from './components/SpawnDialog.vue'
import type { ProcInfo, ProcessDetail, Snapshot, StatusEvent } from './types'
import { fmtDuration, parseArguments } from './utils'

const snapshot = ref<Snapshot | null>(null)
const selected = ref<ProcInfo | null>(null)
const detail = ref<ProcessDetail | null>(null)
const detailLoading = ref(false)
const error = ref('')
const paused = ref(false)
const refreshMs = ref(1000)
const showSpawn = ref(false)
const actionBusy = ref(false)
const events = ref<StatusEvent[]>([])
const histories = reactive({
  cpu: [] as number[], memory: [] as number[],
  netRX: [] as number[], netTX: [] as number[],
  diskRead: [] as number[], diskWrite: [] as number[],
})
let timer: number | null = null
let stopped = false
let polling = false
const MAX_POINTS = 60

function errorMessage(value: unknown) { return value instanceof Error ? value.message : String(value) }
function addEvent(message: string, isError = false) {
  events.value.unshift({ time: new Date().toLocaleTimeString('zh-CN', { hour12: false }), message, error: isError })
  events.value = events.value.slice(0, 8)
}
function pushHistory(values: number[], value: number) {
  values.push(Number.isFinite(value) ? value : 0)
  if (values.length > MAX_POINTS) values.shift()
}
function selectedNetwork(snap: Snapshot) {
  return snap.network.interfaces.find(item => item.name === snap.network.primary) || snap.network.interfaces[0]
}
function updateHistories(snap: Snapshot) {
  pushHistory(histories.cpu, snap.cpu.total)
  pushHistory(histories.memory, snap.memory.usage)
  const network = selectedNetwork(snap)
  pushHistory(histories.netRX, network?.rxRate || 0)
  pushHistory(histories.netTX, network?.txRate || 0)
  const disk = snap.disk.devices.reduce((sum, item) => ({ read: sum.read + item.readRate, write: sum.write + item.writeRate }), { read: 0, write: 0 })
  pushHistory(histories.diskRead, disk.read)
  pushHistory(histories.diskWrite, disk.write)
}
function schedule(delay = refreshMs.value) {
  if (!stopped) timer = window.setTimeout(poll, delay)
}
async function refreshDetail(process = selected.value) {
  if (!process) { detail.value = null; return }
  detailLoading.value = true
  try {
    detail.value = await GetProcessDetail(process.pid, process.startTime)
  } catch (cause) {
    detail.value = null
    addEvent(errorMessage(cause), true)
  } finally {
    detailLoading.value = false
  }
}
async function poll() {
  if (stopped || polling) return
  if (paused.value) { schedule(250); return }
  polling = true
  try {
    const result = await GetSnapshot()
    snapshot.value = result
    error.value = ''
    updateHistories(result)
    if (selected.value) {
      const fresh = result.procs.find(p => p.pid === selected.value?.pid && p.startTime === selected.value?.startTime)
      if (fresh) { selected.value = fresh; await refreshDetail(fresh) }
      else { selected.value = null; detail.value = null; addEvent('选中的进程已退出') }
    }
  } catch (cause) {
    error.value = errorMessage(cause)
    addEvent(error.value, true)
  } finally {
    polling = false
    schedule()
  }
}
function setRefresh(value: number) {
  refreshMs.value = value
  if (timer) window.clearTimeout(timer)
  schedule(50)
}
function togglePause() {
  paused.value = !paused.value
  addEvent(paused.value ? '采样已暂停' : '采样已恢复')
  if (!paused.value) { if (timer) window.clearTimeout(timer); schedule(0) }
}
function selectProcess(process: ProcInfo) {
  if (selected.value?.pid === process.pid && selected.value.startTime === process.startTime) return
  selected.value = process
  refreshDetail(process)
}
async function killProcess(process: ProcInfo, signal: number) {
  const label = signal === 9 ? 'SIGKILL' : 'SIGTERM'
  if (!window.confirm(`向 ${process.name} (PID ${process.pid}) 发送 ${label}？`)) return
  try {
    await KillProcess(process.pid, process.startTime, signal)
    addEvent(`${label} → ${process.name} (${process.pid})`)
  } catch (cause) { addEvent(errorMessage(cause), true) }
}
async function setNice(process: ProcInfo, value: number) {
  try {
    await SetPriority(process.pid, process.startTime, value)
    addEvent(`nice ${value} → ${process.name} (${process.pid})`)
  } catch (cause) { addEvent(errorMessage(cause), true) }
}
async function spawn(name: string, rawArgs: string) {
  actionBusy.value = true
  try {
    const pid = await SpawnProcess(name, parseArguments(rawArgs))
    addEvent(`exec ${name} → PID ${pid}`)
    showSpawn.value = false
  } catch (cause) { addEvent(errorMessage(cause), true) }
  finally { actionBusy.value = false }
}
async function runFork(seconds: number) {
  actionBusy.value = true
  try {
    const result = await ForkDemo(seconds)
    addEvent(`fork PID ${result.pid} ← parent ${result.parentPid} · ${result.implementation}`)
    showSpawn.value = false
  } catch (cause) { addEvent(errorMessage(cause), true) }
  finally { actionBusy.value = false }
}
function globalKey(event: KeyboardEvent) {
  if (document.activeElement instanceof HTMLInputElement) return
  if (event.key.toLowerCase() === 'p') togglePause()
  if (event.key.toLowerCase() === 'n') showSpawn.value = true
}
const latestEvent = computed(() => events.value[0])

onMounted(() => {
  stopped = false
  addEvent('采样器启动 · /proc online')
  poll()
  window.addEventListener('keydown', globalKey)
})
onUnmounted(() => {
  stopped = true
  if (timer) window.clearTimeout(timer)
  window.removeEventListener('keydown', globalKey)
})
</script>

<template>
  <div class="dashboard">
    <header class="status-header">
      <div class="brand"><span class="prompt">❯</span><b>btop-gui</b><span class="live-dot" :class="{ paused }" /></div>
      <div v-if="snapshot" class="host-info">
        <span>{{ snapshot.system.hostname || 'linux' }}</span><span>{{ snapshot.system.kernel || 'kernel N/A' }}</span>
        <span>up {{ fmtDuration(snapshot.uptime) }}</span><span>load {{ snapshot.loadAvg.map(v => v.toFixed(2)).join(' ') }}</span>
      </div>
      <div class="header-actions">
        <div class="rate-switch"><button v-for="rate in [1000, 2000, 5000]" :key="rate" :class="{ active: refreshMs === rate }" @click="setRefresh(rate)">{{ rate / 1000 }}s</button></div>
        <button :class="{ active: paused }" @click="togglePause">{{ paused ? 'resume' : 'pause' }} [P]</button>
        <button class="spawn" @click="showSpawn = true">new [N]</button>
      </div>
    </header>

    <div v-if="error" class="fatal-line"><b>collector error</b> {{ error }}</div>

    <template v-if="snapshot">
      <CpuPanel :cpu="snapshot.cpu" :system="snapshot.system" :load-avg="snapshot.loadAvg" :history="histories.cpu" />
      <main class="lower-grid">
        <ResourcePanels
          :memory="snapshot.memory" :disk="snapshot.disk" :network="snapshot.network"
          :memory-history="histories.memory"
          :disk-history="[histories.diskRead, histories.diskWrite]"
          :network-history="[histories.netRX, histories.netTX]"
        />
        <ProcessPanel
          :procs="snapshot.procs" :selected-pid="selected?.pid || null"
          :detail="detail" :detail-loading="detailLoading"
          @select="selectProcess" @kill="killProcess" @nice="setNice"
        />
      </main>
    </template>
    <div v-else class="boot-screen"><span>initialising collectors</span><i /><i /><i /></div>

    <footer class="event-rail" :class="{ error: latestEvent?.error }">
      <span class="rail-label">event</span><time>{{ latestEvent?.time || '--:--:--' }}</time>
      <span class="rail-message">{{ latestEvent?.message || 'ready' }}</span>
      <span class="keys">J/K select · F filter · T tree · P pause · N new</span>
    </footer>

    <SpawnDialog :open="showSpawn" :busy="actionBusy" @close="showSpawn = false" @spawn="spawn" @fork="runFork" />
  </div>
</template>

<style scoped>
.dashboard { display: grid; grid-template-rows: 36px minmax(190px, 27vh) minmax(0, 1fr) 25px; gap: 8px; width: 100%; height: 100vh; padding: 9px; overflow: hidden; background: var(--bg); }
.status-header { display: grid; grid-template-columns: auto 1fr auto; align-items: center; min-width: 0; border-bottom: 1px solid var(--line); padding: 0 4px 5px; font-size: 11.75px; }
.brand { display: flex; align-items: center; gap: 7px; color: var(--text); font-size: 14px; }.prompt { color: var(--green); }.live-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); box-shadow: 0 0 8px var(--green); }.live-dot.paused { background: var(--amber); box-shadow: none; }
.host-info { display: flex; justify-content: center; gap: 15px; min-width: 0; overflow: hidden; color: var(--muted); white-space: nowrap; }.host-info span:first-child { color: var(--cyan); }
.header-actions { display: flex; gap: 6px; align-items: center; }.header-actions button { border: 1px solid var(--line); border-radius: 3px; background: transparent; color: var(--muted); padding: 4px 7px; font: 500 10.5px var(--mono); cursor: pointer; }.header-actions button:hover,.header-actions button.active { border-color: var(--violet); color: var(--violet); }.header-actions .spawn { border-color: var(--cyan); color: var(--cyan); }
.rate-switch { display: flex; }.rate-switch button { border-radius: 0; }.rate-switch button:first-child { border-radius: 3px 0 0 3px; }.rate-switch button:last-child { border-radius: 0 3px 3px 0; }
.lower-grid { display: grid; grid-template-columns: minmax(360px, .82fr) minmax(580px, 1.45fr); gap: 8px; min-height: 0; }
.fatal-line { position: fixed; z-index: 20; top: 48px; left: 12px; right: 12px; border: 1px solid var(--rose); background: rgba(243,139,168,.12); color: var(--text); padding: 6px 10px; font: 12px Nunito, sans-serif; }.fatal-line b { color: var(--rose); margin-right: 8px; }
.event-rail { display: grid; grid-template-columns: auto auto 1fr auto; align-items: center; gap: 9px; min-width: 0; border-top: 1px solid var(--cyan); color: var(--muted); font-size: 10.75px; }.event-rail.error { border-color: var(--rose); }.rail-label { color: var(--cyan); font-weight: 700; text-transform: uppercase; }.event-rail.error .rail-label { color: var(--rose); }.event-rail time { color: var(--violet); }.rail-message { overflow: hidden; color: var(--text-soft); text-overflow: ellipsis; white-space: nowrap; }.keys { color: var(--muted); white-space: nowrap; }
.boot-screen { display: flex; align-items: center; justify-content: center; gap: 6px; color: var(--muted); }.boot-screen i { width: 5px; height: 5px; background: var(--cyan); animation: blink 1s infinite alternate; }.boot-screen i:nth-child(3) { animation-delay: .2s; }.boot-screen i:nth-child(4) { animation-delay: .4s; }
@keyframes blink { to { opacity: .2; } }
@media (max-width: 1050px) {
  .dashboard { display: block; height: auto; min-height: 100vh; overflow: auto; }.status-header { grid-template-columns: 1fr auto; margin-bottom: 10px; }.host-info { grid-column: 1 / -1; grid-row: 2; justify-content: flex-start; padding-top: 6px; }.lower-grid { grid-template-columns: 1fr; margin-top: 8px; }.event-rail { position: sticky; bottom: 0; height: 28px; margin-top: 8px; background: var(--bg); }.keys { display: none; }
}
@media (prefers-reduced-motion: reduce) { .boot-screen i { animation: none; } }
</style>
