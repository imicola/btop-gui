# btop-gui

基于 Linux `/proc` 文件系统的图形化任务管理器。项目面向操作系统课程设计，所有核心数据均手写解析，不依赖 gopsutil/sysinfo；界面使用 Wails（Linux GTK/WebKitGTK）+ Vue 3 + ECharts SVG。

## 功能

- CPU 总体/每核使用率、负载、型号、频率与 60 点历史曲线。
- 内存、缓存、可用内存与 Swap 使用率。
- `/proc/net/dev` 网络上下行速率和累计流量。
- `/proc/diskstats` 磁盘读写速率、忙碌度与根分区容量。
- 进程平铺/树视图、搜索、排序、用户、CPU、RSS、线程与 Nice。
- 按需进程详情：完整命令、运行时间、I/O、FD、上下文切换、cwd/exe。
- 进程创建、SIGTERM/SIGKILL、Nice 调整；PID+starttime 防复用误操作。
- Linux+cgo 真实 `fork()` → `execv()` → `wait4()` 演示。
- 鼠标与键盘操作：J/K 或方向键选择、F 搜索、T 树视图、P 暂停、N 新建。

## 架构

```text
/proc/stat, meminfo, net/dev, diskstats, [pid]/*
                 │ 手写解析与字段校验
                 ▼
        Go 采样状态 / 差分 / 系统调用
                 │ Wails IPC
                 ▼
        Vue 高密度 TUI×GUI + ECharts SVG
```

核心代码：

- `internal/procfs/`：CPU、内存、网络、磁盘、进程与系统信息解析。
- `internal/processctl/`：受控原生 fork 演示。
- `app.go`：采样状态、速率差分和 Wails 公共接口。
- `frontend/src/`：监控面板、进程树、详情与操作界面。

## 开发与构建

环境要求：Go 1.23+、Node 20+、Wails v2、GTK3、WebKitGTK 4.1。

```bash
# Arch Linux
sudo pacman -S --needed gtk3 webkit2gtk-4.1

# 前端依赖
cd frontend && npm install && cd ..

# 开发模式（必须携带 webkit2_41 tag）
~/go/bin/wails dev -tags webkit2_41

# 生产构建
~/go/bin/wails build -tags webkit2_41
```

WSLg 下保留 `WEBKIT_DISABLE_DMABUF_RENDERER=1` 以避免 WebKit 白屏。图表必须使用 SVG renderer；ECharts 保持 5.x。

## 测试

```bash
GOCACHE=/tmp/btop-gocache go test ./... -race -count=1
GOCACHE=/tmp/btop-gocache go vet ./...
cd frontend && npm run build
```

测试覆盖 `/proc` 字段解析、计数器回退、网络/磁盘速率、进程身份、详情解析和真实 fork 子进程。

## 课程设计材料

- [评分要求对照与验收](docs/评分要求对照与验收.md)
- [系统设计说明](docs/系统设计说明.md)
- [演示与答辩提纲](docs/演示与答辩提纲.md)

## 重要实现说明

### CPU 使用率

```text
total_delta = current_total - previous_total
idle_delta  = current_idle  - previous_idle
usage       = (total_delta - idle_delta) / total_delta × 100%
```

进程 CPU 使用 top 的 Irix 模式：单核满载为 100%，多线程进程可以超过 100%。

### 网络与磁盘速率

`/proc/net/dev` 和 `/proc/diskstats` 提供的是累计量，因此第一次采样返回 0，之后使用 `delta / wall_seconds`。磁盘 sector 按 Linux 接口定义换算为 512 bytes。计数器回退时返回 0，避免无符号下溢。

### fork 与 Go runtime

纯 fork 后继续运行 Go 代码并不安全。本项目通过 cgo 调用 libc `fork()`，子进程不返回 Go runtime，而是立即 `execv("/bin/sleep")`；父进程使用 `wait4` 回收。这既展示真实系统调用，也避免僵尸进程和 runtime 状态损坏。

### 权限边界

普通用户不能提高其他进程优先级，也不能操作无权限进程。部分 `/proc/[pid]` 详情可能受权限限制，界面显示 N/A，不要求 root 运行。
