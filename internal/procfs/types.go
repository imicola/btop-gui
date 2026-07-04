// Package procfs 手写实现 Linux /proc 文件系统解析。
//
// 作为操作系统课程设计的核心知识点展示，所有解析均显式读取
// /proc/stat、/proc/meminfo、/proc/[pid]/stat 等文件，不依赖第三方封装。
package procfs

// CPUUsage CPU 使用率聚合
type CPUUsage struct {
	Total     float64   `json:"total"`
	PerCore   []float64 `json:"perCore"`
	CoreCount int       `json:"coreCount"`
}

// MemoryInfo 内存信息，单位 KB（来自 /proc/meminfo 原始单位）
type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Cached    uint64  `json:"cached"`
	Buffers   uint64  `json:"buffers"`
	Used      uint64  `json:"used"`
	Usage     float64 `json:"usage"`
	SwapTotal uint64  `json:"swapTotal"`
	SwapFree  uint64  `json:"swapFree"`
	SwapUsage float64 `json:"swapUsage"`
}

// SystemInfo 描述主机和内核信息；部分虚拟化环境可能不提供 CPU 频率。
type SystemInfo struct {
	Hostname   string  `json:"hostname"`
	Kernel     string  `json:"kernel"`
	CPUModel   string  `json:"cpuModel"`
	CPUFreqMHz float64 `json:"cpuFreqMHz"`
}

// NetworkInterfaceInfo 是 /proc/net/dev 中一个网络接口的累计量和采样速率。
type NetworkInterfaceInfo struct {
	Name    string  `json:"name"`
	RXBytes uint64  `json:"rxBytes"`
	TXBytes uint64  `json:"txBytes"`
	RXRate  float64 `json:"rxRate"`
	TXRate  float64 `json:"txRate"`
}

type NetworkInfo struct {
	Available  bool                   `json:"available"`
	Error      string                 `json:"error"`
	Primary    string                 `json:"primary"`
	Interfaces []NetworkInterfaceInfo `json:"interfaces"`
}

// DiskDeviceInfo 使用 /proc/diskstats 的 512-byte sector 计数计算吞吐率。
type DiskDeviceInfo struct {
	Name        string  `json:"name"`
	ReadBytes   uint64  `json:"readBytes"`
	WriteBytes  uint64  `json:"writeBytes"`
	ReadRate    float64 `json:"readRate"`
	WriteRate   float64 `json:"writeRate"`
	BusyPercent float64 `json:"busyPercent"`
}

type FilesystemInfo struct {
	MountPoint string  `json:"mountPoint"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	Usage      float64 `json:"usage"`
}

type DiskInfo struct {
	Available bool             `json:"available"`
	Error     string           `json:"error"`
	Root      FilesystemInfo   `json:"root"`
	Devices   []DiskDeviceInfo `json:"devices"`
}

// ProcessInfo 单个进程的快照信息（来自 /proc/[pid]/stat）
type ProcessInfo struct {
	PID       int     `json:"pid"`
	Name      string  `json:"name"`
	State     string  `json:"state"` // R/S/D/Z/T
	PPID      int     `json:"ppid"`
	Nice      int     `json:"nice"`
	CPU       float64 `json:"cpu"`   // 占单核百分比，多线程进程可 > 100%（与 top Irix 模式一致）
	RSS       uint64  `json:"rss"`   // 驻留物理内存 bytes
	VSize     uint64  `json:"vsize"` // 虚拟内存 bytes
	Threads   int     `json:"threads"`
	StartTime uint64  `json:"startTime"` // 自系统启动后的启动 tick；与 PID 共同标识一次进程实例
	UID       uint64  `json:"uid"`
	User      string  `json:"user"`

	// UTime/STime 是计算 CPU% 的中间值，不暴露给前端
	UTime uint64 `json:"-"`
	STime uint64 `json:"-"`
}

// ProcessDetail 按需读取，不放入每秒全量进程扫描，避免 O(N) 额外 I/O。
type ProcessDetail struct {
	PID                 int      `json:"pid"`
	StartTime           uint64   `json:"startTime"`
	User                string   `json:"user"`
	UID                 uint64   `json:"uid"`
	Command             string   `json:"command"`
	Executable          string   `json:"executable"`
	CWD                 string   `json:"cwd"`
	Elapsed             float64  `json:"elapsed"`
	VmPeak              uint64   `json:"vmPeak"`
	VmSwap              uint64   `json:"vmSwap"`
	ReadBytes           uint64   `json:"readBytes"`
	WriteBytes          uint64   `json:"writeBytes"`
	VoluntarySwitches   uint64   `json:"voluntarySwitches"`
	InvoluntarySwitches uint64   `json:"involuntarySwitches"`
	FDCount             int      `json:"fdCount"`
	Unavailable         []string `json:"unavailable"`
}
