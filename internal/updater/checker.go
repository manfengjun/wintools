package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	CurrentVersion = "1.1.1"

	// Gitee raw VERSION 文件（免认证，检测更新用）
	GiteeVersionRaw = "https://gitee.com/3672830/wintools/raw/master/VERSION"

	// 下载 URL 模板：Gitee Release 资源路径
	GiteeDownloadTemplate = "https://gitee.com/3672830/wintools/releases/download/v%s/Wintools_Windows_x86_64.exe"

	// Gitee Releases 页面（检测失败时提示用户手动前往）
	GiteeReleasesPage = "https://gitee.com/3672830/wintools/releases"

	// 默认更新源 URL（用于设置页面的显示与自定义）
	DefaultUpdateAPI = "https://gitee.com/3672830/wintools/releases"
)

// UpdateInfo 更新检测结果
type UpdateInfo struct {
	HasUpdate    bool   `json:"has_update"`
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
	Error        string `json:"error,omitempty"`
}

func parseWindowsProxy(server string) *url.URL {
	server = strings.TrimSpace(server)
	if strings.Contains(server, ";") {
		entries := strings.Split(server, ";")
		server = ""
		for _, entry := range entries {
			parts := strings.SplitN(strings.TrimSpace(entry), "=", 2)
			if len(parts) == 2 && (strings.EqualFold(parts[0], "https") || server == "") {
				server = parts[1]
				if strings.EqualFold(parts[0], "https") {
					break
				}
			}
		}
	}
	if server == "" {
		return nil
	}
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	proxyURL, err := url.Parse(server)
	if err != nil || proxyURL.Host == "" {
		return nil
	}
	return proxyURL
}

func windowsProxyURL() *url.URL {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer key.Close()

	enabled, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil || enabled == 0 {
		return nil
	}
	server, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return nil
	}
	return parseWindowsProxy(server)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: func(request *http.Request) (*url.URL, error) {
			if proxyURL, err := http.ProxyFromEnvironment(request); err != nil || proxyURL != nil {
				return proxyURL, err
			}
			return windowsProxyURL(), nil
		},
		DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", address)
		},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func checkRelease(endpoint, currentVersion string) UpdateInfo {
	client := newHTTPClient(15 * time.Second)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return UpdateInfo{Error: err.Error()}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Wintools/"+currentVersion)
	resp, err := client.Do(req)
	if err != nil {
		return UpdateInfo{Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateInfo{Error: "无法解析版本信息"}
	}

	version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if version == "" {
		return UpdateInfo{Error: "Release 缺少版本号"}
	}
	if !greaterVersion(version, currentVersion) {
		return UpdateInfo{Version: currentVersion}
	}

	for _, asset := range release.Assets {
		if asset.Name == "Wintools_Windows_x86_64.exe" {
			return UpdateInfo{
				HasUpdate:    true,
				Version:      version,
				DownloadURL:  asset.BrowserDownloadURL,
				ReleaseNotes: release.Body,
			}
		}
	}

	return UpdateInfo{Version: version, Error: "Release 中缺少安装包"}
}

// checkVersionRaw 从 raw URL 读取 VERSION 纯文本文件，检测是否有更新。
// 返回 (hasUpdate, version, ok)。ok=false 表示 raw 请求失败。
func checkVersionRaw(client *http.Client, url, currentVersion string) (bool, string, bool) {
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return false, "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", false
	}
	remoteVersion := strings.TrimSpace(string(body))
	if remoteVersion == "" {
		return false, "", false
	}
	if greaterVersion(remoteVersion, currentVersion) {
		return true, remoteVersion, true
	}
	return false, remoteVersion, true
}

// Check 检查最新版本。
// 返回 UpdateInfo，其中 HasUpdate=true 表示有新版本。
// 检测源：Gitee raw VERSION 文件。
func Check() UpdateInfo {
	client := newHTTPClient(10 * time.Second)

	// Gitee raw VERSION 文件 — 免认证直读
	hasUpdate, ver, ok := checkVersionRaw(client, GiteeVersionRaw, CurrentVersion)
	if ok {
		if hasUpdate {
			return UpdateInfo{
				HasUpdate:   true,
				Version:     ver,
				DownloadURL: fmt.Sprintf(GiteeDownloadTemplate, ver),
			}
		}
		return UpdateInfo{Version: CurrentVersion}
	}

	// 全部失败 → 提示手动前往 Gitee 下载
	return UpdateInfo{
		Error: fmt.Sprintf("检测失败，请手动下载更新\n%s", GiteeReleasesPage),
	}
}

// parseVersion 解析语义化版本号
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var nums []int
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		nums = append(nums, n)
	}
	for len(nums) < 3 {
		nums = append(nums, 0)
	}
	return nums[:3]
}

// greaterVersion 判断 a > b
func greaterVersion(a, b string) bool {
	va, vb := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	return false
}

// buildInstallerBatch 为给定的安装程序路径生成批处理脚本内容。
// 使用 start /wait 以分离方式启动 NSIS 安装器并等待其完成。
// 测试用辅助函数，当前 Apply 直接使用 exec.Command。
func buildInstallerBatch(installerPath string) string {
	return fmt.Sprintf("@echo off\r\nstart \"\" /wait \"%s\" /S\r\n", installerPath)
}

// Apply 启动安装器（分离进程），然后由前端关闭应用。
func Apply(updatePath string) string {
	// 写到临时文件方便调试
	logPath := filepath.Join(os.TempDir(), "wintools_apply.log")
	logMsg := fmt.Sprintf("Apply called with: %s\n", updatePath)
	os.WriteFile(logPath, []byte(logMsg), 0644)

	// 不加 /S 静默标志，让安装器显示正常 UI
	cmd := exec.Command(updatePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("启动安装程序失败: %s", err.Error())
		os.WriteFile(logPath, []byte(logMsg+errMsg+"\n"), 0644)
		return errMsg
	}
	return ""
}


