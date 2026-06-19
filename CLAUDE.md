# Wintools (码力工坊) 项目助手

## 项目概览

机器人编程教学桌面工具套件。Go + Wails v2 + Vue 3 构建的 Windows 桌面应用。

## 技术栈

- **框架**: Wails v2 (Go + WebView2)
- **后端**: Go 1.23+
- **前端**: Vue 3 + Vite 5
- **构建**: `wails build`
- **日志**: `%APPDATA%\DesktopSuite\logs\app.log`
- **配置**: `%APPDATA%\DesktopSuite\config.json`

## 目录结构

```
apps/wintools/
├── main.go              # Wails 入口 + 绑定
├── app.go               # App 生命周期 + systray
├── build.ps1            # 一键构建发布脚本
├── internal/
│   ├── common/          # logger / notify / config / admin
│   ├── desktoplock/     # locker / backup / icons
│   ├── pyenv/           # Python 环境安装
│   └── updater/         # 版本检测 + 更新
├── frontend/src/
│   ├── App.vue          # 主布局 + 全局 Toast
│   ├── views/           # DesktopLock / PyEnv / Settings
│   └── locales/         # zh.js / en.js
└── docs/SPEC.md         # 项目规范文档
```

## 关键规范

1. **零外部依赖** — 图标提取不用 PowerShell/.NET，纯 Windows API
2. **统一通知** — 所有弹窗走 `EventsEmit("notify")` → 前端 Toast
3. **文件日志** — 所有操作写入 `logs/app.log`
4. **密码速率限制** — 连续 5 次错误后延迟 30 秒
5. **职责分离** — `locker.go`(锁逻辑) / `backup.go`(备份恢复) / `icons.go`(图标)

## 发布流程

```powershell
.\build.ps1 all patch    # bump → build → tag → release
```

## 构建

```powershell
cd apps/wintools
wails build              # 输出 build/bin/Wintools.exe
```

## 关键文件

| 文件 | 作用 |
|------|------|
| `internal/updater/checker.go:CurrentVersion` | 版本号常量 |
| `internal/common/config.go:DefaultConfig` | 默认配置 |
| `wails.json` | Wails 构建配置 |
| `internal/desktoplock/locker.go` | 核心锁逻辑 |
| `frontend/src/views/Settings.vue` | 设置页（含更新源切换） |

## 联系方式

微信: asd3672830
GitHub: github.com/manfengjun/wintools
Gitee: gitee.com/manfengjun/wintools
