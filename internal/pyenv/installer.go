package pyenv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"DesktopSuite/internal/common"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── Types ──────────────────────────────────────────────────

type InstallerAPI struct {
	ctx context.Context
}

type ProgressInfo struct {
	Step    string  `json:"step"`
	Message string  `json:"message"`
	Percent float64 `json:"percent"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
}

type InstallStatus struct {
	Installed     bool   `json:"installed"`
	Version       string `json:"version"`
	PythonExe     string `json:"python_exe"`
	PipInstalled  bool   `json:"pip_installed"`
}

// PackageInfo describes an installable Python package.
type PackageInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefaultOn   bool   `json:"default_on"`
}

// AvailablePackages returns the list of selectable packages.
func (a *InstallerAPI) AvailablePackages() []PackageInfo {
	return []PackageInfo{
		{ID: "pygame", Name: "pygame", Description: "游戏开发框架", DefaultOn: true},
		{ID: "numpy", Name: "numpy", Description: "科学计算库", DefaultOn: true},
		{ID: "matplotlib", Name: "matplotlib", Description: "数据可视化", DefaultOn: true},
		{ID: "pillow", Name: "Pillow", Description: "图像处理库", DefaultOn: true},
		{ID: "jieba", Name: "jieba", Description: "中文分词库", DefaultOn: true},
		{ID: "easygui", Name: "easygui", Description: "简易 GUI 对话框", DefaultOn: true},
		{ID: "certifi", Name: "certifi", Description: "SSL 根证书包", DefaultOn: true},
	}
}

const installDir = "C:\\Python\\3.12"

// ── Lifecycle ──────────────────────────────────────────────

func NewInstallerAPI() *InstallerAPI {
	return &InstallerAPI{}
}

func (a *InstallerAPI) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *InstallerAPI) emit(step, message string, percent float64, done bool, errMsg string) {
	info := ProgressInfo{
		Step:    step,
		Message: message,
		Percent: percent,
		Done:    done,
		Error:   errMsg,
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "pyenv:progress", info)
	}
	// Also print to console for debug
	fmt.Printf("[pyenv] %s: %s (%.0f%%)\n", step, message, percent)
}

// ── Download helpers ───────────────────────────────────────

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败 %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadFileWithProgress(url, dest, label string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败 %s: %w", label, err)
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	written := int64(0)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			written += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ── Installer ──────────────────────────────────────────────

func (a *InstallerAPI) CheckStatus() InstallStatus {
	exe := filepath.Join(installDir, "python.exe")
	st := InstallStatus{PythonExe: exe}
	if _, err := os.Stat(exe); err == nil {
		st.Installed = true
		// Try to get version
		out, err := exec.Command(exe, "--version").Output()
		if err == nil {
			st.Version = strings.TrimSpace(string(out))
		}
		// Check pip
		out2, err2 := exec.Command(exe, "-m", "pip", "--version").Output()
		if err2 == nil {
			st.PipInstalled = true
			_ = out2
		}
	}
	return st
}

func (a *InstallerAPI) InstallWithPackages(selectedPackages []string) ProgressInfo {
	cfg := common.LoadConfig()
	mirror := cfg.MirrorURL
	pyVersion := cfg.PyVersion
	targetDir := installDir

	a.emit("prepare", "开始安装 Python "+pyVersion, 0, false, "")

	// Step 1: Download Python embeddable
	zipName := fmt.Sprintf("python-%s-embed-amd64.zip", pyVersion)
	zipURL := fmt.Sprintf("https://www.python.org/ftp/python/%s/%s", pyVersion, zipName)
	zipPath := filepath.Join(os.TempDir(), zipName)

	a.emit("download", "正在下载 Python "+pyVersion+"...", 5, false, "")
	if err := downloadFileWithProgress(zipURL, zipPath, "Python "+pyVersion); err != nil {
		return ProgressInfo{Step: "download", Error: err.Error()}
	}

	// Step 2: Extract
	a.emit("extract", "正在解压...", 30, false, "")
	if err := extractZip(zipPath, targetDir); err != nil {
		return ProgressInfo{Step: "extract", Error: "解压失败: " + err.Error()}
	}

	// Step 3: Configure _pth
	a.emit("configure", "正在配置 Python 路径...", 45, false, "")
	if err := configurePth(targetDir); err != nil {
		return ProgressInfo{Step: "configure", Error: err.Error()}
	}

	// Step 4: Install pip
	a.emit("pip", "正在安装 pip...", 55, false, "")
	pythonExe := filepath.Join(targetDir, "python.exe")
	if err := installPip(pythonExe, mirror); err != nil {
		return ProgressInfo{Step: "pip", Error: "pip 安装失败: " + err.Error()}
	}

	// Step 5: Install packages
	if len(selectedPackages) > 0 {
		a.emit("packages", fmt.Sprintf("正在安装 %d 个包...", len(selectedPackages)), 70, false, "")
		if err := installPackages(pythonExe, mirror, selectedPackages); err != nil {
			a.emit("packages", "部分包安装失败: "+err.Error(), 80, false, "")
		}
	} else {
		a.emit("packages", "跳过包安装（未选择任何包）", 80, false, "")
	}

	// Step 6: Add system PATH
	a.emit("path", "正在配置系统 PATH...", 85, false, "")
	if err := addToSystemPath(targetDir); err != nil {
		a.emit("path", "PATH 配置需要手动处理: "+err.Error(), 85, false, "")
	}

	// Step 7: Verify
	a.emit("verify", "正在验证安装...", 92, false, "")
	st := a.CheckStatus()

	// Step 8: Done
	msg := fmt.Sprintf("安装完成！Python %s", st.Version)
	a.emit("done", msg, 100, true, "")
	return ProgressInfo{
		Step:    "done",
		Message: msg,
		Percent: 100,
		Done:    true,
	}
}

func (a *InstallerAPI) InstallWithElevation(packages []string) ProgressInfo {
	if !common.IsAdmin() {
		exe, _ := os.Executable()
		common.RunElevated(exe, []string{"--install-pyenv"})
		return ProgressInfo{Step: "elevate", Message: "正在请求管理员权限...", Percent: 0, Done: false}
	}
	return a.InstallWithPackages(packages)
}
