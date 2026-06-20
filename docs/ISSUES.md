# 问题记录

## 2026-06-20：查看备份时不显示快捷方式图标

### 现象

桌面快捷方式可以正常备份，备份列表也能显示文件名，但所有图标始终为空，只显示占位图标。

### 根因

`internal/desktoplock/icons.go` 调用了不存在的 Windows API 入口 `GetIconInfoW`。该 API 的正确入口名称是 `GetIconInfo`，没有 `W` 后缀。

原实现还在异步批量提取过程中用 `recover()` 吞掉了 panic，并生成 `.fail` 缓存，因此界面只能收到空图标，日志中也没有直接错误。

### 修复

- 将 API 入口改为 `GetIconInfo`。
- `ListBackups` 顺序读取快捷方式图标，并直接填充 `BackupItem.icon_base64`。
- 前端直接渲染 `ListBackups` 返回的图标。
- 删除图标预热、失败标记缓存、目标路径调试文件和前端二次图标请求。
- 单个图标提取失败时保留该备份条目，并显示默认占位图标。

### 验证

- 24 个真实 `.lnk` 备份均成功生成可见 PNG。
- 备份列表检测到 24 个删除按钮及对应图标。
- `go test ./...` 通过。
- 前端 Node 测试通过（2/2）。
- `wails build` 生产构建通过。

### 排查提示

Windows DLL 过程名错误可能在 `syscall.LazyProc.Call` 时触发 panic。不要无日志地吞掉图标提取异常；遇到全量图标为空时，应先验证 DLL 入口名称，再检查前端数据绑定。
