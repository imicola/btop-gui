<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import * as echarts from 'echarts'
import { GetSnapshot, KillProcess, SetPriority, SpawnProcess } from '../wailsjs/go/main/App'

interface CPUUsage { total: number; perCore: number[]; coreCount: number }
interface MemoryInfo {
  total: number; free: number; available: number; used: number; usage: number
  cached: number; buffers: number
  swapTotal: number; swapFree: number; swapUsage: number
}
interface ProcInfo {
  pid: number; name: string; state: string; ppid: number; nice: number
  cpu: number; rss: number; vsize: number; threads: number; startTime: number
}
interface Snapshot {
  timestamp: number
  cpu: CPUUsage
  memory: MemoryInfo
  loadAvg: number[]
  uptime: number
  procs: ProcInfo[]
  procCount: number
}

const snap = ref<Snapshot | null>(null)
const error = ref<string>('')
let timer: number | null = null
let stopped = false

type SortKey = 'pid' | 'name' | 'state' | 'nice' | 'cpu' | 'rss' | 'ppid'
const sortKey = ref<SortKey>('cpu')
const sortDesc = ref(true)
const searchQuery = ref('')

const sortedProcs = computed(() => {
  if (!snap.value) return []
  let arr = [...snap.value.procs]
  // 搜索过滤（按 PID 数字或名称子串）
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    arr = arr.filter(p =>
      String(p.pid).includes(q) || p.name.toLowerCase().includes(q)
    )
  }
  const key = sortKey.value
  const desc = sortDesc.value
  arr.sort((a, b) => {
    if (key === 'name' || key === 'state') {
      const av = a[key] as string
      const bv = b[key] as string
      return desc ? (av > bv ? -1 : av < bv ? 1 : 0) : (av < bv ? -1 : av > bv ? 1 : 0)
    }
    const av = a[key] as number
    const bv = b[key] as number
    return desc ? bv - av : av - bv
  })
  return arr
})

function setSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDesc.value = !sortDesc.value
  } else {
    sortKey.value = key
    sortDesc.value = key !== 'pid'
  }
}

function sortMark(key: SortKey): string {
  if (sortKey.value !== key) return ''
  return sortDesc.value ? ' ↓' : ' ↑'
}

// === 历史曲线（ECharts） ===
const cpuChartEl = ref<HTMLDivElement>()
const memChartEl = ref<HTMLDivElement>()
let cpuChart: echarts.ECharts | null = null
let memChart: echarts.ECharts | null = null

const cpuHistory: number[] = []
const memHistory: number[] = []
const MAX_POINTS = 60

// 构造渐变填充折线配置，color 为主色 hex（如 '#58a6ff'）
function makeChartOption(color: string) {
  return {
    grid: { left: 0, right: 0, top: 4, bottom: 0 },
    xAxis: { type: 'category', show: false, boundaryGap: false },
    yAxis: { type: 'value', show: false, min: 0, max: 100 },
    series: [{
      type: 'line',
      data: [] as number[],
      smooth: true,
      symbol: 'none',
      lineStyle: { color, width: 1.5 },
      areaStyle: {
        color: {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: color + 'cc' },
            { offset: 1, color: color + '08' },
          ],
        },
      },
    }],
    animation: false,
  }
}

// Wails 的 WebKitGTK 在 WSLg 软件渲染下 canvas 可能不工作，用 SVG renderer 走纯 DOM 渲染
function initCharts() {
  if (cpuChartEl.value) {
    cpuChart = echarts.init(cpuChartEl.value, undefined, { renderer: 'svg' })
    cpuChart.setOption(makeChartOption('#58a6ff'))
  }
  if (memChartEl.value) {
    memChart = echarts.init(memChartEl.value, undefined, { renderer: 'svg' })
    memChart.setOption(makeChartOption('#bc8cff'))
  }
  setTimeout(() => {
    cpuChart?.resize()
    memChart?.resize()
  }, 100)
}

function pushHistory(arr: number[], val: number) {
  arr.push(val)
  if (arr.length > MAX_POINTS) arr.shift()
}

function updateCharts() {
  if (snap.value) {
    pushHistory(cpuHistory, snap.value.cpu.total)
    pushHistory(memHistory, snap.value.memory.usage)
  }
  cpuChart?.setOption({ series: [{ data: cpuHistory }] })
  memChart?.setOption({ series: [{ data: memHistory }] })
}

function onResize() {
  cpuChart?.resize()
  memChart?.resize()
}

function fmtBytes(b: number): string {
  if (b >= 1024 * 1024 * 1024) return (b / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (b >= 1024 * 1024) return (b / 1024 / 1024).toFixed(1) + ' MB'
  if (b >= 1024) return (b / 1024).toFixed(1) + ' KB'
  return b + ' B'
}

function fmtKB(kb: number): string {
  return fmtBytes(kb * 1024)
}

function fmtUptime(s: number): string {
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  return `${d}d ${h}h ${m}m`
}

function cpuColor(cpu: number): string {
  if (cpu >= 50) return 'var(--red)'
  if (cpu >= 10) return 'var(--yellow)'
  return 'var(--green)'
}

function stateClass(s: string): string {
  return 'state-' + (s || '').toLowerCase()
}

function stateLabel(s: string): string {
  const map: Record<string, string> = {
    r: 'Running', s: 'Sleeping', d: 'DiskWait', z: 'Zombie', t: 'Stopped',
  }
  return map[(s || '').toLowerCase()] || s
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

async function poll() {
  try {
    const r = await GetSnapshot()
    snap.value = r
    error.value = ''
    updateCharts()
  } catch (e: unknown) {
    error.value = errorMessage(e)
  } finally {
    // 等上一次 IPC 完成后再计时，避免后端变慢时 setInterval 不断堆积请求。
    if (!stopped) timer = window.setTimeout(poll, 1000)
  }
}

// === 进程操作（kill / setpriority）===
const flashMsg = ref('')
const flashErr = ref(false)
let flashTimer: number | null = null

function flash(msg: string, isErr = false) {
  flashMsg.value = msg
  flashErr.value = isErr
  if (flashTimer) window.clearTimeout(flashTimer)
  flashTimer = window.setTimeout(() => { flashMsg.value = '' }, 3000)
}

// 信号编号与 kill(2) 一致：15=SIGTERM（优雅终止）, 9=SIGKILL（强制杀死）
const SIG = { TERM: 15, KILL: 9 } as const

async function killProc(pid: number, startTime: number, sig: number, name: string) {
  const sigName = sig === SIG.TERM ? 'SIGTERM' : sig === SIG.KILL ? 'SIGKILL' : `signal ${sig}`
  if (!window.confirm(`确认向进程 "${name}" (PID ${pid}) 发送 ${sigName}？`)) return
  try {
    await KillProcess(pid, startTime, sig)
    flash(`已发送 ${sigName} → ${name} (${pid})`)
  } catch (e: unknown) {
    flash(`操作失败: ${errorMessage(e)}`, true)
  }
}

async function adjustNice(pid: number, startTime: number, nice: number, name: string) {
  if (nice < -20 || nice > 19) return
  try {
    await SetPriority(pid, startTime, nice)
    flash(`nice → ${nice} (${name})`)
  } catch (e: unknown) {
    flash(`调整优先级失败: ${errorMessage(e)}`, true)
  }
}

// === 进程创建（fork + exec）===
const showSpawn = ref(false)
const spawnName = ref('')
const spawnArgs = ref('')

async function doSpawn() {
  const name = spawnName.value.trim()
  if (!name) return
  const args = spawnArgs.value.trim() ? spawnArgs.value.trim().split(/\s+/) : []
  try {
    const pid = await SpawnProcess(name, args)
    flash(`已启动 ${name} → PID ${pid}`)
    showSpawn.value = false
    spawnName.value = ''
    spawnArgs.value = ''
  } catch (e: unknown) {
    flash(`启动失败: ${errorMessage(e)}`, true)
  }
}

// snap 初始为 null，chart 容器在 v-if="snap" 块内不会立即渲染，
// 必须等第一次数据到达、DOM 更新后再初始化 ECharts
watch(snap, (val) => {
  if (val && !cpuChart && !memChart) {
    nextTick(initCharts)
  }
})

onMounted(() => {
  stopped = false
  poll()
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  stopped = true
  if (timer) window.clearInterval(timer)
  if (flashTimer) window.clearTimeout(flashTimer)
  window.removeEventListener('resize', onResize)
  cpuChart?.dispose()
  memChart?.dispose()
})
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="title">
        <span class="logo">◆</span> btop-gui
      </div>
      <div class="topbar-right">
        <button class="topbar-btn" @click="showSpawn = true">＋ 启动程序</button>
        <div v-if="snap" class="meta">
          uptime <b>{{ fmtUptime(snap.uptime) }}</b>
          · load <b>{{ snap.loadAvg[0].toFixed(2) }}</b> / {{ snap.loadAvg[1].toFixed(2) }} / {{ snap.loadAvg[2].toFixed(2) }}
          · procs <b>{{ snap.procCount }}</b>
        </div>
      </div>
    </header>

    <div v-if="error" class="error-banner">⚠ {{ error }}</div>
    <div v-if="flashMsg" class="flash-banner" :class="{ err: flashErr }">{{ flashMsg }}</div>

    <main v-if="snap" class="main">
      <section class="card">
        <div class="card-head">
          <span class="card-title">CPU</span>
          <span class="card-sub">{{ snap.cpu.coreCount }} cores</span>
        </div>
        <div class="metric-row">
          <div class="big">{{ snap.cpu.total.toFixed(1) }}<span class="unit">%</span></div>
          <div ref="cpuChartEl" class="chart"></div>
        </div>
        <div class="cores">
          <div v-for="(c, i) in snap.cpu.perCore" :key="i" class="core">
            <div class="core-bar">
              <div class="fill" :style="{ height: Math.max(2, c) + '%' }"></div>
            </div>
            <span class="core-idx">{{ i }}</span>
          </div>
        </div>
      </section>

      <section class="card">
        <div class="card-head">
          <span class="card-title">Memory</span>
        </div>
        <div class="metric-row">
          <div class="big">{{ snap.memory.usage.toFixed(1) }}<span class="unit">%</span></div>
          <div ref="memChartEl" class="chart"></div>
        </div>
        <div class="sub">{{ fmtKB(snap.memory.used) }} / {{ fmtKB(snap.memory.total) }}</div>
        <div class="hbar"><div class="fill" :style="{ width: snap.memory.usage + '%' }"></div></div>
        <div class="sub muted" v-if="snap.memory.swapTotal > 0">
          swap {{ snap.memory.swapUsage.toFixed(1) }}%
          ({{ fmtKB(snap.memory.swapTotal - snap.memory.swapFree) }} / {{ fmtKB(snap.memory.swapTotal) }})
        </div>
      </section>
    </main>

    <section v-if="snap" class="proc-section">
      <div class="card-head">
        <span class="card-title">Processes</span>
        <div class="proc-head-right">
          <input
            v-model="searchQuery"
            class="proc-search"
            type="text"
            placeholder="搜索 PID / 名称..."
            spellcheck="false"
          >
          <span class="card-sub">showing {{ sortedProcs.length }} of {{ snap.procCount }}</span>
        </div>
      </div>
      <div class="table-wrap">
        <table class="proc-table">
          <thead>
            <tr>
              <th class="r sortable" @click="setSort('pid')">PID{{ sortMark('pid') }}</th>
              <th class="l sortable" @click="setSort('name')">Name{{ sortMark('name') }}</th>
              <th class="sortable" @click="setSort('state')">State{{ sortMark('state') }}</th>
              <th class="r sortable" @click="setSort('cpu')">CPU%{{ sortMark('cpu') }}</th>
              <th class="r sortable" @click="setSort('rss')">MEM{{ sortMark('rss') }}</th>
              <th class="r sortable" @click="setSort('nice')">Nice{{ sortMark('nice') }}</th>
              <th class="r">Threads</th>
              <th class="r sortable" @click="setSort('ppid')">PPID{{ sortMark('ppid') }}</th>
              <th class="actions-col">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in sortedProcs" :key="p.pid">
              <td class="r">{{ p.pid }}</td>
              <td class="l name">{{ p.name }}</td>
              <td><span class="state" :class="stateClass(p.state)">{{ p.state }} · {{ stateLabel(p.state) }}</span></td>
              <td class="r cpu-cell" :style="{ color: cpuColor(p.cpu) }"><b>{{ p.cpu.toFixed(1) }}</b></td>
              <td class="r">{{ fmtBytes(p.rss) }}</td>
              <td class="r" :class="{ pos: p.nice > 0, neg: p.nice < 0 }">{{ p.nice }}</td>
              <td class="r muted">{{ p.threads }}</td>
              <td class="r muted">{{ p.ppid }}</td>
              <td class="actions">
                <button class="btn btn-term" title="SIGTERM 优雅终止" @click="killProc(p.pid, p.startTime, SIG.TERM, p.name)">TERM</button>
                <button class="btn btn-kill" title="SIGKILL 强制杀死" @click="killProc(p.pid, p.startTime, SIG.KILL, p.name)">KILL</button>
                <button class="btn btn-nice" title="降低优先级 (nice +1)" :disabled="p.nice >= 19" @click="adjustNice(p.pid, p.startTime, p.nice + 1, p.name)">−</button>
                <span class="nice-val">{{ p.nice }}</span>
                <button class="btn btn-nice" title="提高优先级 (nice -1)" :disabled="p.nice <= -20" @click="adjustNice(p.pid, p.startTime, p.nice - 1, p.name)">+</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 进程创建对话框（fork + exec） -->
    <div v-if="showSpawn" class="spawn-overlay" @click.self="showSpawn = false">
      <div class="spawn-dialog">
        <div class="spawn-title">启动新进程 <span class="spawn-sub">fork + execve</span></div>
        <label class="spawn-label">程序名 / 路径</label>
        <input
          v-model="spawnName"
          class="spawn-input"
          placeholder="如: sleep / python3 / ls / xeyes"
          spellcheck="false"
          @keyup.enter="doSpawn"
        >
        <label class="spawn-label">参数（空格分隔）</label>
        <input
          v-model="spawnArgs"
          class="spawn-input"
          placeholder="如: 30  或  -m http.server 8080"
          spellcheck="false"
          @keyup.enter="doSpawn"
        >
        <div class="spawn-actions">
          <button class="btn" @click="showSpawn = false">取消</button>
          <button class="btn btn-primary" @click="doSpawn">启动</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: 16px;
  gap: 14px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
  padding-bottom: 10px;
}
.title {
  font-size: 17px; font-weight: 600; color: var(--text);
  letter-spacing: 0.5px;
}
.logo { color: var(--green); margin-right: 2px; }
.meta { color: var(--muted); font-size: 13px; }
.meta b { color: var(--text-secondary); font-weight: 500; }

.error-banner {
  background: var(--red); color: #fff;
  padding: 8px 14px; border-radius: 6px; font-weight: 600; font-size: 13px;
}

.main {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 14px;
  align-items: stretch;
}

.card {
  background: var(--card);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow);
}
.card-head {
  display: flex; justify-content: space-between; align-items: baseline;
  margin-bottom: 10px;
}
.card-title {
  color: var(--muted); font-size: 12px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 1.2px;
}
.card-sub { color: var(--muted); font-size: 12px; }

.metric-row {
  display: flex;
  align-items: center;
  gap: 14px;
}
.big {
  font-size: 36px; font-weight: 700; color: var(--accent); line-height: 1;
  flex-shrink: 0; font-variant-numeric: tabular-nums;
}
.unit { font-size: 16px; color: var(--muted); margin-left: 3px; font-weight: 500; }
.chart {
  flex: 1;
  height: 60px;
  min-width: 0;
}
.sub { font-size: 12px; color: var(--text-secondary); margin-top: 8px; }
.muted { color: var(--muted); }

.cores {
  display: flex; gap: 4px; align-items: flex-end; height: 52px;
  margin-top: 12px; border-top: 1px solid var(--border-subtle); padding-top: 10px;
}
.core {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; gap: 3px; height: 100%;
}
.core-bar {
  flex: 1; width: 100%; background: rgba(255, 255, 255, 0.04);
  border-radius: 2px; position: relative; overflow: hidden;
}
.core-bar .fill {
  position: absolute; bottom: 0; left: 0; right: 0;
  background: linear-gradient(to top, var(--green), var(--yellow) 70%, var(--red) 100%);
  transition: height 0.4s ease;
  border-radius: 2px;
}
.core-idx { font-size: 10px; color: var(--muted); }

.hbar {
  height: 7px; background: rgba(255, 255, 255, 0.04);
  border-radius: 4px; margin-top: 8px; overflow: hidden;
}
.hbar .fill {
  height: 100%;
  background: linear-gradient(to right, var(--green), var(--yellow) 70%, var(--red) 100%);
  transition: width 0.4s ease;
  border-radius: 4px;
}

.proc-section {
  flex: 1; background: var(--card);
  border: 1px solid var(--border-subtle); border-radius: 8px;
  padding: 16px; display: flex; flex-direction: column;
  overflow: hidden; min-height: 0;
  box-shadow: var(--shadow);
}
.table-wrap { flex: 1; overflow: auto; margin-top: 10px; min-height: 0; }
.proc-table {
  width: 100%; border-collapse: collapse; font-size: 13px;
}
.proc-table thead th {
  position: sticky; top: 0; z-index: 1;
  background: var(--card);
  text-align: center; color: var(--muted);
  font-weight: 600; font-size: 11px;
  text-transform: uppercase; letter-spacing: 0.8px;
  padding: 8px 12px; border-bottom: 1px solid var(--border);
}
.proc-table th.l { text-align: left; }
.proc-table th.r { text-align: right; }
.proc-table th.sortable { cursor: pointer; user-select: none; transition: color 0.15s; }
.proc-table th.sortable:hover { color: var(--accent); }
.proc-table td {
  padding: 5px 12px;
  border-bottom: 1px solid var(--border-subtle);
  text-align: center;
}
.proc-table td.l { text-align: left; }
.proc-table td.r { text-align: right; }
.proc-table td.name {
  color: var(--text); max-width: 280px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.proc-table tbody tr { transition: background 0.12s; }
.proc-table tbody tr:hover { background: rgba(88, 166, 255, 0.06); }
.proc-table td.cpu-cell { font-variant-numeric: tabular-nums; }

.state {
  font-size: 11px; padding: 2px 8px;
  border-radius: 4px; font-weight: 600;
}
.state-r { background: rgba(63, 185, 80, 0.15); color: var(--green); }
.state-s { background: rgba(88, 166, 255, 0.12); color: var(--accent); }
.state-d { background: rgba(210, 153, 34, 0.15); color: var(--yellow); }
.state-z { background: rgba(248, 81, 73, 0.15); color: var(--red); }
.state-t { background: rgba(139, 148, 158, 0.15); color: var(--muted); }

td.pos { color: var(--yellow); }
td.neg { color: var(--green); }

.flash-banner {
  background: var(--green); color: #fff;
  padding: 8px 14px; border-radius: 6px; font-weight: 600; font-size: 13px;
}
.flash-banner.err { background: var(--red); }

.topbar-right { display: flex; align-items: center; gap: 18px; }
.topbar-btn {
  background: var(--accent); color: #fff;
  border: none; border-radius: 6px;
  padding: 6px 14px; font-size: 13px; font-weight: 600;
  cursor: pointer; transition: all 0.15s;
  font-family: inherit;
}
.topbar-btn:hover { background: #4493f8; transform: translateY(-1px); }
.topbar-btn:active { transform: translateY(0); }

.spawn-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex; align-items: center; justify-content: center;
  z-index: 100;
  backdrop-filter: blur(2px);
}
.spawn-dialog {
  background: var(--card); border: 1px solid var(--border);
  border-radius: 12px; padding: 24px; width: 420px;
  display: flex; flex-direction: column; gap: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}
.spawn-title { font-size: 16px; font-weight: 600; color: var(--text); margin-bottom: 6px; }
.spawn-sub { font-size: 12px; color: var(--muted); font-weight: 400; margin-left: 8px; }
.spawn-label { font-size: 12px; color: var(--muted); margin-top: 6px; font-weight: 500; }
.spawn-input {
  background: var(--bg); border: 1px solid var(--border);
  border-radius: 6px; padding: 8px 12px;
  color: var(--text); font-size: 13px; outline: none;
  transition: border-color 0.15s;
  font-family: inherit;
}
.spawn-input:focus { border-color: var(--accent); }
.spawn-input::placeholder { color: var(--muted); opacity: 0.5; }
.spawn-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 12px; }
.btn-primary { background: var(--accent); color: #fff; border: none; }
.btn-primary:hover { background: #4493f8; border: none; color: #fff; }

.actions { white-space: nowrap; text-align: center; }
.actions-col { text-align: center; }
.proc-head-right {
  display: flex; align-items: center; gap: 12px;
}
.proc-search {
  background: var(--bg); border: 1px solid var(--border-subtle);
  border-radius: 6px; padding: 5px 10px;
  color: var(--text); font-size: 12px; width: 180px; outline: none;
  transition: border-color 0.15s;
  font-family: inherit;
}
.proc-search:focus { border-color: var(--accent); }
.proc-search::placeholder { color: var(--muted); opacity: 0.6; }
.btn {
  border: 1px solid var(--border); background: transparent;
  color: var(--text-secondary); font-size: 11px; font-weight: 600;
  padding: 3px 8px; border-radius: 4px; cursor: pointer;
  transition: all 0.15s; letter-spacing: 0.5px;
  font-family: inherit;
}
.btn:hover { border-color: var(--accent); color: var(--accent); }
.btn:disabled { opacity: 0.3; cursor: not-allowed; }
.btn-term:hover { border-color: var(--yellow); color: var(--yellow); }
.btn-kill:hover { border-color: var(--red); color: var(--red); background: rgba(248, 81, 73, 0.08); }
.btn-nice { padding: 3px 6px; min-width: 24px; }
.nice-val {
  display: inline-block; min-width: 22px; text-align: center;
  font-size: 12px; color: var(--muted); font-variant-numeric: tabular-nums;
}
</style>
