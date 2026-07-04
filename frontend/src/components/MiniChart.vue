<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

const props = withDefaults(defineProps<{
  series: number[][]
  colors: string[]
  max?: number | 'auto'
  fill?: boolean
}>(), { max: 'auto', fill: false })

const root = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

function option() {
  return {
    animation: false,
    grid: { left: 0, right: 0, top: 2, bottom: 0 },
    xAxis: { type: 'category', show: false, boundaryGap: false },
    yAxis: { type: 'value', show: false, min: 0, max: props.max === 'auto' ? undefined : props.max },
    series: props.series.map((data, index) => ({
      type: 'line', data, smooth: .22, symbol: 'none', connectNulls: true,
      lineStyle: { color: props.colors[index], width: 1.3 },
      areaStyle: props.fill && index === 0 ? { color: props.colors[index] + '18' } : undefined,
    })),
  }
}

function render() { chart?.setOption(option(), true) }
function resize() { chart?.resize() }

watch(() => props.series, render, { deep: true })
onMounted(() => nextTick(() => {
  if (!root.value) return
  chart = echarts.init(root.value, undefined, { renderer: 'svg' })
  render()
  window.addEventListener('resize', resize)
}))
onUnmounted(() => {
  window.removeEventListener('resize', resize)
  chart?.dispose()
})
</script>

<template><div ref="root" class="mini-chart" /></template>

<style scoped>.mini-chart { width: 100%; height: 100%; min-height: 38px; }</style>
