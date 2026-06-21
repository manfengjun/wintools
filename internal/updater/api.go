package updater

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/manfengjun/wintools/internal/common"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// API 暴露给前端的更新检查接口
type API struct {
	ctx context.Context
}

func NewAPI() *API {
	return &API{}
}

// Startup 由 Wails 在应用启动时调用，保存上下文用于发送事件
func (a *API) Startup(ctx context.Context) {
	a.ctx = ctx
}

// CheckUpdate 检查更新，返回 UpdateInfo
func (a *API) CheckUpdate() UpdateInfo {
	return Check()
}

// DownloadUpdate 下载更新（带进度通知），返回下载文件路径
func (a *API) DownloadUpdate(url string) string {
	client := newHTTPClient(900 * time.Second) // 15 分钟超时，大文件下载
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "wintools_update.exe")
	f, err := os.Create(tmpFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	total := resp.ContentLength
	downloaded := int64(0)
	lastTick := time.Now()
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return ""
			}
			downloaded += int64(n)

			// 每 200ms 发送一次进度
			if time.Since(lastTick) > 200*time.Millisecond && a.ctx != nil && total > 0 {
				lastTick = time.Now()
				percent := int(downloaded * 100 / total)
				wailsRuntime.EventsEmit(a.ctx, "update:download-progress", map[string]interface{}{
					"downloaded": downloaded,
					"total":      total,
					"percent":    percent,
				})
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return ""
		}
	}

	// 下载完成通知
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "update:download-progress", map[string]interface{}{
			"downloaded": downloaded,
			"total":      total,
			"percent":    100,
		})
	}

	return tmpFile
}

// ApplyUpdate 应用更新
func (a *API) ApplyUpdate(path string) string {
	return Apply(path)
}

// GetUpdateURL 返回当前配置的更新源 URL
func (a *API) GetUpdateURL() string {
	cfg := common.LoadConfig()
	if cfg.UpdateURL != "" {
		return cfg.UpdateURL
	}
	return DefaultUpdateAPI
}

// SetUpdateURL 设置更新源 URL
func (a *API) SetUpdateURL(url string) bool {
	cfg := common.LoadConfig()
	cfg.UpdateURL = url
	if err := common.SaveConfig(cfg); err != nil {
		return false
	}
	return true
}
