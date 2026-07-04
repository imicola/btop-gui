export interface CPUUsage { total: number; perCore: number[]; coreCount: number }
export interface MemoryInfo {
  total: number; free: number; available: number; used: number; usage: number
  cached: number; buffers: number; swapTotal: number; swapFree: number; swapUsage: number
}
export interface SystemInfo { hostname: string; kernel: string; cpuModel: string; cpuFreqMHz: number }
export interface NetworkInterfaceInfo {
  name: string; rxBytes: number; txBytes: number; rxRate: number; txRate: number
}
export interface NetworkInfo {
  available: boolean; error: string; primary: string; interfaces: NetworkInterfaceInfo[]
}
export interface DiskDeviceInfo {
  name: string; readBytes: number; writeBytes: number
  readRate: number; writeRate: number; busyPercent: number
}
export interface FilesystemInfo {
  mountPoint: string; total: number; used: number; available: number; usage: number
}
export interface DiskInfo {
  available: boolean; error: string; root: FilesystemInfo; devices: DiskDeviceInfo[]
}
export interface ProcInfo {
  pid: number; name: string; state: string; ppid: number; nice: number
  cpu: number; rss: number; vsize: number; threads: number; startTime: number
  uid: number; user: string
}
export interface ProcessDetail {
  pid: number; startTime: number; user: string; uid: number; command: string
  executable: string; cwd: string; elapsed: number; vmPeak: number; vmSwap: number
  readBytes: number; writeBytes: number; voluntarySwitches: number
  involuntarySwitches: number; fdCount: number; unavailable: string[]
}
export interface Snapshot {
  timestamp: number; cpu: CPUUsage; memory: MemoryInfo; loadAvg: number[]
  uptime: number; procs: ProcInfo[]; procCount: number; system: SystemInfo
  network: NetworkInfo; disk: DiskInfo
}
export interface StatusEvent { time: string; message: string; error?: boolean }
