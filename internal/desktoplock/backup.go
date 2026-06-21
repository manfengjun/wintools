package desktoplock

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/manfengjun/wintools/internal/common"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ── Directories ────────────────────────────────────────────

func appDataDir() string {
	d := os.Getenv("APPDATA")
	if d == "" {
		d = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(d, "DesktopSuite")
}

func configPath() string { return filepath.Join(appDataDir(), "lock-config.json") }
func backupDir() string  { return filepath.Join(appDataDir(), "lock-backup") }

// desktopPath 从注册表读取实际桌面目录路径。
// OneDrive 已知文件夹移动或企业重定向时，实际桌面不是 USERPROFILE\Desktop。
// 使用 Windows ExpandEnvironmentStringsW API 展开 %VAR% 格式环境变量。
func desktopPath() string {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`,
		registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		val, _, err := k.GetStringValue("Desktop")
		if err == nil && val != "" {
			// Windows 环境变量为 %VAR% 格式，用 Win32 ExpandEnvironmentStringsW 展开
			if expanded := expandWindowsEnv(val); expanded != "" {
				return expanded
			}
			return val
		}
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
}

// expandWindowsEnv 调用 Win32 ExpandEnvironmentStringsW 展开 %VAR% 格式环境变量。
func expandWindowsEnv(input string) string {
	src, err := syscall.UTF16PtrFromString(input)
	if err != nil {
		return ""
	}
	// 先查询所需缓冲区大小
	n, _ := windows.ExpandEnvironmentStrings(src, nil, 0)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n)
	windows.ExpandEnvironmentStrings(src, &buf[0], n)
	return syscall.UTF16ToString(buf)
}

// publicDesktopPath 返回公用桌面路径（如 C:\Users\Public\Desktop）。
// 系统级安装的程序（Chrome、微信、QQ 等）的快捷方式放在这里。
func publicDesktopPath() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`,
		registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		val, _, err := k.GetStringValue("Common Desktop")
		if err == nil && val != "" {
			if expanded := expandWindowsEnv(val); expanded != "" {
				return expanded
			}
			return val
		}
	}
	return filepath.Join(os.Getenv("PUBLIC"), "Desktop")
}

// scanDesktopShortcuts 扫描用户桌面和公用桌面，返回所有 .lnk / .url 文件名。
// 公用桌面存放系统级安装的程序快捷方式（如 Chrome、微信、QQ），
// Windows Explorer 会在桌面视图合并显示两个目录的内容。
func scanDesktopShortcuts() []string {
	seen := map[string]bool{}
	var result []string

	// Windows Explorer 合并显示用户桌面和公用桌面，两个目录都要扫描
	for _, dir := range []string{desktopPath(), publicDesktopPath()} {
		if dir == "" {
			continue
		}
		f, err := os.Open(dir)
		if err != nil {
			continue
		}
		names, err := f.Readdirnames(-1)
		f.Close()
		if err != nil {
			continue
		}
		for _, name := range names {
			if seen[name] {
				continue
			}
			low := strings.ToLower(name)
			if !strings.HasSuffix(low, ".lnk") && !strings.HasSuffix(low, ".url") {
				continue
			}
			info, err := os.Stat(filepath.Join(dir, name))
			if err == nil && !info.IsDir() {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// scanBackupDir 扫描备份目录，返回文件名列表（.lnk/.url 过滤）。
func scanBackupDir() []string {
	bd := backupDir()
	f, err := os.Open(bd)
	if err != nil {
		return nil
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil
	}

	var files []string
	for _, name := range names {
		path := filepath.Join(bd, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		low := strings.ToLower(name)
		if strings.HasSuffix(low, ".lnk") || strings.HasSuffix(low, ".url") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files
}

// Backup 备份桌面快捷方式到备份目录。
func (a *API) Backup() BackupResult {
	bd := backupDir()
	os.MkdirAll(bd, 0755)
	ok := 0
	skipped := 0

	for _, name := range scanDesktopShortcuts() {
		src := resolveShortcutPath(name)
		if src == "" {
			skipped++
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			skipped++
			continue
		}
		if err := os.WriteFile(filepath.Join(bd, name), data, 0644); err != nil {
			skipped++
			continue
		}
		ok++
	}

	common.Info("备份完成: %d 成功, %d 跳过", ok, skipped)
	return BackupResult{OK: ok, Skipped: skipped, Dir: bd}
}

// resolveShortcutPath 在用户桌面和公用桌面中查找快捷方式的完整路径。
func resolveShortcutPath(name string) string {
	for _, dir := range []string{desktopPath(), publicDesktopPath()} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Restore 从备份目录恢复缺失的快捷方式到桌面。
func (a *API) Restore() RestoreResult {
	bd := backupDir()
	if _, err := os.Stat(bd); os.IsNotExist(err) {
		return RestoreResult{Error: "没有找到备份"}
	}
	desktop := desktopPath()
	restored := 0
	skipped := 0

	pattern := filepath.Join(bd, "*")
	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		name := filepath.Base(path)
		target := filepath.Join(desktop, name)
		if _, err := os.Stat(target); err == nil {
			skipped++
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		os.WriteFile(target, data, 0644)
		restored++
	}

	if restored > 0 {
		common.Info("恢复 %d 个快捷方式", restored)
	}
	return RestoreResult{Restored: restored, Skipped: skipped}
}

// ListBackups 返回备份文件列表。
func (a *API) ListBackups() []BackupItem {
	files := scanBackupDir()
	return buildBackupItems(backupDir(), files, iconDataURL)
}

func buildBackupItems(dir string, files []string, readIcon func(string) string) []BackupItem {
	items := make([]BackupItem, 0, len(files))
	for _, name := range files {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		var size int64
		modTime := ""
		if err == nil {
			size = info.Size()
			modTime = info.ModTime().Format("2006-01-02 15:04:05")
		}
		items = append(items, BackupItem{
			Name:       name,
			Size:       size,
			ModTime:    modTime,
			IconBase64: readIcon(path),
		})
	}
	return items
}

// DeleteBackup 删除单个备份文件。
func (a *API) DeleteBackup(name string) bool {
	path := filepath.Join(backupDir(), name)
	if err := os.Remove(path); err != nil {
		return false
	}
	common.Info("删除备份: %s", name)
	return true
}
