# Wintools 码力工坊

机器人编程教学桌面工具套件。面向课堂场景，帮助老师管理学生电脑。

## 功能

- **桌面锁** — 锁定桌面快捷方式，防止学生误删。锁定期间自动备份，删除即恢复。
- **Python 环境** — 一键部署 Python 3.12 开发环境，含 pygame 等常用库。
- **管理密码** — 防止学生通过退出程序绕过桌面锁。

## 下载

从 [Releases](https://github.com/manfengjun/wintools/releases) 下载最新版本。

## 构建

```powershell
cd apps/wintools
wails build
```

需要 Go 1.23+ 和 Wails v2。

## 开源协议

MIT
