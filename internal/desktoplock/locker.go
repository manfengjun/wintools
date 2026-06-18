package desktoplock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ── Types ──────────────────────────────────────────────────

type API struct {
	ctx    context.Context
	config *Config
}

type Config struct {
	PasswordHash string `json:"password_hash"`
}

type StatusResult struct {
	Locked       bool     `json:"locked"`
	BackupCount  int      `json:"backup_count"`
	DesktopCount int      `json:"desktop_count"`
	Missing      []string `json:"missing,omitempty"`
}

type BackupResult struct {
	OK      int    `json:"ok"`
	Skipped int    `json:"skipped"`
	Dir     string `json:"backup_dir,omitempty"`
}

type RestoreResult struct {
	Restored int    `json:"restored"`
	Skipped  int    `json:"skipped"`
	Error    string `json:"error,omitempty"`
}

const defaultPassword = "123456"

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procSendMessage = user32.NewProc("SendMessageTimeoutW")
	procSetAttr     = kernel32.NewProc("SetFileAttributesW")
	procGetAttr     = kernel32.NewProc("GetFileAttributesW")
)

// ── Lifecycle ──────────────────────────────────────────────

func NewAPI() *API {
	return &API{config: &Config{}}
}

func (a *API) Startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
}

// ── Paths ──────────────────────────────────────────────────

func appDataDir() string {
	d := os.Getenv("APPDATA")
	if d == "" {
		d = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(d, "DesktopSuite")
}

func configPath() string   { return filepath.Join(appDataDir(), "lock-config.json") }
func backupDir() string    { return filepath.Join(appDataDir(), "lock-backup") }

// ── Password ───────────────────────────────────────────────

func hashPassword(pwd string) string {
	h := sha256.Sum256([]byte(pwd))
	return hex.EncodeToString(h[:])
}

func (a *API) VerifyPassword(pwd string) bool {
	return a.config.PasswordHash == hashPassword(pwd)
}

func (a *API) ChangePassword(oldPwd, newPwd string) (bool, string) {
	if !a.VerifyPassword(oldPwd) {
		return false, "当前密码不正确"
	}
	if len(newPwd) < 1 {
		return false, "密码不能为空"
	}
	a.config.PasswordHash = hashPassword(newPwd)
	if err := a.saveConfig(); err != nil {
		return false, "保存失败: " + err.Error()
	}
	return true, "密码已修改"
}

func (a *API) IsDefaultPassword() bool {
	return a.config.PasswordHash == hashPassword(defaultPassword)
}

func (a *API) loadConfig() {
	data, err := os.ReadFile(configPath())
	if err != nil {
		a.config = &Config{PasswordHash: hashPassword(defaultPassword)}
		return
	}
	json.Unmarshal(data, a.config)
}

func (a *API) saveConfig() error {
	os.MkdirAll(appDataDir(), 0755)
	data, _ := json.MarshalIndent(a.config, "", "  ")
	return os.WriteFile(configPath(), data, 0644)
}

// ── Registry ───────────────────────────────────────────────

const policiesPath = "Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\Explorer"

func setRegPolicy(name string, value uint32) bool {
	keyStr, _ := syscall.UTF16PtrFromString(policiesPath)
	valStr, _ := syscall.UTF16PtrFromString(name)
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	procCreateKey := advapi32.NewProc("RegCreateKeyExW")
	procSetValue := advapi32.NewProc("RegSetValueExW")

	var hkey uintptr
	const (
		HKEY_CURRENT_USER = 0x80000001
		REG_DWORD         = 4
		KEY_SET_VALUE     = 0x0002
	)
	ret, _, _ := procCreateKey.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(keyStr)),
		0, 0, 0, KEY_SET_VALUE, 0, uintptr(unsafe.Pointer(&hkey)), 0)
	if ret != 0 {
		return false
	}
	valBytes := []byte{byte(value), 0, 0, 0}
	ret, _, _ = procSetValue.Call(hkey, uintptr(unsafe.Pointer(valStr)), REG_DWORD, 0,
		uintptr(unsafe.Pointer(&valBytes[0])), 4)
	syscall.Syscall(procCreateKey.Addr(), 1, hkey, 0, 0)
	return ret == 0
}

func deleteRegPolicy(name string) {
	keyStr, _ := syscall.UTF16PtrFromString(policiesPath)
	valStr, _ := syscall.UTF16PtrFromString(name)
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	procOpenKey := advapi32.NewProc("RegOpenKeyExW")
	procDeleteValue := advapi32.NewProc("RegDeleteValueW")

	var hkey uintptr
	const KEY_SET_VALUE = 0x0002
	ret, _, _ := procOpenKey.Call(uintptr(0x80000001), uintptr(unsafe.Pointer(keyStr)),
		0, KEY_SET_VALUE, uintptr(unsafe.Pointer(&hkey)))
	if ret == 0 {
		procDeleteValue.Call(hkey, uintptr(unsafe.Pointer(valStr)))
		syscall.Syscall(procOpenKey.Addr(), 1, hkey, 0, 0)
	}
}

func refreshDesktop() {
	procSendMessage.Call(0xFFFF, 0x001A, 0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Policy"))), 0, 500, 0)
}

func lockFallback() {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	if desktop != "" {
		p, _ := syscall.UTF16PtrFromString(desktop)
		procSetAttr.Call(uintptr(unsafe.Pointer(p)), 0x4|0x2)
	}
}

func unlockRestoreAttr() {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	if desktop != "" {
		p, _ := syscall.UTF16PtrFromString(desktop)
		procSetAttr.Call(uintptr(unsafe.Pointer(p)), 0x80)
	}
}

func isDesktopHidden() bool {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	if desktop == "" {
		return false
	}
	p, _ := syscall.UTF16PtrFromString(desktop)
	attrs, _, _ := procGetAttr.Call(uintptr(unsafe.Pointer(p)))
	if attrs == 0xFFFFFFFF {
		return false
	}
	return attrs&0x4 != 0
}

// ── Public API ─────────────────────────────────────────────

func (a *API) Lock() bool {
	ok := setRegPolicy("NoDesktop", 1) && setRegPolicy("NoViewContextMenu", 1)
	if !ok {
		lockFallback()
	}
	refreshDesktop()
	return true
}

func (a *API) Unlock() bool {
	deleteRegPolicy("NoDesktop")
	deleteRegPolicy("NoViewContextMenu")
	unlockRestoreAttr()
	refreshDesktop()
	return true
}

func (a *API) Status() StatusResult {
	locked := isDesktopHidden()
	backupNames := map[string]bool{}
	entries, _ := os.ReadDir(backupDir())
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".lnk") || strings.HasSuffix(e.Name(), ".url")) {
			backupNames[e.Name()] = true
		}
	}
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	desktopNames := map[string]bool{}
	entries2, _ := os.ReadDir(desktop)
	for _, e := range entries2 {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".lnk") || strings.HasSuffix(e.Name(), ".url")) {
			desktopNames[e.Name()] = true
		}
	}
	var missing []string
	for n := range backupNames {
		if !desktopNames[n] {
			missing = append(missing, n)
		}
	}
	return StatusResult{
		Locked:       locked,
		BackupCount:  len(backupNames),
		DesktopCount: len(desktopNames),
		Missing:      missing,
	}
}

func (a *API) Backup() BackupResult {
	bd := backupDir()
	os.MkdirAll(bd, 0755)
	ok := 0
	skipped := 0
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	entries, _ := os.ReadDir(desktop)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !strings.HasSuffix(name, ".lnk") && !strings.HasSuffix(name, ".url") {
			continue
		}
		src := filepath.Join(desktop, e.Name())
		dst := filepath.Join(bd, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			skipped++
			continue
		}
		os.WriteFile(dst, data, 0644)
		ok++
	}
	return BackupResult{OK: ok, Skipped: skipped, Dir: bd}
}

func (a *API) Restore() RestoreResult {
	bd := backupDir()
	if _, err := os.Stat(bd); os.IsNotExist(err) {
		return RestoreResult{Error: "没有找到备份"}
	}
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	restored := 0
	skipped := 0
	entries, _ := os.ReadDir(bd)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		target := filepath.Join(desktop, e.Name())
		if _, err := os.Stat(target); err == nil {
			skipped++
			continue
		}
		src := filepath.Join(bd, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		os.WriteFile(target, data, 0644)
		restored++
	}
	return RestoreResult{Restored: restored, Skipped: skipped}
}

func (a *API) StartWatcher() {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			a.Restore()
		}
	}()
}
