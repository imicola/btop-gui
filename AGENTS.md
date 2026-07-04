# AGENTS.md

> 本文件供 AI Agent 阅读，提供项目背景、开发约束、已踩过的坑。后续 Agent 接手前必读。

## 项目背景

**btop-gui** 是操作系统课程设计项目：基于 Linux `/proc` 文件系统实现图形化任务管理器（受 btop 启发）。

- **核心约束**：所有 `/proc` 解析必须**手写**（不依赖 gopsutil/sysinfo 等第三方封装），这是课设的评分点——显式展示对操作系统知识的理解。
- **目标用户**：本科生课设，评卷老师看重 `/proc` 解析、进程控制、系统调用等 OS 知识点的展示。

## 开发环境

| 项 | 值 |
|----|-----|
| OS | WSL2 (Arch Linux)，kernel 6.18.x-microsoft-standard-WSL2 |
| GUI 转发 | WSLg（DISPLAY=:0、Wayland、PulseServer 全启用） |
| Go | 1.26+ |
| Node | 25.x / npm 11.x |
| Wails CLI | v2.12.0（位于 `/home/imicola/go/bin/wails`，不在 PATH） |
| npm registry | 已设 `https://registry.npmmirror.com`（国内加速） |
| Go proxy | `proxy.golang.org,direct` |

### 系统依赖（已装）

`gtk3` + `webkit2gtk-4.1`（通过 `pacman -S gtk3 webkit2gtk-4.1`）。

验证：`pkg-config --exists webkit2gtk-4.1 && echo OK`。

## 必须遵守的开发约束

### 1. wails 命令必须加 `-tags webkit2_41`

```bash
wails dev  -tags webkit2_41   # 开发
wails build -tags webkit2_41  # 构建
```

**原因**：Wails v2 默认链接 webkit2gtk-4.0，但 Arch/Ubuntu 24.04+ 只提供 4.1。不加 tag 会编译失败。

### 2. WSLg DMA-BUF workaround 必须保留

`main.go` 里的 `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")` **不能删除**。删了 WSLg 下窗口会白屏。

### 3. ECharts 必须用 SVG renderer

```ts
echarts.init(el, undefined, { renderer: 'svg' })
```

**原因**：WSLg 的 WebKitGTK 软件渲染对 canvas 支持不可靠，SVG 是纯 DOM 一定工作。不要改回 canvas（默认）。

### 4. ECharts 初始化必须用 `watch(snap)`，不能用 `onMounted`

**原因**：chart 容器在 `v-if="snap"` 块内，`onMounted` 时 snap 还是 null，DOM 里没元素，`echarts.init()` 会被跳过。必须等第一次数据到达后才初始化。

```ts
watch(snap, (val) => {
  if (val && !cpuChart && !memChart) {
    nextTick(initCharts)
  }
})
```

### 5. `/proc` 解析必须手写

`internal/procfs/` 里的所有解析（`/proc/stat`、`/proc/meminfo`、`/proc/[pid]/stat` 等）**禁止用 gopsutil/sysinfo 等库替代**。这是课设核心评分点。

gopsutil 只能作为**对照验证**（验证手写解析的数值正确），不能作为生产路径。

### 6. ECharts 版本锁定 5.x

`package.json` 里 `"echarts": "^5.6.0"`。**不要升级到 6.x**——6.x 类型定义与项目 tsconfig 的 `moduleResolution: Node` 不兼容，vue-tsc 会报错。

## 已踩过的坑（避免重蹈覆辙）

| 坑 | 现象 | 解法 |
|----|------|------|
| webkit2gtk-4.0 缺失 | `wails dev` 编译报 `Package 'webkit2gtk-4.0' not found` | 加 `-tags webkit2_41` |
| WSLg 白屏 | 窗口边框有但内容全白 | `WEBKIT_DISABLE_DMABUF_RENDERER=1` |
| wails doctor 误报 | 提示 libwebkit Not Found | 以 `pkg-config` 为准，忽略 |
| ECharts 6 类型错 | vue-tsc 报 Cannot find module 'echarts' | 降级到 5.6.0 |
| ECharts canvas 不渲染 | WSLg 下图表空白 | 用 `renderer: 'svg'` |
| ECharts + v-if 时机 | `onMounted` 时 cpuChart 为 null | 改用 `watch(snap)` 初始化 |
| npm install 超时 | wails dev 卡在 Installing frontend dependencies | `npm config set registry npmmirror` |
| libEGL DRI3 warning | 日志出现 DRI3 error | 无害，软件渲染下正常工作，忽略 |

## 开发工作流

### 启动开发模式

项目持续在 tmux session `btop` 里运行 wails dev：

```bash
# tmux session 'btop' 已存在，里面跑着：
wails dev -tags webkit2_41
```

- 前端改动（App.vue / style.css）→ vite HMR 秒级生效，但**注意 ECharts 实例在 HMR 后会失效**（onMounted/watch 不重新触发），需要刷新窗口或重启 wails dev
- 后端改动（*.go）→ wails 自动重编译，5-10 秒生效
- 日志：`/tmp/wails-dev.log`

### 重启 wails dev

```bash
# 在 tmux 里 Ctrl-C 停止，然后重新启动
tmux send-keys -t btop C-c
tmux send-keys -t btop "cd /home/imicola/imicola_resource/os_design/btop-gui && /home/imicola/go/bin/wails dev -tags webkit2_41 2>&1 | tee /tmp/wails-dev.log" Enter
```

### 访问 dev server 调试

浏览器可访问 `http://localhost:34115`，但**注意**：浏览器里 `window.go` 不存在（Wails 只在 webview 注入），`GetSnapshot` 调用会失败。这只影响后端数据，不影响前端 UI/ECharts 验证。

## 文件职责

| 文件 | 职责 | 改动注意事项 |
|------|------|-------------|
| `main.go` | Wails 入口 + WSLg workaround | 不要删 `os.Setenv` 那行 |
| `app.go` | App 上下文、采样与进程操作 | CPU% 状态需加锁；kill/setpriority 必须校验 PID + starttime |
| `internal/procfs/types.go` | 数据结构 | `UTime`/`STime` 用 `json:"-"` 隐藏，是计算 CPU% 的中间值 |
| `internal/procfs/cpu.go` | `/proc/stat` 解析 + CPU% 算法 | 公式见注释，不要改 |
| `internal/procfs/mem.go` | `/proc/meminfo` 解析 | `Used = Total - Available`（比 Free 更准） |
| `internal/procfs/system.go` | `/proc/loadavg`、`/proc/uptime`、CPU/主机信息 | `ClkTck = 100` 是 Linux USER_HZ 硬编码 |
| `internal/procfs/process.go` | `/proc/[pid]/stat` + 用户解析 | **comm 字段括号陷阱**：必须用第一个 `(` 和最后一个 `)` 截取 |
| `internal/procfs/network.go` | `/proc/net/dev` 累计量与速率 | 首帧/计数回退必须返回 0 |
| `internal/procfs/disk.go` | `/proc/diskstats` + `statfs(2)` | sector 固定按 512 bytes 换算，过滤分区/虚拟盘 |
| `internal/procfs/detail.go` | 选中进程 status/cmdline/io/fd | 权限不足是正常情况，返回 unavailable |
| `internal/processctl/` | Linux+cgo 原生 fork 演示 | 子进程只能立即 execv 或 _exit，不能返回 Go runtime |
| `frontend/src/App.vue` | 采样编排、动作与事件轨 | 后一次轮询必须等前一次 IPC 完成 |
| `frontend/src/components/` | btop 高密度面板 | 所有 ECharts 必须保持 SVG renderer |
| `frontend/src/style.css` | 深色主题 CSS 变量 | 颜色变量定义在这里 |

## 注释约定

本项目注释偏多，因为这是**操作系统课设**，注释的核心价值是**展示对 /proc 文件系统的理解**（评卷/答辩加分点）。以下注释类型是必要的，不要删：

1. **数据格式说明**：`/proc/stat`、`/proc/meminfo` 等文件的字段格式
2. **字段索引映射**：`/proc/[pid]/stat` 的字段号对应（man proc 参考）
3. **关键陷阱警示**：comm 括号陷阱、WSLg DMA-BUF 等
4. **数学公式**：CPU% 计算公式
5. **非显然行为说明**：如 "多线程进程 CPU% 可 > 100%"、"首次调用 CPU% 为 0"
6. **Go doc 注释**：公共 API 的标准文档

**不需要的注释**：步骤标注（如 `// 1) 先做 X`）、显而易见的行内注释。已清理过。

## 后续工作待办

1. 根据最终答辩机器分辨率微调密度并补充正式截图。
2. 如需跨架构运行，将 `ClkTck` 改为 `sysconf(_SC_CLK_TCK)` 运行时探测。

## 禁止事项

- ❌ 不要用 `gopsutil`/`sysinfo` 替代手写 `/proc` 解析（违背课设初衷）
- ❌ 不要把 ECharts renderer 改回 canvas（WSLg 下不工作）
- ❌ 不要删 `main.go` 的 `WEBKIT_DISABLE_DMABUF_RENDERER` 环境变量
- ❌ 不要升级 ECharts 到 6.x（类型不兼容）
- ❌ 不要用 `as any` / `@ts-ignore` 压制类型错误
- ❌ 不要忘记 `wails dev -tags webkit2_41`（不加 tag 编译失败）
1
