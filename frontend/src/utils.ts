export function fmtBytes(value: number): string {
  if (!Number.isFinite(value)) return 'N/A'
  if (value >= 1024 ** 4) return `${(value / 1024 ** 4).toFixed(1)} TiB`
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} GiB`
  if (value >= 1024 ** 2) return `${(value / 1024 ** 2).toFixed(1)} MiB`
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${Math.round(value)} B`
}

export function fmtRate(value: number): string {
  return `${fmtBytes(value)}/s`
}

export function fmtKB(value: number): string {
  return fmtBytes(value * 1024)
}

export function fmtDuration(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  return d > 0 ? `${d}d ${h}h ${m}m` : `${h}h ${m}m ${s}s`
}

// 为低负载 CPU 历史计算动态纵轴。保留上下留白并使用整刻度，
// 同时严格限制在百分比合法范围 0..100。
export function adaptivePercentRange(values: number[]): { min: number; max: number } {
  const samples = values.filter(value => Number.isFinite(value)).map(value => Math.max(0, Math.min(100, value)))
  if (!samples.length) return { min: 0, max: 10 }
  const low = Math.min(...samples)
  const high = Math.max(...samples)
  const span = Math.max(high - low, 2)
  const padding = span * 0.35
  const rawMin = Math.max(0, low - padding)
  const rawMax = Math.min(100, high + padding)
  const step = rawMax <= 15 ? 1 : rawMax <= 40 ? 2 : 5
  let min = Math.max(0, Math.floor(rawMin / step) * step)
  let max = Math.min(100, Math.ceil(rawMax / step) * step)
  if (max - min < 4) {
    max = Math.min(100, min + 4)
    min = Math.max(0, max - 4)
  }
  return { min, max }
}

// 仅做 argv 切分，不调用 shell；支持单双引号和反斜线转义。
export function parseArguments(input: string): string[] {
  const args: string[] = []
  let current = ''
  let quote: 'single' | 'double' | null = null
  let escaped = false
  let started = false
  for (const char of input) {
    if (escaped) {
      current += char
      escaped = false
      started = true
    } else if (char === '\\' && quote !== 'single') {
      escaped = true
      started = true
    } else if (char === "'" && quote !== 'double') {
      quote = quote === 'single' ? null : 'single'
      started = true
    } else if (char === '"' && quote !== 'single') {
      quote = quote === 'double' ? null : 'double'
      started = true
    } else if (/\s/.test(char) && quote === null) {
      if (started) {
        args.push(current)
        current = ''
        started = false
      }
    } else {
      current += char
      started = true
    }
  }
  if (escaped || quote !== null) throw new Error('参数中存在未闭合的引号或转义符')
  if (started) args.push(current)
  return args
}
