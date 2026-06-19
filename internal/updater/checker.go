package updater

import (
	"context"
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
	CurrentVersion = "1.0.0"
	DefaultCheckURL = "https://api.github.com/repos/jj/wintools/releases/latest"
)

// UpdateInfo 更新检测结果
type UpdateInfo struct {
	HasUpdate    bool   `json:"has_update"`
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	ReleaseNotes string `json:"release_notes"`
	Error        string `json:"error,omitempty"`
}

// parseVersion 解析语义化版本号，用于比较。
// "1.2.3" → []int{1,2,3}，"v1.2" → []int{1,2,0}
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
	// 补齐到 3 位
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

// Check 检查是否有新版本
func Check() UpdateInfo {
	// 从配置文件读取更新 URL，若未配置使用默认值
	cfg := common.LoadConfig()
	checkURL := cfg.UpdateURL
	if checkURL == "" {
		checkURL = DefaultCheckURL
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		return UpdateInfo{Error: "请求失败: " + err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{Error: "网络错误: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return UpdateInfo{HasUpdate: false, Version: CurrentVersion,
			Error: "未找到更新源，请确保项目已发布到 GitHub Releases"}
	}
	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{Error: fmt.Sprintf("服务器返回 %d", resp.StatusCode)}
	}

	body, _ := io.ReadAll(resp.Body)
	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return UpdateInfo{Error: "解析失败: " + err.Error()}
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return UpdateInfo{HasUpdate: false, Version: CurrentVersion}
	}
	if !greaterVersion(latest, CurrentVersion) {
		return UpdateInfo{HasUpdate: false, Version: CurrentVersion}
	}

	dlURL := ""
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip") {
			dlURL = asset.BrowserDownloadURL
			break
		}
	}

	return UpdateInfo{
		HasUpdate:     true,
		Version:       latest,
		DownloadURL:   dlURL,
		ReleaseNotes:  release.Body,
	}
}

// Download 下载更新文件到临时目录，返回路径
func Download(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
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

	// 对路径加引号防止空格问题
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

	// 优雅退出：通知前端即将关闭
	os.Exit(0)
	return ""
}
