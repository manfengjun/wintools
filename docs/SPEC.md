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

更新流程:
  前端检查更新 → Go Check()
    → 读取 config.json update_url
    → HTTP GET Releases API
    → 解析 JSON、比较版本号
    → 返回 UpdateInfo
  前端下载 → Go Download(url)
    → HTTP GET → 写入 %TEMP%/wintools_update.exe
  前端应用 → Go Apply(path)
    → 写批处理脚本 → cmd /c → 替换 exe → 重启
```

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
