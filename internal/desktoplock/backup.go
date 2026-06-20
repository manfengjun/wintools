package desktoplock

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/manfengjun/wintools/internal/common"
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

// scanDesktopShortcuts 扫描桌面，返回所有 .lnk / .url 文件名。
// 使用 os.Open + Readdirnames（避免 os.ReadDir 的 DirEntry.Info 在部分 Windows 上报错）。
func scanDesktopShortcuts() []string {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	f, err := os.Open(desktop)
	if err != nil {
		common.Warn("scanDesktopShortcuts: 打开桌面目录失败: %v", err)
		return nil
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		common.Warn("scanDesktopShortcuts: 读取桌面目录失败: %v", err)
		return nil
	}

	var result []string
	for _, name := range names {
		low := strings.ToLower(name)
		if strings.HasSuffix(low, ".lnk") || strings.HasSuffix(low, ".url") {
			path := filepath.Join(desktop, name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
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
		src := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", name)
		dst := filepath.Join(bd, name)
		data, err := os.ReadFile(src)
		if err != nil {
			skipped++
			continue
		}

		if err := os.WriteFile(dst, data, 0644); err != nil {
			skipped++
			continue
		}
		ok++
	}

	common.Info("备份完成: %d 成功, %d 跳过", ok, skipped)
	return BackupResult{OK: ok, Skipped: skipped, Dir: bd}
}

// Restore 从备份目录恢复缺失的快捷方式到桌面。
func (a *API) Restore() RestoreResult {
	bd := backupDir()
	if _, err := os.Stat(bd); os.IsNotExist(err) {
		return RestoreResult{Error: "没有找到备份"}
	}
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
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
