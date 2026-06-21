package desktoplock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

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

// originFile 返回备份目录中记录快捷方式来源的清单文件路径。
func originFile() string { return filepath.Join(backupDir(), ".origin.json") }

// saveOrigin 保存快捷方式来源清单：{文件名: "user"|"public"}
func saveOrigin(origin map[string]string) error {
	data, _ := json.MarshalIndent(origin, "", "  ")
	os.MkdirAll(backupDir(), 0755)
	return os.WriteFile(originFile(), data, 0644)
}

// loadOrigin 读取快捷方式来源清单。
func loadOrigin() map[string]string {
	data, err := os.ReadFile(originFile())
	if err != nil {
		return nil
	}
	var origin map[string]string
	json.Unmarshal(data, &origin)
	return origin
}

// scanDesktopShortcuts 扫描用户桌面和公用桌面，返回所有 .lnk/.url 文件名。
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

// scanDesktopShortcutsWithOrigin 同 scanDesktopShortcuts，同时返回来源映射。
func scanDesktopShortcutsWithOrigin() ([]string, map[string]string) {
	origin := map[string]string{}
	seen := map[string]bool{}
	var result []string

	userDir := desktopPath()
	pubDir := publicDesktopPath()

	for _, pair := range [][2]string{{userDir, "user"}, {pubDir, "public"}} {
		dir, label := pair[0], pair[1]
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
				origin[name] = label
			}
		}
	}
	return result, origin
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
		if name == ".origin.json" {
			continue
		}
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

// scanUserShortcuts 仅扫描用户桌面目录，不扫描公用桌面。
// 用于锁定保护，避免公用桌面图标被复制到用户桌面造成重复。
func scanUserShortcuts() []string {
	desktop := desktopPath()
	f, err := os.Open(desktop)
	if err != nil {
		return nil
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil
	}

	var result []string
	for _, name := range names {
		low := strings.ToLower(name)
		if strings.HasSuffix(low, ".lnk") || strings.HasSuffix(low, ".url") {
			if info, err := os.Stat(filepath.Join(desktop, name)); err == nil && !info.IsDir() {
				result = append(result, name)
			}
		}
	}
	return result
}

// Backup 备份桌面快捷方式到备份目录（扫描用户桌面 + 公用桌面，供手动备份使用）。
func (a *API) Backup() BackupResult {
	names, origin := scanDesktopShortcutsWithOrigin()
	result := backupShortcuts(names)
	if len(origin) > 0 {
		saveOrigin(origin)
	}
	return result
}

// backupLockShortcuts 仅备份用户桌面快捷方式（供 Lock 使用）。
func backupLockShortcuts() error {
	bd := backupDir()
	os.MkdirAll(bd, 0755)
	origin := map[string]string{}
	for _, name := range scanUserShortcuts() {
		src := filepath.Join(desktopPath(), name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(bd, name), data, 0644); err != nil {
			continue
		}
		origin[name] = "user"
	}
	if len(origin) > 0 {
		saveOrigin(origin)
	}
	return nil
}

// backupShortcuts 将给定的快捷方式列表备份到备份目录。
func backupShortcuts(names []string) BackupResult {
	bd := backupDir()
	os.MkdirAll(bd, 0755)
	ok := 0
	skipped := 0

	for _, name := range names {
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
// 根据 .origin.json 记录的来源恢复到对应目录，避免重复。
func (a *API) Restore() RestoreResult {
	bd := backupDir()
	if _, err := os.Stat(bd); os.IsNotExist(err) {
		return RestoreResult{Error: "没有找到备份"}
	}
	origin := loadOrigin()
	userDir := desktopPath()
	pubDir := publicDesktopPath()
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
		if name == ".origin.json" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// 根据来源选择目标目录
		var target string
		switch origin[name] {
		case "public":
			target = filepath.Join(pubDir, name)
		default:
			target = filepath.Join(userDir, name)
		}

		if _, err := os.Stat(target); err == nil {
			skipped++
			continue
		}
		os.WriteFile(target, data, 0644)
		restored++
	}

	if restored > 0 {
		common.Info("恢复 %d 个快捷方式", restored)
		refreshDesktop(userDir)
		refreshDesktop(pubDir)
	}
	return RestoreResult{Restored: restored, Skipped: skipped}
}

// SHChangeNotify 通知 Windows Shell 刷新桌面视图。
const (
	shcnfPath  = 0x0001
	shcnfFlush = 0x1000
	shcneUpdateDir = 0x0008
)

func refreshDesktop(dir string) {
	pathPtr, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return
	}
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("SHChangeNotify")
	proc.Call(
		uintptr(shcneUpdateDir),
		uintptr(shcnfPath|shcnfFlush),
		uintptr(unsafe.Pointer(pathPtr)),
		0,
	)
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
