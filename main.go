package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/manfengjun/wintools/internal/common"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icon.ico
var appIconBytes []byte

func main() {
	common.InitLogger()
	defer common.CloseLogger()
	common.Info("应用启动")
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "码力工坊",
		Width:     860,
		Height:    640,
		MinWidth:  720,
		MinHeight: 520,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
			app.DesktopLock,
			app.PyEnv,
			app.Updater,
		},
		// 单实例锁：第二个启动实例会激活第一个实例的窗口
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "Wintools-SingleInstance",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				app.showMainWindow()
			},
		},
	})

	if err != nil {
		common.Error("应用异常退出: %v", err)
	}
	common.Info("应用正常退出")
}

// beforeClose 在窗口关闭前调用。
// - 用户点 X → 隐藏到系统托盘（返回 true 阻止退出）
// - 用户点托盘"退出" → 真正退出（quitting = true，返回 false）
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting {
		return false // 真正退出
	}
	// 不退出，只隐藏窗口到系统托盘
	wailsRuntime.WindowHide(ctx)
	return true // 阻止窗口关闭
}
