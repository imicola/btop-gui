<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

const props = withDefaults(defineProps<{
  series: number[][]
  colors: string[]
  min?: number | 'auto'
  max?: number | 'auto'
}>(), { min: 0, max: 'auto' })

const root = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

function option(): echarts.EChartsOption {
  return {
    // 历史数组满后会整体左移；关闭 update tween，避免整条曲线向后插值重绘。
    // smooth 仅控制曲线几何形状，仍保留圆滑观感。
    animation: false,
    grid: { left: 0, right: 0, top: 2, bottom: 0 },
    xAxis: { type: 'category', show: false, boundaryGap: false },
    yAxis: {
      type: 'value', show: false, scale: true,
      min: props.min === 'auto' ? undefined : props.min,
      max: props.max === 'auto' ? undefined : props.max,
    },
    series: props.series.map((data, index) => ({
      id: `line-${index}`,
      type: 'line', data, smooth: .48, smoothMonotone: 'x',
      symbol: 'none', showSymbol: false, connectNulls: true,
      lineStyle: { color: props.colors[index], width: 1.65, opacity: .96 },
      emphasis: { disabled: true },
    })),
  }
}

function render() { chart?.setOption(option(), { notMerge: false, lazyUpdate: true }) }
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
