package updater

import (
	"github.com/manfengjun/wintools/internal/common"
)

// API 暴露给前端的更新检查接口
type API struct{}

func NewAPI() *API {
	return &API{}
}

// CheckUpdate 检查更新，返回 UpdateInfo
func (a *API) CheckUpdate() UpdateInfo {
	return Check()
}

// DownloadUpdate 下载更新，返回下载文件路径
func (a *API) DownloadUpdate(url string) string {
	path, err := Download(url)
	if err != nil {
		return ""
	}
	return path
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
	return DefaultCheckURL
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
