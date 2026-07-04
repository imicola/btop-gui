<script setup lang="ts">
import PanelFrame from './PanelFrame.vue'
import MiniChart from './MiniChart.vue'
import type { CPUUsage, SystemInfo } from '../types'

defineProps<{ cpu: CPUUsage; system: SystemInfo; history: number[]; loadAvg: number[] }>()
</script>

<template>
  <PanelFrame title="cpu" tone="violet" :meta="system.cpuModel || 'processor'" class="cpu-panel">
    <div class="cpu-layout">
      <div class="cpu-history">
        <div class="cpu-readout"><strong>{{ cpu.total.toFixed(1) }}</strong><span>%</span></div>
        <MiniChart :series="[history]" :colors="['#7BDFF2']" :max="100" fill />
        <div class="cpu-foot">
          <span>freq <b>{{ system.cpuFreqMHz ? `${(system.cpuFreqMHz / 1000).toFixed(2)} GHz` : 'N/A' }}</b></span>
          <span>load <b>{{ loadAvg.map(v => v.toFixed(2)).join(' · ') }}</b></span>
        </div>
      </div>
      <div class="core-grid">
        <div v-for="(usage, index) in cpu.perCore" :key="index" class="core-line">
          <span class="core-label">C{{ index }}</span>
          <span class="meter"><i :style="{ width: `${Math.min(100, usage)}%` }" /></span>
          <b :class="{ hot: usage >= 70 }">{{ usage.toFixed(0) }}%</b>
        </div>
      </div>
    </div>
  </PanelFrame>
</template>

<style scoped>
.cpu-panel { min-height: 190px; }
.cpu-layout { display: grid; grid-template-columns: minmax(300px, .88fr) 1.35fr; height: 100%; gap: 18px; }
.cpu-history { display: grid; grid-template-rows: auto 1fr auto; min-height: 0; }
.cpu-readout { display: flex; align-items: baseline; gap: 4px; color: var(--cyan); }
.cpu-readout strong { font-size: clamp(29px, 3vw, 42px); line-height: .9; letter-spacing: -.08em; }
.cpu-readout span { color: var(--muted); font-size: 13px; }
.cpu-foot { display: flex; justify-content: space-between; gap: 12px; color: var(--muted); font-size: 11.5px; }
.cpu-foot b { color: var(--text); font-weight: 500; }
.core-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); align-content: center; gap: 6px 13px; }
.core-line { display: grid; grid-template-columns: 27px 1fr 33px; align-items: center; gap: 6px; font-size: 11.5px; }
.core-label { color: var(--violet); }
.core-line b { color: var(--green); text-align: right; font-weight: 600; }
.core-line b.hot { color: var(--rose); }
.meter { height: 4px; overflow: hidden; background: var(--track); }
.meter i { display: block; height: 100%; background: linear-gradient(90deg, var(--cyan), var(--green)); transition: width .35s ease; }
@media (max-width: 1050px) {
  .cpu-layout { grid-template-columns: 1fr; }
  .core-grid { grid-template-columns: repeat(4, 1fr); }
}
@media (prefers-reduced-motion: reduce) { .meter i { transition: none; } }
</style>
