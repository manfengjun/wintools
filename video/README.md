# Wintools 视频自动化

桌面锁录制使用仅在 `demo` 构建标签中启用的控制入口。正式发布构建不包含该入口。

```powershell
$env:WINTOOLS_DEMO_PASSWORD = '<当前管理密码>'
go build -tags "demo debug desktop production" -ldflags "-H windowsgui" -o build/bin/Wintools-demo.exe .
./video/record-desktop-lock.ps1
```

脚本只操作桌面上的 `Wintools演示.lnk`，如果同名快捷方式或备份在运行前已经存在，脚本会立即停止。
