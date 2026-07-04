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
