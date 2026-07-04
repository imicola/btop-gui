# btop-gui frontend

Vue 3 + TypeScript 前端，负责高密度 TUI×GUI 监控界面。所有数据通过 Wails 生成绑定调用 Go 后端。

```bash
npm install
npm run dev
npm run build
```

浏览器直接打开 Vite 地址时没有 Wails IPC，只能验证静态布局；完整功能请从项目根目录运行：

```bash
~/go/bin/wails dev -tags webkit2_41
```

约束：ECharts 使用 5.x 和 SVG renderer，不使用 `any` 或 `@ts-ignore` 绕过类型检查。
