package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"time"

	"btop-gui/internal/processctl"
	"btop-gui/internal/procfs"
	"golang.org/x/sys/unix"
)

// App 是 Wails 暴露给前端的应用上下文，持有跨调用的采样状态
type App struct {
	ctx context.Context
	mu  sync.Mutex

	prevAllCPU   *procfs.CPUSample
	prevPerCPU   map[string]procfs.CPUSample
	prevProcCPU  map[int]procCPUSample
	prevWall     time.Time
	prevNetwork  map[string]procfs.NetworkSample
	prevNetWall  time.Time
	prevDisk     map[string]procfs.DiskSample
	prevDiskWall time.Time
	system       procfs.SystemInfo
}

type procCPUSample struct {
	ticks     uint64
	startTime uint64
}

func NewApp() *App {
	return &App{system: procfs.ReadSystemInfo()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SnapshotData 是一次采样的完整快照，由前端定时拉取
type SnapshotData struct {
	Timestamp int64                `json:"timestamp"`
	CPU       procfs.CPUUsage      `json:"cpu"`
	Memory    procfs.MemoryInfo    `json:"memory"`
	LoadAvg   [3]float64           `json:"loadAvg"`
	Uptime    float64              `json:"uptime"`
	Procs     []procfs.ProcessInfo `json:"procs"`
	ProcCount int                  `json:"procCount"`
	System    procfs.SystemInfo    `json:"system"`
	Network   procfs.NetworkInfo   `json:"network"`
	Disk      procfs.DiskInfo      `json:"disk"`
}

type ForkResult struct {
	PID             int    `json:"pid"`
	ParentPID       int    `json:"parentPid"`
	DurationSeconds int    `json:"durationSeconds"`
	Implementation  string `json:"implementation"`
}

// GetSnapshot 采集一次系统快照（CPU/内存/负载/进程列表）
// 首次调用 prevAllCPU 为 nil，CPU% 为 0；从第二次开始返回真实使用率
func (a *App) GetSnapshot() (*SnapshotData, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	wallNow := time.Now()
	resp := &SnapshotData{Timestamp: wallNow.UnixMilli()}
	resp.System = a.system

	allCPU, perCPU, err := procfs.ReadCPUSamples()
	if err != nil {
		return nil, fmt.Errorf("读取 CPU 数据: %w", err)
	}
	var total float64
	if a.prevAllCPU != nil {
		total = procfs.CPUUsagePercent(*a.prevAllCPU, allCPU)
	}
	perCore := make([]float64, len(perCPU))
	for i, cur := range perCPU {
		if prev, ok := a.prevPerCPU[cur.Name]; ok {
			perCore[i] = procfs.CPUUsagePercent(prev, cur)
		}
	}
	resp.CPU = procfs.CPUUsage{Total: total, PerCore: perCore, CoreCount: len(perCPU)}
	allCopy := allCPU
	a.prevAllCPU = &allCopy
	a.prevPerCPU = make(map[string]procfs.CPUSample, len(perCPU))
	for _, sample := range perCPU {
		a.prevPerCPU[sample.Name] = sample
	}

	mem, err := procfs.ReadMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("读取内存数据: %w", err)
	}
	resp.Memory = mem
	la, err := procfs.ReadLoadAvg()
	if err != nil {
		return nil, fmt.Errorf("读取系统负载: %w", err)
	}
	resp.LoadAvg = la
	up, err := procfs.ReadUptime()
	if err != nil {
		return nil, fmt.Errorf("读取系统运行时间: %w", err)
	}
	resp.Uptime = up

	if samples, netErr := procfs.ReadNetworkSamples(); netErr != nil {
		resp.Network.Error = netErr.Error()
	} else {
		seconds := 0.0
		if !a.prevNetWall.IsZero() {
			seconds = wallNow.Sub(a.prevNetWall).Seconds()
		}
		resp.Network.Interfaces = procfs.NetworkRates(samples, a.prevNetwork, seconds)
		resp.Network.Primary = procfs.PrimaryNetworkInterface(resp.Network.Interfaces)
		resp.Network.Available = true
		a.prevNetwork = make(map[string]procfs.NetworkSample, len(samples))
		for _, sample := range samples {
			a.prevNetwork[sample.Name] = sample
		}
		a.prevNetWall = wallNow
	}

	root, rootErr := procfs.ReadRootFilesystem()
	diskSamples, diskErr := procfs.ReadDiskSamples()
	if rootErr == nil {
		resp.Disk.Root = root
		resp.Disk.Available = true
	}
	if diskErr == nil {
		seconds := 0.0
		if !a.prevDiskWall.IsZero() {
			seconds = wallNow.Sub(a.prevDiskWall).Seconds()
		}
		resp.Disk.Devices = procfs.DiskRates(diskSamples, a.prevDisk, seconds)
		resp.Disk.Available = true
		a.prevDisk = make(map[string]procfs.DiskSample, len(diskSamples))
		for _, sample := range diskSamples {
			a.prevDisk[sample.Name] = sample
		}
		a.prevDiskWall = wallNow
	}
	if rootErr != nil {
		resp.Disk.Error = "statfs: " + rootErr.Error()
	}
	if diskErr != nil {
		if resp.Disk.Error != "" {
			resp.Disk.Error += "; "
		}
		resp.Disk.Error += "diskstats: " + diskErr.Error()
	}

	procs, err := procfs.ListProcesses()
	if err != nil {
		return nil, fmt.Errorf("读取进程列表: %w", err)
	}
	resp.ProcCount = len(procs)

	// 进程 CPU% 计算公式：pcpu = delta_ticks / (wall_seconds * ClkTck) * 100
	// 单核满载 = 100%，多线程进程可 > 100%（与 top 的 Irix 模式一致）
	var wallDelta float64
	if !a.prevWall.IsZero() {
		wallDelta = wallNow.Sub(a.prevWall).Seconds()
	}
	nextPrev := make(map[int]procCPUSample, len(procs))
	for i := range procs {
		p := &procs[i]
		totalTicks := p.UTime + p.STime
		if wallDelta > 0 && a.prevProcCPU != nil {
			if prev, ok := a.prevProcCPU[p.PID]; ok &&
				prev.startTime == p.StartTime && totalTicks >= prev.ticks {
				p.CPU = float64(totalTicks-prev.ticks) / (wallDelta * procfs.ClkTck) * 100
			}
		}
		nextPrev[p.PID] = procCPUSample{ticks: totalTicks, startTime: p.StartTime}
	}
	a.prevProcCPU = nextPrev

	sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })
	resp.Procs = procs
	a.prevWall = wallNow

	return resp, nil
}

// GetProcessDetail 读取选中进程的扩展信息，并校验 PID 实例身份。
func (a *App) GetProcessDetail(pid int, startTime uint64) (*procfs.ProcessDetail, error) {
	detail, err := procfs.ReadProcessDetail(pid, startTime)
	if err != nil {
		return nil, fmt.Errorf("读取进程 %d 详情: %w", pid, err)
	}
	return &detail, nil
}

// ForkDemo 调用真实 fork()，子进程立即 execv /bin/sleep，父进程异步回收。
func (a *App) ForkDemo(seconds int) (*ForkResult, error) {
	if seconds == 0 {
		seconds = 10
	}
	if seconds < 1 || seconds > 30 {
		return nil, fmt.Errorf("fork 演示时长必须在 1 到 30 秒之间")
	}
	pid, err := processctl.ForkSleep(seconds)
	if err != nil {
		return nil, err
	}
	return &ForkResult{
		PID: pid, ParentPID: os.Getpid(), DurationSeconds: seconds,
		Implementation: "libc fork() → execv(/bin/sleep) → wait4()",
	}, nil
}

// KillProcess 向指定进程发送 Unix 信号（对应 kill(2) 系统调用）
// sig: 信号编号，常用值：
//   - 15 = SIGTERM（优雅终止，进程可捕获处理）
//   - 9  = SIGKILL（强制杀死，不可捕获/忽略）
//
// startTime 来自 /proc/[pid]/stat 字段 22，与 PID 一起校验进程身份，避免 PID 复用误杀。
// pid <= 0 会被拒绝，避免误发广播信号（kill(2) 对 pid=0 发给同进程组，pid=-1 发给所有）。
func (a *App) KillProcess(pid int, startTime uint64, sig int) error {
	if pid <= 1 {
		// pid=1 是 init/systemd，杀了会搞坏系统；pid<=0 是广播语义，都拒绝
		return fmt.Errorf("拒绝操作：pid %d 不允许被信号发送", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("拒绝操作：不能终止任务管理器自身 (pid %d)", pid)
	}
	if sig != int(syscall.SIGTERM) && sig != int(syscall.SIGKILL) {
		return fmt.Errorf("拒绝操作：仅允许 SIGTERM(%d) 或 SIGKILL(%d)", syscall.SIGTERM, syscall.SIGKILL)
	}
	if err := verifyProcessIdentity(pid, startTime); err != nil {
		return err
	}
	if err := syscall.Kill(pid, syscall.Signal(sig)); err != nil {
		return fmt.Errorf("kill(%d, %d): %w", pid, sig, err)
	}
	return nil
}

// SetPriority 调整目标进程的 nice 值（对应 setpriority(2) 系统调用）
// nice 取值范围 [-20, 19]：
//   - -20 最高优先级（只有 root 能设）
//   - 0   默认值
//   - 19  最低优先级
//
// 普通用户只能提高 nice（降低优先级），不能降低；root 可任意调整。
// 内核通过 PRIO_PROCESS 标识 "who 参数是 pid"。
// startTime 与 PID 一起校验进程身份，避免 PID 复用后修改到无关进程。
func (a *App) SetPriority(pid int, startTime uint64, nice int) error {
	if pid <= 1 {
		return fmt.Errorf("拒绝操作：pid %d 不允许调整优先级", pid)
	}
	if nice < -20 || nice > 19 {
		return fmt.Errorf("nice 值 %d 越界，合法范围 [-20, 19]", nice)
	}
	if err := verifyProcessIdentity(pid, startTime); err != nil {
		return err
	}
	// unix.Setpriority(which, who, prio)
	//   which = PRIO_PROCESS (0)：按进程 ID 查找
	//   who   = pid
	//   prio  = 目标 nice 值
	if err := unix.Setpriority(unix.PRIO_PROCESS, pid, nice); err != nil {
		return fmt.Errorf("setpriority(%d, %d): %w", pid, nice, err)
	}
	return nil
}

// verifyProcessIdentity 防止用户确认操作后，原进程退出且 PID 被内核复用，
// 导致信号或优先级修改落到一个无关的新进程上。
func verifyProcessIdentity(pid int, expectedStartTime uint64) error {
	if expectedStartTime == 0 {
		return fmt.Errorf("拒绝操作：缺少 pid %d 的进程启动标识", pid)
	}
	p, err := procfs.ReadProcess(pid)
	if err != nil {
		return fmt.Errorf("进程 %d 已退出或无法读取: %w", pid, err)
	}
	if p.StartTime != expectedStartTime {
		return fmt.Errorf("拒绝操作：pid %d 已被其他进程复用，请刷新后重试", pid)
	}
	return nil
}

// SpawnProcess 通过 fork+exec 创建新进程（对应 execve(2) 系统调用）。
//
// Linux 上 exec.Command 底层调用 clone()+execve()（clone 是 fork 的增强版，
// fork 在内核里就是 clone 的子集），这是 Unix 标准的进程创建模式：
// 父进程 fork 出子进程 → 子进程 exec 新程序。
//
// 为什么 Go 不支持纯 fork()：Go runtime 是 M:N 调度模型（多个 goroutine
// 映射到少数 OS 线程），fork 只复制调用线程，子进程中其他 runtime 线程
// 消失，导致调度器/GC/锁状态不一致。因此 Go 只提供 fork+exec（原子操作，
// exec 会覆盖整个进程镜像，runtime 状态损坏无影响）。
//
// 返回新进程的 PID。
func (a *App) SpawnProcess(name string, args []string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("程序名不能为空")
	}
	cmd := exec.Command(name, args...)
	// cmd.Stdout/Stderr 为 nil 时，子进程输出连接到 /dev/null
	// （不 capture 输出，避免 pipe 缓冲区满导致子进程阻塞）
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("exec %s: %w", name, err)
	}
	pid := cmd.Process.Pid
	// 后台 Wait 回收子进程退出后的 zombie
	// （不阻塞，goroutine 等待子进程结束后自动释放资源）
	go func() { _ = cmd.Wait() }()
	return pid, nil
}
