# Wintools 抖音视频自动化

本目录包含四条 1080×1920 竖屏视频的录制、配音文案和渲染配置：桌面锁、Python 环境一键安装、软件全貌、开源与下载地址。

桌面锁和软件界面录制使用仅在 `demo` 构建标签中启用的控制入口。Python 安装画面是安全演示，不会修改电脑上的真实 Python 环境；正式发布构建不包含任何演示入口。

```powershell
$env:WINTOOLS_DEMO_PASSWORD = '<当前管理密码>'
go build -tags "demo debug desktop production" -ldflags "-H windowsgui" -o build/bin/Wintools-demo.exe .
./video/record-desktop-lock.ps1
./video/record-python-env.ps1 -DemoCommand python-demo -OutputDir '<输出目录>'
./video/record-python-env.ps1 -DemoCommand overview-demo -OutputDir '<输出目录>'
```

脚本只操作桌面上的 `Wintools演示.lnk`，如果同名快捷方式或备份在运行前已经存在，脚本会立即停止。

配音使用 `zh-CN-YunyangNeural` 男声生成，最终成片由 FFmpeg 合成。第 4 条直接渲染项目 README 中维护的 GitHub、Gitee 和 Releases 下载信息，不依赖网页加载。
