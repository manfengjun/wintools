package main

import (
	"context"
	"runtime"
	"time"

	"fyne.io/systray"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/manfengjun/wintools/internal/common"
	"github.com/manfengjun/wintools/internal/desktoplock"
	"github.com/manfengjun/wintools/internal/pyenv"
	"github.com/manfengjun/wintools/internal/updater"
)

type App struct {
	ctx         context.Context
	DesktopLock *desktoplock.API
	PyEnv       *pyenv.InstallerAPI
	Updater     *updater.API
	quitting    bool
}

func NewApp() *App {
	return &App{
		DesktopLock: desktoplock.NewAPI(),
		PyEnv:       pyenv.NewInstallerAPI(),
		Updater:     updater.NewAPI(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.DesktopLock.Startup(ctx)
	a.PyEnv.Startup(ctx)

	go func() {
		time.Sleep(500 * time.Millisecond)
		a.startTray()
	}()
}

func (a *App) shutdown(ctx context.Context) {
	systray.Quit()
}

func (a *App) startTray() {
	defer func() {
		if r := recover(); r != nil {
			common.Error("systray panic: %v", r)
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) onTrayReady() {
	systray.SetIcon(appIconBytes)
	systray.SetTitle("码力工坊")
	systray.SetTooltip("码力工坊 — 机器人编程教学工具套件")

	systray.SetOnTapped(func() {
		a.showMainWindow()
	})

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
			a.quitApp()
		}
	}()
}

func (a *App) onTrayExit() {}

func (a *App) showMainWindow() {
	wailsRuntime.WindowShow(a.ctx)
}

// quitApp 触发退出流程：先显示窗口，再通知前端弹出密码验证弹窗。
func (a *App) quitApp() {
	if a.ctx == nil {
		return
	}
	wailsRuntime.WindowShow(a.ctx)
	wailsRuntime.EventsEmit(a.ctx, "request-quit")
}

// ConfirmQuit 由前端密码验证通过后调用，真正退出。
func (a *App) ConfirmQuit() {
	a.quitting = true
	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
}
