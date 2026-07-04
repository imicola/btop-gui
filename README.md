# btop-gui

> 基于 Linux `/proc` 文件系统的图形化任务管理器 —— 操作系统课程设计

受 [btop](https://github.com/aristocratos/btop) 启发，使用 **Wails (Go) + Vue 3 + ECharts** 实现，所有 `/proc` 解析与系统调用均**手写**，显式展示操作系统知识点。

---

## 功能特性

### 已实现

| 模块 | 说明 |
|------|------|
| **CPU 监控** | 总体使用率 + 每核使用率（柱状）+ 60 秒历史曲线（渐变填充折线） |
| **内存监控** | 物理内存 + Swap 使用率 + 60 秒历史曲线 |
| **进程列表** | PID / 名称 / 状态 / CPU% / RSS 内存 / Nice / 线程数 / 父 PID |
| **进程排序** | 表头点击排序，支持 7 列（PID/Name/State/CPU/MEM/Nice/PPID） |
| **进程操作** | SIGTERM/SIGKILL、Nice 调整；PID + starttime 防复用误操作 |
| **进程创建** | 通过 `exec.Command` 演示 Linux fork/clone + execve 创建流程 |
| **系统信息** | Uptime + 1/5/15 分钟平均负载 |
| **CPU% 算法** | top Irix 模式，单核满载 100%，多线程进程可 > 100% |
| **刷新频率** | 每秒一次 |

---

## 技术栈

| 层 | 技术 | 版本 |
|----|------|------|
| 桌面框架 | [Wails](https://wails.io) | v2.12.0 |
| 后端语言 | Go | 1.26+ |
| 前端框架 | Vue 3 + TypeScript | - |
| 构建工具 | Vite | 3.x |
| 图表库 | ECharts（SVG renderer） | 5.6.0 |
| GUI 后端 | WebKitGTK 4.1 | 系统依赖 |

---

## 项目架构

```
btop-gui/
├── main.go                          # Wails 入口 + WSLg DMA-BUF workaround
├── app.go                           # App 上下文 + GetSnapshot() 暴露给前端
├── go.mod / wails.json
├── internal/procfs/                 # ★ 核心知识点：手写 /proc 解析
│   ├── types.go                     # 数据结构定义
│   ├── cpu.go                       # /proc/stat → CPU 总体/每核使用率
│   ├── mem.go                       # /proc/meminfo → 内存/Swap
│   ├── system.go                    # /proc/loadavg、/proc/uptime + ClkTck/PageSize 常量
│   └── process.go                   # /proc/[pid]/stat（含 comm 括号陷阱处理）
└── frontend/src/
    ├── App.vue                      # 主界面（CPU/Memory 卡片 + 进程表 + 历史曲线）
    ├── style.css                    # btop 风格深色主题
    └── main.ts                      # Vue 入口
```

### 数据流

```
/proc/*  ──(手写解析)──▶  Go 后端 (internal/procfs)
                              │
                              ▼
                         App.GetSnapshot()  ──(Wails IPC)──▶  前端 poll() (1Hz)
                                                              │
                                                              ▼
                                                         Vue 响应式更新
                                                              │
                                              ┌───────────────┼───────────────┐
                                              ▼               ▼               ▼
                                         CPU/MEM 卡片     ECharts 曲线     进程表格
```

---

## 核心技术点

### 1. `/proc` 文件系统解析（全手写）

所有解析位于 `internal/procfs/`，不依赖 gopsutil 等第三方封装：

| 文件 | 字段 | 用途 |
|------|------|------|
| `/proc/stat` | `cpu` 行的 user/nice/system/idle/iowait/irq/softirq/steal | 计算 CPU 使用率 |
| `/proc/meminfo` | MemTotal/MemAvailable/Cached/Buffers/SwapTotal/SwapFree | 内存信息 |
| `/proc/[pid]/stat` | 字段 3(state) 4(ppid) 14(utime) 15(stime) 19(nice) 20(threads) 23(vsize) 24(rss) | 进程信息 |
| `/proc/loadavg` | 1/5/15 分钟平均负载 | 系统负载 |
| `/proc/uptime` | 系统运行秒数 | Uptime |

**关键陷阱**：`/proc/[pid]/stat` 的 comm 字段（字段 2）被括号包围，且进程名本身可能含空格或括号。必须用第一个 `(` 和最后一个 `)` 截取 comm，再从 `)` 之后解析剩余字段。见 `process.go` 的 `parseStatLine`。

### 2. CPU 使用率计算

**总体 CPU%**（来自 `/proc/stat`）：

```
totalDelta = currTotal - prevTotal      // 两次采样间总 tick 数
idleDelta  = currIdle  - prevIdle       // 两次采样间空闲 tick 数（idle + iowait）
CPU% = (totalDelta - idleDelta) / totalDelta * 100
```

**进程 CPU%**（来自 `/proc/[pid]/stat` 的 utime + stime）：

```
pcpu = delta_ticks / (wall_seconds * ClkTck) * 100
```

- `delta_ticks` = 当前采样 (utime+stime) - 上次采样 (utime+stime)
- `wall_seconds` = 两次采样的墙钟时间差
- `ClkTck` = 100（Linux USER_HZ）
- **单核满载 = 100%**，多线程进程可 > 100%（与 top 的 Irix 模式一致）

### 3. RSS 内存

`/proc/[pid]/stat` 字段 24 是 rss（驻留页数），需要乘以页大小：

```go
RSS = statField(24) * PageSize  // PageSize = syscall.Getpagesize() = 4096
```

### 4. 系统调用（待实现）

| 操作 | Go 调用 | 对应 C 接口 |
|------|---------|-------------|
| 杀死进程 | `syscall.Kill(pid, sig)` | `kill(2)` |
| 调整优先级 | `unix.Setpriority(PRIO_PROCESS, pid, nice)` | `setpriority(2)` |
| 创建进程 | `exec.Command(...)` (fork+exec 原子操作) | `fork(2)` + `exec(3)` |

> **Go 的 fork 限制**：Go runtime 是 M:N 线程模型，纯 `fork()`（fork 后不 exec）会导致 runtime 状态不一致而死锁。Go 社区共识是永远用 `exec.Command`（= fork+exec 原子操作）。

---

## 环境要求与运行

### 系统依赖

**Linux（含 WSL2）需要安装 GTK3 + WebKitGTK 4.1 开发包：**

```bash
# Arch Linux (WSL2)
sudo pacman -S --needed gtk3 webkit2gtk-4.1

# Ubuntu/Debian
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
```

### 开发工具

```bash
# Go 1.23+
go version

# Node.js 18+ / npm
node --version

# Wails CLI（用户级安装，不污染系统）
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 国内网络优化（推荐）

```bash
# Go 模块代理
go env -w GOPROXY=https://goproxy.cn,direct

# npm 国内镜像
npm config set registry https://registry.npmmirror.com
```

### 启动开发模式

```bash
cd btop-gui
wails dev -tags webkit2_41
```

> ⚠️ **必须加 `-tags webkit2_41`**：Wails v2 默认链接 webkit2gtk-4.0，但 Arch/Ubuntu 24.04+ 只提供 4.1。详见踩坑记录。

### 构建生产版本

```bash
wails build -tags webkit2_41
# 产物在 build/bin/btop-gui
```

---

## 踩坑记录

### 1. `webkit2gtk-4.0 not found`（关键 ⚠️）

**现象**：`wails dev` 编译报 `Package 'webkit2gtk-4.0' not found`。

**原因**：Wails v2 默认链接 webkit2gtk-4.0，但 Arch 新版 / Ubuntu 24.04+ 只提供 webkit2gtk-4.1。

**解法**：所有 wails 命令加 `-tags webkit2_41`。

### 2. WSLg 窗口白屏

**现象**：WSLg 下 WebKit 窗口边框有但内容全白。

**原因**：WebKitGTK 的 DMA-BUF 渲染器在 WSLg（NVIDIA GPU / 无 DRI3 支持）下 EGL 初始化失败。

**解法**：`main.go` 启动时 `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")`，回退软件渲染。

### 3. `libEGL warning: DRI3 error`（无害）

**现象**：日志出现 `libEGL warning: DRI3 error: Could not get DRI3 device`。

**结论**：WSLg 的 X server 不支持 DRI3 硬件加速，软件渲染仍能正常工作，忽略即可。

### 4. wails doctor 误报 `libwebkit Not Found`

**现象**：`wails doctor` 提示 libwebkit 缺失，但 `pkg-config --exists webkit2gtk-4.1` 正常。

**原因**：wails doctor 的依赖映射表里 libwebkit 对应的包名没覆盖 `webkit2gtk-4.1`。

**解法**：以 `pkg-config` 为准，忽略这条 warning。

### 5. ECharts 6 类型不兼容

**现象**：vue-tsc 报 `Cannot find module 'echarts'`。

**原因**：ECharts 6.1.0（2025 新版）类型定义需要新的 moduleResolution。

**解法**：降级到 ECharts 5.6.0（稳定成熟）。

### 6. ECharts 在 WSLg 不渲染（canvas 问题）

**现象**：ECharts 初始化成功但图表不可见。

**原因**：WebKitGTK 在 WSLg 软件渲染下 canvas 渲染不可靠。

**解法**：使用 SVG renderer：`echarts.init(el, undefined, { renderer: 'svg' })`。

### 7. ECharts + Vue `v-if` 时机问题

**现象**：`onMounted` 时初始化 ECharts，但 `cpuChart` 始终为 null。

**原因**：chart 容器在 `v-if="snap"` 块内，`onMounted` 时 `snap` 还是 null（异步 poll 未返回），DOM 里没有 chart 元素，`echarts.init()` 被跳过。

**解法**：用 `watch(snap)` 在第一次数据到达、DOM 渲染后再初始化：

```ts
watch(snap, (val) => {
  if (val && !cpuChart && !memChart) {
    nextTick(initCharts)
  }
})
```

### 8. npm install 慢/超时

**现象**：wails dev 启动时 `Installing frontend dependencies` 卡住。

**解法**：`npm config set registry https://registry.npmmirror.com`。

---

## 课设要求对应表

| 要求 | 状态 | 实现 |
|------|------|------|
| 1. 实时显示 CPU/内存使用率、进程列表（PID/状态/资源占比） | ✅ | `/proc/stat`、`/proc/meminfo`、`/proc/[pid]/stat` 手写解析 |
| 2. 进程操作（创建、杀死、优先级调整） | ✅ | `exec.Command` + `syscall.Kill` + `unix.Setpriority` |
| 3. 图形界面展示历史曲线 | ✅ | ECharts SVG renderer，60 秒环形缓冲 |
| 4. 解析 `/proc/pid/stat`、`/proc/meminfo` 等 | ✅ | `internal/procfs/` 全手写 |
| 5. 调用 `fork()`、`kill()` 等系统调用 | ✅ | `kill(2)`；Go 通过 `exec.Command` 使用 clone/fork + execve |

---

## 技术选型理由

### 为什么是 Wails（Go）而不是 Tauri（Rust）/ GTK / Qt？

| 维度 | Wails (Go) | Tauri (Rust) | GTK/Qt (C/C++) |
|------|-----------|--------------|----------------|
| 不写 C/C++ | ✅ Web 技术 | ✅ Web 技术 | ❌ 必须写 |
| 构建速度 | ✅ 5-10s | ❌ 70s+ | ✅ 快 |
| 学习曲线 | ✅ 低 | ❌ 高（所有权系统） | 中 |
| 系统调用便利 | ✅ 标准库 | 需 nix crate | ✅ 原生 |
| WSL2 适配 | ✅ | ✅ | N/A |
| UI 美观度 | ✅ Web 自由度 | ✅ Web 自由度 | ❌ 受限 |

**最终决策**：Wails (Go)，主要因为构建速度快 10x+、学习曲线低、syscall 在标准库内。

### 为什么用 ECharts SVG renderer？

WSLg 的 WebKitGTK 软件渲染对 canvas 支持不可靠（DRI3 缺失），SVG 是纯 DOM 渲染，稳定兼容。
