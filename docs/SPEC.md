# 码力工坊 项目规范

## 开发规范

### Go 后端

| 规范 | 说明 |
|------|------|
| 包名 | 全小写，单数形式 (`common`, `updater`) |
| 文件命名 | 蛇形命名 (`locker.go`, `backup.go`) |
| 导入顺序 | 标准库 → 第三方 → 内部 (空行分隔) |
| 错误处理 | 不静默忽略 `error`，记录日志 `common.Error(...)` |
| 日志 | 所有操作记录：`common.Info` / `Warn` / `Error` |
| 并发 | `sync.Mutex` 保护共享状态，`atomic` 用于计数器 |
| 超时 | 外部调用 (HTTP/进程) 必须使用 `context.WithTimeout` |

### Vue 前端

| 规范 | 说明 |
|------|------|
| 组件命名 | PascalCase (`DesktopLock.vue`) |
| 模板 | 使用 `t('key')` 引用翻译，不写死中文字符串 |
| 样式 | scoped，BEM-like 命名 |
| 弹窗 | 通过 `EventsOn("notify")` 统一 Toast，`EventsOn("request-quit")` 退出验证 |
| 状态管理 | `ref` / `reactive`，不引入 Vuex/Pinia |

### 语言文件

- `locales/zh.js`、`locales/en.js`
- 所有 UI 文本必须通过 `t(key)` 引用
- 新增翻译键时两个文件同时更新

## 架构图

```
┌─────────────────────────────────────────────────────┐
│  Wails App                                          │
│  ┌──────────────────────────────────────────────┐   │
│  │  Go Backend                                   │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐      │   │
│  │  │common    │ │desktoplock│ │updater   │      │   │
│  │  │ logger   │ │ locker    │ │ checker  │      │   │
│  │  │ notify   │ │ backup    │ │ api      │      │   │
│  │  │ config   │ │ icons     │ └──────────┘      │   │
│  │  └──────────┘ └──────────┘                    │   │
│  │  ┌────────────────────────────────────────┐   │   │
│  │  │ EventsEmit("notify") → 前端 Toast      │   │   │
│  │  │ EventsEmit("request-quit") → 密码验证   │   │   │
│  │  └────────────────────────────────────────┘   │   │
│  └──────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────┐   │
│  │  Frontend (Vue 3)                            │   │
│  │  App.vue → Sidebar + Toast + 退出密码验证    │   │
│  │  ├── DesktopLock.vue → 锁定/备份/恢复        │   │
│  │  ├── PyEnv.vue → Python 安装                │   │
│  │  └── Settings.vue → 主题/镜像/密码/更新/关于  │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## 数据流

```
锁定流程:
  前端点击锁定 → VerifyPassword → Go Lock()
    → Backup() 备份桌面快捷方式
    → 启动 watcher goroutine (500ms 轮询)
    → 检测到删除 → Restore() 恢复 → SendWarn() 前端通知
  前端解锁 → VerifyPassword → Go Unlock()
    → 停止 watcher
    → Restore() 最终恢复
    → SendInfo() 汇总通知
```

## 更新机制

### 检测流程

```
Check()                                          → UpdateInfo
  ├─ ① GitHub raw VERSION 文件 (首选，无限流)
  │   GET https://raw.githubusercontent.com/.../VERSION
  │   返回纯文本版本号，如 "1.0.7"
  │   greaterVersion("1.0.7", CurrentVersion) → HasUpdate
  │
  ├─ ② GitHub Release API (回退，60次/小时限流)
  │   GET https://api.github.com/repos/.../releases/latest
  │   解析 JSON → 匹配资源名 → 返回下载地址
  │
  └─ ③ 均失败 → Error: "检测失败，请手动下载更新"
```

### 下载流程

```
DownloadUpdate(url)                              → filePath
  ├─ HTTP GET 下载安装包 (32KB 缓冲块读取)
  ├─ 每 200ms 发送 update:download-progress 事件
  │   payload: { downloaded, total, percent }
  ├─ 写入 %TEMP%\wintools_update.exe
  └─ 下载完成发送 percent: 100
```

### 安装流程

```
ApplyUpdate(path)                                → "" (成功)
  ├─ exec.Command(installer, "/S")
  │   SysProcAttr: HideWindow, DETACHED_PROCESS
  ├─ 安装器作为独立进程启动（脱离父进程 Job Object）
  └─ 返回成功 → 前端调 ConfirmQuit() → 应用退出
      安装器继续运行 → UAC 提权 → 替换文件 → 启动新版
```

### 更新检测优先级

| 优先级 | 源 | URL | 限流 | 认证 |
|--------|---|-----|------|------|
| 1 (首选) | GitHub raw | `raw.githubusercontent.com/.../VERSION` | 无 | 无 |
| 2 (回退) | GitHub API | `api.github.com/repos/.../releases/latest` | 60次/时 | 无 |

### 关键文件

| 文件 | 作用 |
|------|------|
| `internal/updater/checker.go` | 版本比较、VERSION 文件检查、API 检查、下载 |
| `internal/updater/api.go` | Wails 绑定接口，含进度通知 |
| `VERSION` (repo 根部) | 纯文本版本号，供 raw 方式直读 |
| `build.ps1` | 构建发布入口，自动更新 VERSION |

## 更新问题记录

### v1.0.4 — 资源名不匹配

**问题:** GitHub Release 上传的资源名为 `Wintools_Windows_x86_64.exe`，但代码中查找 `Wintools_Windows_x86_64_Setup.exe`，导致 API 检查成功但找不到安装包。

**修复:** 将匹配名改为 `Wintools_Windows_x86_64.exe`（与 `$AssetName` 定义一致）。

### v1.0.5 — Apply 中 os.Exit(0) 导致前端挂起

**问题:** `Apply()` 在启动 batch 后调用 `os.Exit(0)`，Go 进程立即终止，Wails 无法返回响应给前端，JavaScript `await` 永远挂起。

**修复:** `Apply()` 只启动安装器后返回 `""`，由前端调用 `ConfirmQuit()`（Wails Go binding 官方退出方式）。

### v1.0.6 — cmd.exe 找不到 + batch 窗口闪烁

**问题:** Wails 打包后 `%PATH%` 可能不含 `C:\Windows\System32`，`exec.Command("cmd")` 失败。batch 的 `start` 命令对 GUI 程序行为不稳定且窗口闪烁。

**修复:** 去掉 batch 中间层，`exec.Command(installer, "/S")` 直接启动 NSIS 安装器，用完整路径避免 PATH 问题。

### v1.0.6 — UAC 提权弹窗被抑制

**问题:** NSIS 安装器需要管理员权限，`DETACHED_PROCESS` 下 UAC 弹窗可能被阻止。

**修复:** 确保安装器以分离进程启动，UAC 在 Windows 安全桌面上不受父进程状态影响。去掉了 `/S` 静默标志让安装器显示界面以便用户确认。

### v1.0.7 — GitHub API 限流

**问题:** 无认证 GitHub API 限流 60次/小时，频繁测试时触发 403。

**修复:** 新增 `raw.githubusercontent.com/.../VERSION` 检测方式，纯文本文件无 API 限流。VERSION 文件同时推送到 GitHub + Gitee。

## 规范与约束

### 更新机制规范

| 规范 | 说明 |
|------|------|
| VERSION 文件 | 每次发布前必须更新 `VERSION` 文件并提交推送 |
| 发布流程 | 必须使用 `build.ps1 release` 或 `build.ps1 all`，不手动创建 Release |
| 资源名 | GitHub/Gitee Release 资源名必须为 `Wintools_Windows_x86_64.exe` |
| 编码 | `checker.go` 中所有中文注释/字符串使用 UTF-8，禁止 GB2312/Latin-1 |
| 回退 | 检测源依次降级：raw → API → 手动下载提示 |
| 下载进度 | 新增下载功能必须通过 `update:download-progress` 事件报告进度 |

### 安装器行为约束

- NSIS 安装器必须支持 `/S` 静默安装
- 安装后必须自动启动新版应用（Wails NSIS 模板默认行为）
- 安装器需要管理员权限（`RequestExecutionLevel admin`），UAC 提权由 Windows 处理
- `Apply()` 不调用 `os.Exit()`，由前端通过 `ConfirmQuit()` 正常退出

### 构建约束

- 构建前确保 `master` 分支干净、测试通过
- `local-tokens.ps1` 包含双平台 token，此文件已被 `.gitignore` 排除
- `build.ps1 all` 修改 git 历史 + 推送远程，仅在确认发布时使用

## 关键配置

### 文件位置
```
%APPDATA%\DesktopSuite\
├── config.json        # 通用配置
├── lock-config.json   # 密码配置
├── lock-backup\       # 快捷方式备份
│   └── .iconcache\    # 图标缓存
└── logs\app.log       # 运行日志
```

### config.json
```json
{
  "mirror_url": "https://pypi.tuna.tsinghua.edu.cn/simple",
  "py_version": "3.12.0",
  "py_install_dir": "C:\\Python\\3.12",
  "update_url": "https://gitee.com/api/v5/repos/manfengjun/wintools/releases/latest"
}
```

## 发布流程

```powershell
# 全自动 (推荐)
.\build.ps1 all patch
# 等价于:
#   1. bump version (v1.0.0 → v1.0.1)
#   2. wails build
#   3. git tag v1.0.1 && git push (gitee + github)
#   4. gh release create (gitee + github)

# 分步执行
.\build.ps1 bump patch    # 版本号升级
.\build.ps1 build         # 本地构建
.\build.ps1 tag           # 打标签推送
.\build.ps1 release       # 发布 Release
```

## 依赖清单

### 运行时依赖 (用户需安装)
- **WebView2 Runtime** — Windows 10/11 自带，Win7/8 需手动安装
- 无其他运行时依赖

### 构建时依赖 (开发者需安装)
- Go 1.23+
- Node.js 18+
- Wails CLI v2
- GitHub CLI (`gh`) — 用于发布 Release

## 常见问题

**Q: 为什么图标提取不需要外部依赖？**
A: 使用 `SHGetFileInfoW` + `DrawIconEx` + `GetDIBits` 纯 Windows API，所有 Windows 系统自带。

**Q: 桌面锁的原理？**
A: 锁定：备份快捷方式 + 启动 watcher (500ms 轮询恢复)。解锁：停止 watcher + 最终恢复。不涉及注册表或权限修改。

**Q: 如何切换更新源？**
A: 设置 → 检查更新 → 点击 Gitee/GitHub/自定义。地址保存在 `config.json` 的 `update_url` 字段。
