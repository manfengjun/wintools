package updater

import (
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
	CurrentVersion = "1.0.1"

	// VERSION 文件地址（无需 API、无需 token，raw 直连）
	VersionURLGitHub = "https://raw.githubusercontent.com/manfengjun/wintools/master/VERSION"
	VersionURLGitee  = "https://gitee.com/3672830/wintools/raw/master/VERSION"

	// 下载地址模板（根据版本号拼接）
	DownloadURLGitHub = "https://github.com/manfengjun/wintools/releases/download/v%s/Wintools_Windows_x86_64.exe"
	DownloadURLGitee  = "https://gitee.com/3672830/wintools/releases/download/v%s/Wintools_Windows_x86_64.exe"
)

// UpdateInfo 更新检测结果
type UpdateInfo struct {
	HasUpdate    bool   `json:"has_update"`
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
	Error        string `json:"error,omitempty"`
}

// fetchVersion 从 URL 读取 VERSION 文件，返回版本号字符串。
func fetchVersion(url string) (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Check 检查是否有新版本。
// 依次尝试 GitHub raw → Gitee raw 读取 VERSION 文件，无需 API、无需 token。
func Check() UpdateInfo {
	// 获取更新源配置（用于用户自定义，非必需）
	cfg := common.LoadConfig()

	// 根据配置决定优先尝试哪个源
	urls := []string{VersionURLGitHub, VersionURLGitee}
	if cfg.UpdateURL != "" {
		// 如果用户配置了自定义源，优先尝试
		if strings.Contains(cfg.UpdateURL, "gitee.com") {
			urls = []string{VersionURLGitee, VersionURLGitHub}
		}
	}

	var lastErr string
	for _, url := range urls {
		ver, err := fetchVersion(url)
		if err != nil {
			lastErr = fmt.Sprintf("检测失败: %v", err)
			continue
		}

		// 解析版本号
		ver = strings.TrimPrefix(ver, "v")
		if ver == "" {
			continue
		}

		if !greaterVersion(ver, CurrentVersion) {
			// 已是最新
			return UpdateInfo{HasUpdate: false, Version: CurrentVersion}
		}

		// 有新版本，根据数据来源构造下载地址
		dlURL := fmt.Sprintf(DownloadURLGitHub, ver)
		if strings.Contains(url, "gitee.com") {
			dlURL = fmt.Sprintf(DownloadURLGitee, ver)
		}

		return UpdateInfo{
			HasUpdate:    true,
			Version:      ver,
			DownloadURL:  dlURL,
			ReleaseNotes: fmt.Sprintf("发现新版本 v%s，点击下载更新。", ver),
		}
	}

	// 全部失败
	if lastErr != "" {
		return UpdateInfo{Error: lastErr}
	}
	return UpdateInfo{
		Error: "未找到更新源，请确认网络连接正常",
	}
}

// parseVersion 解析语义化版本号。"1.2.3" → []int{1,2,3}
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

// greaterVersion 判断 a > b（语义化版本比较）
func greaterVersion(a, b string) bool {
	va, vb := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	return false
}

// Download 下载更新文件到临时目录，返回路径
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

// Apply 应用更新：用批处理脚本替换当前 exe 并重启
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
