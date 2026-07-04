package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// WSLg / 部分 Linux GPU 环境下 WebKitGTK 的 DMA-BUF 渲染器会触发白屏/黑屏
	// （EGL 初始化失败），禁用后回退到软件渲染
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "btop-gui",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 13, G: 17, B: 23, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
