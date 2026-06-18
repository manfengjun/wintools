package main

import (
	"context"
	"runtime"

	"fyne.io/systray"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"DesktopSuite/internal/desktoplock"
	"DesktopSuite/internal/pyenv"
)

type App struct {
	ctx         context.Context
	DesktopLock *desktoplock.API
	PyEnv       *pyenv.InstallerAPI
}

func NewApp() *App {
	return &App{
		DesktopLock: desktoplock.NewAPI(),
		PyEnv:       pyenv.NewInstallerAPI(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.DesktopLock.Startup(ctx)
	a.PyEnv.Startup(ctx)

	// 启动系统托盘（在独立 goroutine 中运行消息循环）
	go a.startTray()
}

func (a *App) shutdown(ctx context.Context) {
	systray.Quit()
}

// ── 系统托盘 ──

func (a *App) startTray() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) onTrayReady() {
	systray.SetIcon(appIconBytes)
	systray.SetTitle("码力工坊")
	systray.SetTooltip("码力工坊 — 机器人编程教学工具套件")

	// 左键点击托盘图标 → 显示主窗口
	systray.SetOnTapped(func() {
		a.showMainWindow()
	})

	// 右键菜单
	openItem := systray.AddMenuItem("显示窗口", "显示码力工坊主窗口")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出码力工坊")

	go func() {
		for range openItem.ClickedCh {
			a.showMainWindow()
		}
	}()
	go func() {
		for range quitItem.ClickedCh {
			systray.Quit()
			a.quitApp()
		}
	}()
}

func (a *App) onTrayExit() {
	// 清理工作
}

// ── 窗口管理 ──

func (a *App) showMainWindow() {
	wailsRuntime.WindowShow(a.ctx)
}

func (a *App) quitApp() {
	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
}
