<script setup lang="ts">
import { computed } from 'vue'
import PanelFrame from './PanelFrame.vue'
import MiniChart from './MiniChart.vue'
import type { DiskInfo, MemoryInfo, NetworkInfo } from '../types'
import { fmtBytes, fmtKB, fmtRate } from '../utils'

const props = defineProps<{
  memory: MemoryInfo; disk: DiskInfo; network: NetworkInfo
  memoryHistory: number[]; diskHistory: number[][]; networkHistory: number[][]
}>()

const primaryNetwork = computed(() =>
  props.network.interfaces.find(item => item.name === props.network.primary) || props.network.interfaces[0]
)
const diskTotals = computed(() => props.disk.devices.reduce((sum, device) => ({
  read: sum.read + device.readRate,
  write: sum.write + device.writeRate,
  busy: Math.max(sum.busy, device.busyPercent),
}), { read: 0, write: 0, busy: 0 }))
const primaryDisk = computed(() => [...props.disk.devices].sort((a, b) =>
  b.readRate + b.writeRate - a.readRate - a.writeRate
)[0])
</script>

<template>
  <div class="resource-stack">
    <div class="resource-pair">
      <PanelFrame title="mem" tone="green" :meta="`${fmtKB(memory.total)} total`">
        <div class="metric-head"><strong>{{ memory.usage.toFixed(1) }}%</strong><span>{{ fmtKB(memory.used) }} used</span></div>
        <div class="chart-short"><MiniChart :series="[memoryHistory]" :colors="['#A6E3A1']" :max="100" /></div>
        <div class="memory-lines">
          <span>avail <b>{{ fmtKB(memory.available) }}</b></span>
          <span>cache <b>{{ fmtKB(memory.cached) }}</b></span>
          <span>swap <b>{{ memory.swapUsage.toFixed(1) }}%</b></span>
        </div>
      </PanelFrame>

      <PanelFrame title="disk" tone="violet" :meta="primaryDisk?.name || 'rootfs'">
        <template v-if="disk.available">
          <div class="metric-head"><strong>{{ disk.root.usage.toFixed(1) }}%</strong><span>{{ fmtBytes(disk.root.used) }} / {{ fmtBytes(disk.root.total) }}</span></div>
          <div class="chart-short"><MiniChart :series="diskHistory" :colors="['#7BDFF2', '#F38BA8']" /></div>
          <div class="io-lines"><span class="down">R {{ fmtRate(diskTotals.read) }}</span><span class="up">W {{ fmtRate(diskTotals.write) }}</span><span>busy {{ diskTotals.busy.toFixed(0) }}%</span></div>
        </template>
        <div v-else class="empty">N/A · {{ disk.error }}</div>
      </PanelFrame>
    </div>

    <PanelFrame title="net" tone="rose" :meta="primaryNetwork?.name || 'no interface'" class="net-panel">
      <template v-if="network.available && primaryNetwork">
        <div class="net-layout">
          <div class="net-chart"><MiniChart :series="networkHistory" :colors="['#7BDFF2', '#F38BA8']" /></div>
          <div class="net-values">
            <span class="down">▼ {{ fmtRate(primaryNetwork.rxRate) }}</span><small>total {{ fmtBytes(primaryNetwork.rxBytes) }}</small>
            <span class="up">▲ {{ fmtRate(primaryNetwork.txRate) }}</span><small>total {{ fmtBytes(primaryNetwork.txBytes) }}</small>
          </div>
        </div>
      </template>
      <div v-else class="empty">N/A · {{ network.error || '未发现网络接口' }}</div>
    </PanelFrame>
  </div>
</template>

<style scoped>
.resource-stack { display: grid; grid-template-rows: minmax(145px, 1fr) minmax(150px, 1fr); gap: 9px; min-height: 0; }
.resource-pair { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; min-height: 0; }
.metric-head { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
.metric-head strong { color: var(--text); font-size: 22px; letter-spacing: -.05em; }
.metric-head span { color: var(--muted); font-size: 10.5px; text-align: right; }
.chart-short { height: 48px; }
.memory-lines, .io-lines { display: flex; justify-content: space-between; gap: 5px; color: var(--muted); font-size: 10.5px; }
.memory-lines b { color: var(--text); font-weight: 500; }
.net-layout { display: grid; grid-template-columns: 1fr 142px; height: 100%; gap: 12px; }
.net-chart { min-height: 80px; }
.net-values { display: grid; grid-template-columns: 1fr auto; align-content: center; gap: 7px 6px; font-size: 11.5px; }
.net-values small { color: var(--muted); text-align: right; }
.down { color: var(--cyan); }
.up { color: var(--rose); }
.empty { display: grid; height: 100%; place-items: center; color: var(--muted); font: 13px Nunito, sans-serif; text-align: center; }
@media (max-width: 1050px) { .resource-stack { min-height: 380px; } }
</style>
