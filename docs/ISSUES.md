# 问题记录

## 2026-06-20：Python 安装使用 embeddable 方式不可靠

### 现象

Python 环境安装偶尔出现进度停滞、安装后找不到 python.exe、pip 未配置等异常。
用户看到进度条卡在某个百分比，或安装完成但 `CheckStatus` 仍报告未安装。

### 根因

原实现使用 embeddable Python ZIP（`python-*-embed-amd64.zip`），需要手动解压并配置 `python._pth` 文件
以启用 `import site` 和 `site-packages`，再自行安装 pip。这一流程对 Windows 系统差异敏感：

1. embeddable 包本质上是一个最小分发包，缺少标准库 DLL、注册表集成和 PATH 配置。
2. 解压到 `C:\Python\3.12` 后需要修改 `python._pth` 文件启用 `import site`。
3. pip 安装依赖手工下载 `get-pip.py` 再到目标 Python 执行。
4. 所有步骤在非提权进程中进行，无法进行系统级注册（如 `py.exe` 启动器注册）。
5. 进度通过 Wails Events 广播，跨 UAC 边界时事件不穿透，导致前端收不到进度更新。

### 修复

将 embeddable ZIP 部署全部替换为**官方 Python 安装程序**（`python-*-amd64.exe`）：

- 下载官方 exe 安装程序 → `UAC 提权` → 静默安装（`/quiet InstallAllUsers=1 PrependPath=1 Include_pip=1 Include_launcher=1`）
- 安装完成后通过标准路径发现 Python（`%ProgramFiles%\Python312\python.exe`）
- 使用 JSON 文件 IPC 跨 UAC 边界传递进度：提权子进程写 TaskState 到临时文件，普通进程 300ms 轮询并转发 Wails 事件
- 前端使用纯函数 reducer 管理进度状态，支持包级别状态（pending/installing/success/failed）和日志持久化

### 验证

- `go test ./internal/pyenv ./internal/common -v` 全部 PASS（13 个用例）
- 前端 Node 测试全部 PASS（9 个用例）
- `npm run build` Vite 构建成功
- `go vet` 通过（预存 tools/test_elevate.go 除外）

### 排查提示

Python 安装问题需要先确认是安装器下载失败、安装过程退出码非零、还是安装后发现路径失败。
子系统日志位于 `%APPDATA%\DesktopSuite\logs\app.log`，搜索 `[pyenv]` 可跟踪安装步骤。
工作器任务文件位于 `%TEMP%\Wintools\pyenv-*\state.json`，可直接查看终态。

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
