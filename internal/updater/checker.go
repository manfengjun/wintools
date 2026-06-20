package updater

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/manfengjun/wintools/internal/common"
)

const (
	CurrentVersion = "1.0.2"

	// GitHub raw（无需 token，全球正常网络可用）
	VersionURLGitHubRaw = "https://raw.githubusercontent.com/manfengjun/wintools/master/VERSION"
	// GitHub API（无需 token，公开仓库可用）
	VersionURLGitHubAPI = "https://api.github.com/repos/manfengjun/wintools/contents/VERSION"

	// 下载地址模板
	DownloadURLGitHub = "https://github.com/manfengjun/wintools/releases/download/v%s/Wintools_Windows_x86_64.exe"
	DownloadURLGitee  = "https://gitee.com/3672830/wintools/releases/download/v%s/Wintools_Windows_x86_64.exe"

	// Gitee 手动下载页
	GiteeReleasesPage = "https://gitee.com/3672830/wintools/releases"
)

// UpdateInfo 更新检测结果
type UpdateInfo struct {
	HasUpdate    bool   `json:"has_update"`
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
	Error        string `json:"error,omitempty"`
}

// fetchVersion 从 URL 读取版本号（支持纯文本和 GitHub API JSON 两种格式）
func fetchVersion(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	// 尝试作为纯文本解析（raw 格式）
	text := strings.TrimSpace(string(body))
	if text != "" && !strings.HasPrefix(text, "{") {
		return text, nil
	}

	// 尝试作为 GitHub API JSON 解析
	var ghResp struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(body, &ghResp); err == nil && ghResp.Type == "file" {
		decoded, err := base64.StdEncoding.DecodeString(ghResp.Content)
		if err == nil {
			return strings.TrimSpace(string(decoded)), nil
		}
	}

	return "", fmt.Errorf("无法解析响应")
}

// Check 检查是否有新版本。
// 根据用户配置的更新源决定策略：
//   - GitHub（默认）：GitHub raw → GitHub API → 失败提示
//   - Gitee：提示 Gitee 需 token + 手动下载链接（仍尝试 GitHub 作为备选）
func Check() UpdateInfo {
	cfg := common.LoadConfig()
	isGitee := strings.Contains(cfg.UpdateURL, "gitee.com")

	urls := []string{VersionURLGitHubRaw, VersionURLGitHubAPI}

	var lastErr string
	for _, url := range urls {
		ver, err := fetchVersion(url)
		if err != nil {
			lastErr = "连接 GitHub 失败"
			continue
		}

		ver = strings.TrimPrefix(ver, "v")
		if ver == "" {
			continue
		}

		if !greaterVersion(ver, CurrentVersion) {
			return UpdateInfo{HasUpdate: false, Version: CurrentVersion}
		}

		dlURL := fmt.Sprintf(DownloadURLGitHub, ver)
		if isGitee {
			dlURL = fmt.Sprintf(DownloadURLGitee, ver)
		}

		return UpdateInfo{
			HasUpdate:    true,
			Version:      ver,
			DownloadURL:  dlURL,
			ReleaseNotes: fmt.Sprintf("发现新版本 v%s", ver),
		}
	}

	// 全部探测失败，根据配置源给出不同提示
	if isGitee {
		return UpdateInfo{
			Error: fmt.Sprintf("Gitee 更新检测需要配置访问令牌（Token）\n\n请手动前往 Gitee 下载最新版本:\n%s", GiteeReleasesPage),
		}
	}
	if lastErr != "" {
		return UpdateInfo{
			Error: fmt.Sprintf("%s\n\n提示：如果无法访问 GitHub，请手动前往 Gitee 下载最新版本:\n%s", lastErr, GiteeReleasesPage),
		}
	}
	return UpdateInfo{
		Error: fmt.Sprintf("未找到更新源\n\n请手动前往 Gitee 下载最新版本:\n%s", GiteeReleasesPage),
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

// Download 下载更新文件
func Download(url string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "wintools_update.exe")
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	return tmpFile, nil
}

// Apply 应用更新：批处理替换 exe 并重启
func Apply(updatePath string) string {
	currentExe, err := os.Executable()
	if err != nil {
		return "获取当前程序路径失败: " + err.Error()
	}

	batchContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
:retry
del /f /q "%s" 2>nul
if exist "%s" goto retry
copy /y "%s" "%s"
if exist "%s" start "" "%s"
del /f /q "%%~f0"
`, currentExe, currentExe, updatePath, currentExe, currentExe, currentExe)

	batchPath := filepath.Join(os.TempDir(), "wintools_update.bat")
	if err := os.WriteFile(batchPath, []byte(batchContent), 0644); err != nil {
		return "写入更新脚本失败: " + err.Error()
	}

	cmd := exec.Command("cmd", "/c", batchPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return "启动更新脚本失败: " + err.Error()
	}

	os.Exit(0)
	return ""
}
