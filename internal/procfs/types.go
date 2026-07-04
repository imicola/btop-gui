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

	// UTime/STime 是计算 CPU% 的中间值，不暴露给前端
	UTime uint64 `json:"-"`
	STime uint64 `json:"-"`
}
