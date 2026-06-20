# 码力工坊 / Wintools

机器人编程教学桌面工具套件。面向课堂场景，帮助老师管理学生电脑。

> 微信: asd3672830 | [GitHub](https://github.com/manfengjun/wintools) | [Gitee](https://gitee.com/3672830/wintools)

---

## 功能

| 功能 | 说明 |
|------|------|
| **桌面锁** | 锁定桌面快捷方式，防止学生误删。锁定期间自动备份，删除即恢复，退出需管理密码 |
| **Python 环境** | 一键部署 Python 3.12 + pygame/numpy 等常用库，国内镜像加速 |
| **管理密码** | 默认密码 `admin123`，可在设置中修改 |
| **检查更新** | 支持 Gitee / GitHub / 自定义更新源 |

## 技术栈

| 层 | 技术 |
|----|------|
| 桌面框架 | [Wails v2](https://wails.io) (Go + WebView2) |
| 后端 | Go 1.23+ |
| 前端 | Vue 3 + Vite 5 |
| 路由 | Vue Router (hash) |
| 样式 | 自定义 CSS (无 UI 框架) |
| 图标提取 | 纯 Windows GDI API (无外部依赖) |
| 通知 | Wails Events + 前端 Toast |
| 日志 | 文件日志 `%APPDATA%\DesktopSuite\logs\app.log` |
| 配置 | JSON 文件 `%APPDATA%\DesktopSuite\config.json` |

## 目录结构

```
apps/wintools/
├── main.go              # 入口：Wails 初始化、绑定
├── app.go               # App 生命周期、systray、退出流程
├── wails.json           # Wails 配置
├── go.mod               # Go 模块
├── internal/
│   ├── common/
│   │   ├── logger.go    # 文件日志
│   │   ├── notify.go    # 统一通知 (EventsEmit → 前端 Toast)
│   │   ├── config.go    # 配置读写
│   │   └── admin.go     # 管理员提权
│   ├── desktoplock/
│   │   ├── locker.go    # 密码管理、锁定/解锁、状态
│   │   ├── backup.go    # 桌面快捷方式备份/恢复/扫描
│   │   └── icons.go     # GDI 图标提取 (零外部依赖)
│   ├── pyenv/
│   │   ├── installer.go # Python 安装流程
│   │   ├── mirror.go    # 镜像源配置
│   │   └── util.go      # zip 解压、pip 安装
│   └── updater/
│       ├── checker.go   # 版本检测、下载、更新
│       └── api.go       # 前端 API 绑定
├── frontend/
│   ├── src/
│   │   ├── App.vue      # 主布局、全局 Toast、退出验证
│   │   ├── views/       # 页面视图
│   │   │   ├── DesktopLock.vue  # 桌面锁页
│   │   │   ├── PyEnv.vue        # Python 环境页
│   │   │   └── Settings.vue     # 设置模态框
│   │   ├── components/  # 通用组件
│   │   ├── locales/     # 中英文翻译
│   │   └── style.css    # 全局样式
│   └── wailsjs/         # Wails 自动生成绑定
└── build/               # 构建输出
```

## 构建

### 前置条件

| 依赖 | 版本 | 安装方式 |
|------|------|----------|
| Go | ≥ 1.23 | `winget install Go` |
| Node.js | ≥ 18 | `winget install NodeJS` |
| Wails CLI | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| NSIS | 3.x | [NSIS Download](https://nsis.sourceforge.io/Download)（生成安装包所需） |

### 构建命令

```powershell
# 开发模式 (前端热重载)
cd apps/wintools
wails dev

# 生产安装包（包含 WebView2 联网引导程序）
./build.ps1 build

# 一键升级版本 + 构建 + 发布（会提交、打标签并推送）
./build.ps1 all patch
```

### 构建产物

```
apps/wintools/build/bin/Wintools.exe             # 本地调试程序
apps/wintools/build/bin/*installer*.exe          # 对外发布的 NSIS 安装包
```

## 架构设计原则

1. **零外部依赖** — 图标提取用纯 Windows API，不依赖 PowerShell/.NET
2. **统一通知** — 所有弹窗/通知通过 `EventsEmit("notify")` 到前端 Toast
3. **文件日志** — 所有操作记录到 `%APPDATA%\DesktopSuite\logs\app.log`
4. **安全速率限制** — 密码连续 5 次错误后延迟 30 秒
5. **职责分离** — `locker.go` / `backup.go` / `icons.go` 各司其职
6. **前端无 UI 框架** — 纯自定义 CSS，无需 Element Plus 等额外依赖

## 常见问题

**Q: 为什么图标提取不需要 PowerShell？**
A: 使用 Windows GDI API (`SHGetFileInfoW` + `DrawIconEx` + `GetDIBits`)，纯原生调用，零外部依赖。

**Q: 检测更新失败 (404)?**
A: 确保仓库已发布 Release，且在设置中选择了正确的更新源 (Gitee/GitHub)。

**Q: 桌面锁定时如何退出？**
A: 右键系统托盘图标 → 退出 → 输入管理密码。默认密码 `admin123`。

**Q: Python 环境安装失败？**
A: 在设置中检查镜像源配置，国内用户推荐清华 Tuna 镜像。

**Q: 如何修改管理密码？**
A: 设置 → 管理密码，默认密码 `admin123`，首次修改无需旧密码。

## 版本发布流程

```powershell
# 1. 更新版本号 (internal/updater/checker.go:CurrentVersion)
# 仅构建并检查安装包
./build.ps1 build

# 全自动：升级版本、构建 NSIS、提交、打标签并发布到 Gitee/GitHub
./build.ps1 all patch
```

安装包内嵌的是微软 WebView2 在线引导程序；目标电脑缺少 Runtime 时需要联网完成安装。

## 开源协议

[MIT License](LICENSE) © 2026 manfengjun

微信: asd3672830
