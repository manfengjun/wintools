package desktoplock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/manfengjun/wintools/internal/common"
)

// ── Types ──────────────────────────────────────────────────

type API struct {
	ctx          context.Context
	config       *Config
	mu           sync.Mutex
	locked       bool
	quitCh       chan struct{}
	deletedCount int32
	lastAlertUnix int64
	failCount    int32
	lastFailTime int64
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

type BackupItem struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ModTime   string `json:"mod_time"`
	IconBase64 string `json:"icon_base64"`
}

const defaultPassword = "admin123"
const oldDefaultPassword = "123456"

// ── Lifecycle ──────────────────────────────────────────────

func NewAPI() *API {
	return &API{config: &Config{}}
}

func (a *API) Startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
}

// ── Password ───────────────────────────────────────────────

func hashPassword(pwd string) string {
	h := sha256.Sum256([]byte(pwd))
	return hex.EncodeToString(h[:])
}

func (a *API) VerifyPassword(pwd string) bool {
	// 速率限制：连续 5 次失败后需等待 30 秒
	count := atomic.LoadInt32(&a.failCount)
	last := atomic.LoadInt64(&a.lastFailTime)
	if count >= 5 {
		if time.Since(time.Unix(last, 0)).Seconds() < 30 {
			return false
		}
		// 超过 30 秒重置
		atomic.StoreInt32(&a.failCount, 0)
	}

	ok := a.config.PasswordHash == hashPassword(pwd)
	if !ok {
		atomic.AddInt32(&a.failCount, 1)
		atomic.StoreInt64(&a.lastFailTime, time.Now().Unix())
	} else {
		atomic.StoreInt32(&a.failCount, 0)
	}
	return ok
}

func (a *API) ChangePassword(oldPwd, newPwd string) (bool, string) {
	if !a.VerifyPassword(oldPwd) {
		return false, "当前密码不正确"
	}
	if len(newPwd) < 1 {
		return false, "密码不能为空"
	}
	// 比较哈希，而非明文
	if hashPassword(newPwd) == a.config.PasswordHash {
		return false, "新密码不能与旧密码相同"
	}
	a.config.PasswordHash = hashPassword(newPwd)
	if err := a.saveConfig(); err != nil {
		return false, "保存失败: " + err.Error()
	}
	common.Info("密码已修改")
	return true, "密码已修改"
}

func (a *API) ChangeDefaultPassword(newPwd string) (bool, string) {
	if !a.IsDefaultPassword() {
		return false, "当前密码不是默认密码，请使用修改密码功能"
	}
	if len(newPwd) < 1 {
		return false, "密码不能为空"
	}
	a.config.PasswordHash = hashPassword(newPwd)
	if err := a.saveConfig(); err != nil {
		return false, "保存失败: " + err.Error()
	}
	common.Info("默认密码已修改")
	return true, "密码已设置"
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
	if err := json.Unmarshal(data, a.config); err != nil {
		common.Warn("lock-config.json 解析失败，使用默认密码: %v", err)
		a.config = &Config{PasswordHash: hashPassword(defaultPassword)}
		return
	}
	if a.config.PasswordHash == hashPassword(oldDefaultPassword) {
		a.config.PasswordHash = hashPassword(defaultPassword)
		a.saveConfig()
	}
}

func (a *API) saveConfig() error {
	os.MkdirAll(appDataDir(), 0755)
	data, _ := json.MarshalIndent(a.config, "", "  ")
	return os.WriteFile(configPath(), data, 0644)
}

// ── Lock / Unlock ──────────────────────────────────────────

func (a *API) Lock() bool {
	a.mu.Lock()
	if a.locked {
		a.mu.Unlock()
		return true
	}
	a.locked = true
	a.mu.Unlock()

	// 不在锁内执行文件 I/O
	a.Backup()

	atomic.StoreInt32(&a.deletedCount, 0)
	atomic.StoreInt64(&a.lastAlertUnix, 0)

	a.quitCh = make(chan struct{})
	go func() {
		for {
			select {
			case <-a.quitCh:
				return
			case <-time.After(500 * time.Millisecond):
				r := a.Restore()
				if r.Restored > 0 {
					atomic.AddInt32(&a.deletedCount, int32(r.Restored))
					now := time.Now().Unix()
					last := atomic.LoadInt64(&a.lastAlertUnix)
					if last == 0 || now-last >= 60 {
						atomic.StoreInt64(&a.lastAlertUnix, now)
						if a.ctx != nil {
							common.SendWarn(a.ctx, "桌面锁定",
								"检测到快捷方式被删除，已自动恢复。")
						}
					}
				}
			}
		}
	}()

	common.Info("桌面已锁定")
	return true
}

func (a *API) Unlock() bool {
	a.mu.Lock()
	if !a.locked {
		a.mu.Unlock()
		return true
	}
	a.locked = false
	a.mu.Unlock()

	if a.quitCh != nil {
		close(a.quitCh)
		a.quitCh = nil
	}

	r := a.Restore()
	total := atomic.LoadInt32(&a.deletedCount) + int32(r.Restored)
	if total > 0 {
		if a.ctx != nil {
			msg := fmt.Sprintf("锁定期间有 %d 个快捷方式被删除，已自动恢复。", total)
			common.SendInfo(a.ctx, "桌面解锁", msg)
		}
	}

	common.Info("桌面已解锁")
	return true
}

// Status 返回当前状态。先读锁再 I/O，避免持锁阻塞。
func (a *API) Status() StatusResult {
	a.mu.Lock()
	locked := a.locked
	a.mu.Unlock()

	backupNames := map[string]bool{}
	for _, name := range scanBackupDir() {
		backupNames[name] = true
	}

	desktopNames := map[string]bool{}
	for _, name := range scanDesktopShortcuts() {
		desktopNames[name] = true
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
