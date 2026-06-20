package pyenv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/manfengjun/wintools/internal/common"
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
	Installed    bool   `json:"installed"`
	Version      string `json:"version"`
	PythonExe    string `json:"python_exe"`
	PipInstalled bool   `json:"pip_installed"`
}

// PackageInfo describes an installable Python package.
type PackageInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefaultOn   bool   `json:"default_on"`
}

// ── Dependency injection for testability ──────────────────

type installDependencies struct {
	downloadFile func(url, dest string) error
	runCmd       func(exe string, args ...string) (string, error)
	findPython   func(version string) (string, error)
	writeState   func(path string, state TaskState) error
}

// defaultDeps returns the production implementation of all dependencies.
func defaultDeps() installDependencies {
	return installDependencies{
		downloadFile: downloadFile,
		runCmd:       runCommand,
		findPython:   findInstalledPython,
		writeState:   writeTaskState,
	}
}

// ── Available packages ────────────────────────────────────

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
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
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

// ── Status ─────────────────────────────────────────────────

func (a *InstallerAPI) CheckStatus() InstallStatus {
	// Try official-installer locations first, then fall back to legacy path.
	exe, err := findInstalledPython("3.12")
	if err != nil {
		// Fallback to legacy embeddable location
		legacyPath := filepath.Join("C:\\", "Python", "3.12", "python.exe")
		if _, err2 := os.Stat(legacyPath); err2 == nil {
			exe = legacyPath
		} else {
			return InstallStatus{PythonExe: "C:\\Python\\3.12\\python.exe"}
		}
	}

	st := InstallStatus{PythonExe: exe}
	if _, err := os.Stat(exe); err == nil {
		st.Installed = true
		out, err := exec.Command(exe, "--version").Output()
		if err == nil {
			st.Version = strings.TrimSpace(string(out))
		}
		out2, err2 := exec.Command(exe, "-m", "pip", "--version").Output()
		if err2 == nil {
			st.PipInstalled = true
			_ = out2
		}
	}
	return st
}

// ── Elevated worker ───────────────────────────────────────

// RunInstallWorker executes the full Python installation sequence
// and writes progress to the state file at each step.
// It is designed to run in the elevated worker process.
func RunInstallWorker(req InstallRequest) TaskState {
	return runInstallWorker(req, defaultDeps())
}

func runInstallWorker(req InstallRequest, deps installDependencies) TaskState {
	step := func(s string) { deps.writeState(req.StatePath, TaskState{Step: s, Running: true}) }
	finish := func(s TaskState) TaskState {
		deps.writeState(req.StatePath, s)
		return s
	}

	// prepare
	state := TaskState{Step: "prepare", Message: "开始安装 Python " + req.PythonVersion, Percent: 0, Running: true}
	deps.writeState(req.StatePath, state)

	// download
	state.Step = "download"
	state.Message = "正在下载 Python " + req.PythonVersion + " 安装程序..."
	state.Percent = 10
	deps.writeState(req.StatePath, state)

	spec := officialInstallerSpec(req.PythonVersion)
	installerPath := filepath.Join(os.TempDir(), "python-installer-"+spec.URL[strings.LastIndex(spec.URL, "/")+1:])
	if err := deps.downloadFile(spec.URL, installerPath); err != nil {
		return finish(TaskState{Step: "error", Error: "下载安装程序失败: " + err.Error()})
	}
	step("install-python")
	deps.writeState(req.StatePath, TaskState{
		Step: "install-python", Message: "正在安装 Python（需要几分钟）...", Percent: 40, Running: true,
	})

	// install python (silent)
	// install python (silent)
	if out, err := deps.runCmd(installerPath, spec.Args...); err != nil {
		return finish(TaskState{Step: "error", Error: fmt.Sprintf("Python 安装程序退出错误: %v\n输出: %s", err, out)})
	}

	step("verify-python")
	deps.writeState(req.StatePath, TaskState{
		Step: "verify-python", Message: "正在验证 Python 安装...", Percent: 60, Running: true,
	})

	// find python
	pythonExe, err := deps.findPython(req.PythonVersion)
	if err != nil {
		return finish(TaskState{Step: "error", Error: "安装后未找到 Python: " + err.Error()})
	}

	// install packages
	packages := make([]PackageState, len(req.Packages))
	for i, pkg := range req.Packages {
		packages[i] = PackageState{Name: pkg, Status: "installing"}
	}

	state = TaskState{
		Step:      "install-package",
		Message:   fmt.Sprintf("正在安装 %d 个包...", len(req.Packages)),
		Percent:   70,
		Running:   true,
		PythonExe: pythonExe,
		Packages:  packages,
	}
	deps.writeState(req.StatePath, state)

	allSucceeded := true
	for i, pkg := range req.Packages {
		state.Message = fmt.Sprintf("正在安装 %s (%d/%d)...", pkg, i+1, len(req.Packages))
		state.Percent = 70 + float64(i+1)/float64(len(req.Packages))*20
		state.Packages[i].Status = "installing"
		deps.writeState(req.StatePath, state)

		if out, err := deps.runCmd(pythonExe, "-m", "pip", "install", "--no-warn-script-location", pkg); err != nil {
			state.Packages[i].Status = "failed"
			state.Packages[i].Error = err.Error()
			_ = out
			allSucceeded = false
		} else {
			state.Packages[i].Status = "success"
		}
		deps.writeState(req.StatePath, state)
	}

	// done
	state.Step = "done"
	state.Running = false
	state.Done = true
	state.Percent = 100
	if allSucceeded {
		state.Message = "安装完成！"
	} else {
		state.Message = "部分包安装失败"
	}
	deps.writeState(req.StatePath, state)
	return state
}

// ── Frontend API ──────────────────────────────────────────

func (a *InstallerAPI) InstallWithElevation(packages []string) ProgressInfo {
	if !common.IsAdmin() {
		return a.installWithElevationWorker(packages)
	}
	return a.installDirect(packages)
}

// installWithElevationWorker writes a request file, starts the elevated
// worker, polls the state file, and relays progress via Wails events.
func (a *InstallerAPI) installWithElevationWorker(packages []string) ProgressInfo {
	exe, err := os.Executable()
	if err != nil {
		return ProgressInfo{Step: "error", Error: "无法获取程序路径: " + err.Error()}
	}

	// Use config for version
	cfg := common.LoadConfig()
	pyVersion := cfg.PyVersion
	if pyVersion == "" {
		pyVersion = "3.12.0"
	}

	// Create task directory
	taskID := fmt.Sprintf("pyenv-%d", time.Now().UnixMilli())
	taskDir := filepath.Join(os.TempDir(), "Wintools", taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return ProgressInfo{Step: "error", Error: "无法创建任务目录: " + err.Error()}
	}

	statePath := filepath.Join(taskDir, "state.json")
	requestPath := filepath.Join(taskDir, "request.json")

	// Write request
	req := InstallRequest{
		PythonVersion: pyVersion,
		Packages:      packages,
		StatePath:     statePath,
	}
	if err := writeTaskState(requestPath, TaskState{}); err != nil {
		// Reuse write helper — marshal request manually
	}
	reqData, _ := json.Marshal(req)
	if err := os.WriteFile(requestPath, reqData, 0644); err != nil {
		return ProgressInfo{Step: "error", Error: "无法写入请求文件: " + err.Error()}
	}

	// Polling goroutine — reads statePath every 300ms and emits events
	done := make(chan struct{})
	defer close(done)

	var lastStep, lastMessage, lastError string
	var lastPercent float64
	var lastDone bool
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				state, err := readTaskState(statePath)
				if err != nil {
					continue
				}
				if state.Step == lastStep && state.Message == lastMessage &&
					state.Percent == lastPercent && state.Done == lastDone &&
					state.Error == lastError {
					continue
				}
				lastStep, lastMessage, lastError = state.Step, state.Message, state.Error
				lastPercent, lastDone = state.Percent, state.Done
				a.emit(state.Step, state.Message, state.Percent, state.Done, state.Error)
			case <-done:
				return
			}
		}
	}()

	a.emit("elevate", "正在请求管理员权限...", 0, false, "")

	// Start elevated worker
	err = common.RunElevatedWait(exe, []string{"--install-pyenv-worker", requestPath})
	if err != nil {
		return ProgressInfo{Step: "error", Error: "安装失败: " + err.Error()}
	}

	// Read final state
	finalState, err := readTaskState(statePath)
	if err != nil {
		// Check status as fallback
		st := a.CheckStatus()
		if st.Installed {
			a.emit("done", "安装完成！", 100, true, "")
			return ProgressInfo{Step: "done", Message: "安装完成！", Percent: 100, Done: true}
		}
		return ProgressInfo{Step: "error", Error: "安装似乎未完成，请检查 Python 环境"}
	}

	if finalState.Error != "" {
		return ProgressInfo{Step: "error", Error: finalState.Error}
	}

	a.emit("done", "安装完成！", 100, true, "")
	return ProgressInfo{
		Step:    "done",
		Message: "安装完成！",
		Percent: 100,
		Done:    true,
	}
}

// installDirect runs the installation directly (already elevated).
func (a *InstallerAPI) installDirect(packages []string) ProgressInfo {
	cfg := common.LoadConfig()
	req := InstallRequest{
		PythonVersion: cfg.PyVersion,
		Packages:      packages,
		StatePath:     filepath.Join(os.TempDir(), "wintools-pyenv-direct.json"),
	}
	state := runInstallWorker(req, defaultDeps())
	return ProgressInfo{
		Step:    state.Step,
		Message: state.Message,
		Percent: state.Percent,
		Done:    state.Done,
		Error:   state.Error,
	}
}
