package pyenv

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

// ── Download helper ────────────────────────────────────────

type commandRunner func(name string, args ...string) ([]byte, error)

func downloadWithCurl(curlPath, url, dest string, run commandRunner) error {
	output, err := run(curlPath,
		"--fail", "--location", "--silent", "--show-error",
		"--connect-timeout", "10", "--max-time", "120",
		"--retry", "2", "--retry-delay", "1", "--retry-all-errors",
		"--output", dest, url,
	)
	if err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("curl 下载失败: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(dest)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(dest)
		return fmt.Errorf("curl 未生成有效文件")
	}
	return nil
}

func systemCurlPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "curl.exe")
}

func runCommandOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func downloadWithSystemCurl(url, dest string) error {
	curlPath := systemCurlPath()
	if _, err := os.Stat(curlPath); err != nil {
		return fmt.Errorf("系统 curl 不可用: %w", err)
	}
	return downloadWithCurl(curlPath, url, dest, runCommandOutput)
}

func downloadFileWith(url, dest string, curlDownload, goDownload func(string, string) error) error {
	if err := curlDownload(url, dest); err == nil {
		return nil
	}
	return goDownload(url, dest)
}

func downloadFromSources(
	sources []DownloadSource,
	dest string,
	download func(string, string) error,
	validate func(string) error,
	onAttempt func(DownloadSource),
) error {
	if validate(dest) == nil {
		return nil
	}
	_ = os.Remove(dest)
	part := dest + ".part"
	var failures []string
	for _, source := range sources {
		_ = os.Remove(part)
		if onAttempt != nil {
			onAttempt(source)
		}
		if err := download(source.URL, part); err != nil {
			failures = append(failures, source.Name+": "+err.Error())
			continue
		}
		if err := validate(part); err != nil {
			failures = append(failures, source.Name+": "+err.Error())
			continue
		}
		_ = os.Remove(dest)
		if err := os.Rename(part, dest); err != nil {
			return err
		}
		return nil
	}
	_ = os.Remove(part)
	return fmt.Errorf("所有下载源均失败: %s", strings.Join(failures, "; "))
}

func validateNonEmpty(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("文件为空")
	}
	return nil
}

func validateZip(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) == 0 {
		return fmt.Errorf("ZIP 文件不包含任何条目")
	}
	return nil
}

func runWithPipMirrors(run func(index string) error) error {
	var failures []string
	for _, mirror := range DefaultMirrors {
		if err := run(mirror.URL); err == nil {
			return nil
		} else {
			failures = append(failures, mirror.Name+": "+err.Error())
		}
	}
	return fmt.Errorf("所有 pip 源均失败: %s", strings.Join(failures, "; "))
}

func getPipInstallBatch(pythonExe, getPipPath string) string {
	var batch strings.Builder
	// Embeddable Python lacks CA certs; disable HTTPS cert verification
	// for the get-pip.py process (safe for one-shot install).
	pipLog := filepath.Join(userTempDir(), "wintools-pip-install.log")
	batch.WriteString("set PYTHONHTTPSVERIFY=0\n")
	for _, mirror := range DefaultMirrors {
		host := extractHost(mirror.URL)
		fmt.Fprintf(&batch, `"%s" "%s" --no-warn-script-location --disable-pip-version-check --timeout 30 --retries 2 --index-url "%s" --trusted-host "%s" >>"%s" 2>&1
if not errorlevel 1 goto pip_ok
`, pythonExe, getPipPath, mirror.URL, host, pipLog)
	}
	batch.WriteString(fmt.Sprintf("type \"%s\" 2>nul\n", pipLog))
	batch.WriteString("if errorlevel 1 exit /b 1\n:pip_ok\n")
	return batch.String()
}

// readPipLog reads the pip install log for error details.
func readPipLog() string {
	logPath := filepath.Join(userTempDir(), "wintools-pip-install.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}
	return string(data)
}

var defaultHTTPClient = &http.Client{Timeout: 60 * time.Second}

func downloadFileWithHTTP(url, dest string) error {
	resp, err := defaultHTTPClient.Get(url)
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

func downloadFile(url, dest string) error {
	return downloadFileWith(url, dest, downloadWithSystemCurl, downloadFileWithHTTP)
}

// downloadFileWithProgress downloads a file using system curl when available,
// emitting progress events at 5% intervals.
func (a *InstallerAPI) downloadFileWithProgress(url, dest, label string) error {
	if err := downloadWithSystemCurl(url, dest); err == nil {
		a.emit("download", fmt.Sprintf("%s (100%%)", label), 25, false, "")
		return nil
	}

	resp, err := defaultHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败 %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	var written int64
	var lastPct float64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if totalSize > 0 {
				pct := float64(written) / float64(totalSize) * 100
				if pct-lastPct >= 5 {
					lastPct = pct
					a.emit("download", fmt.Sprintf("%s (%.0f%%)", label, pct), pct*0.25, false, "")
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// userTempDir returns the current user's temporary directory.
// Uses LOCALAPPDATA\Temp directly to avoid system TEMP when elevated.
func userTempDir() string {
	dir := os.Getenv("LOCALAPPDATA")
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(dir, "Temp")
}

// ── Status ─────────────────────────────────────────────────

func (a *InstallerAPI) CheckStatus() InstallStatus {
	exe, err := findInstalledPython("3.12")
	if err != nil {
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
		verCmd := exec.Command(exe, "--version")
		verCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := verCmd.Output()
		if err == nil {
			st.Version = strings.TrimSpace(string(out))
		}
		pipCmd := exec.Command(exe, "-m", "pip", "--version")
		pipCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out2, err2 := pipCmd.Output()
		if err2 == nil {
			st.PipInstalled = true
			_ = out2
		}
	}
	return st
}

// ── install helpers ────────────────────────────────────────

// waitForPythonInstall polls for python.exe to appear after installation.
// The installer bootstrapper may return before MSI completes, so we retry.
func waitForPythonInstall(fullVersion string, timeoutSec int) (string, error) {
	tag := strings.ReplaceAll(fullVersion, ".", "")
	if len(tag) > 3 {
		tag = tag[:3]
	}

	roots := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		"C:\\Python",
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		for _, root := range roots {
			candidate := filepath.Join(root, "Python"+tag, "python.exe")
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				return candidate, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("等待超时（%d 秒），未找到 python.exe", timeoutSec)
}

// getFullVersion normalises "3.12" to "3.12.0".
func getFullVersion(v string) string {
	if strings.Count(v, ".") == 1 {
		return v + ".0"
	}
	return v
}

// embedZipURL returns the download URL for the embeddable Python zip package.
func embedZipURL(fullVersion string) string {
	return fmt.Sprintf("https://www.python.org/ftp/python/%s/python-%s-embed-amd64.zip",
		fullVersion, fullVersion)
}

// installDir returns the target directory for a given Python version, e.g. "C:\Python312".
func installDir(fullVersion string) string {
	tag := strings.ReplaceAll(fullVersion, ".", "")
	if len(tag) > 3 {
		tag = tag[:3]
	}
	return fmt.Sprintf("C:\\Python%s", tag)
}

// ── Frontend API ──────────────────────────────────────────

// InstallPython downloads the embeddable zip, extracts it to C:\Python<tag>,
// configures pip, and adds Python to the system PATH.
// Elevation is required only for copying files into C:\ and setting PATH.
func (a *InstallerAPI) InstallPython() ProgressInfo {
	cfg := common.LoadConfig()
	pyVersion := cfg.PyVersion
	if pyVersion == "" {
		pyVersion = "3.12.0"
	}
	fullVersion := getFullVersion(pyVersion)
	targetDir := installDir(fullVersion)

	// ── 1. Download embeddable zip ──
	zipName := fmt.Sprintf("python-%s-embed-amd64.zip", fullVersion)
	zipPath := filepath.Join(userTempDir(), zipName)
	if err := downloadFromSources(
		PythonDownloadSources(fullVersion),
		zipPath,
		func(url, part string) error {
			fmt.Printf("[pyenv] 下载: %s → %s\n", url, part)
			return a.downloadFileWithProgress(url, part, "正在下载 Python "+fullVersion)
		},
		validateZip,
		func(source DownloadSource) {
			a.emit("download", "正在从 "+source.Name+" 下载 Python "+fullVersion, 5, false, "")
		},
	); err != nil {
		return ProgressInfo{Step: "error", Error: "下载失败: " + err.Error()}
	}

	// ── 2. Extract to temp dir (Go archive/zip, no elevation needed) ──
	a.emit("extract", "正在解压...", 30, false, "")
	tmpExtract := filepath.Join(userTempDir(), "wintools-python-"+fullVersion)
	if err := extractZip(zipPath, tmpExtract); err != nil {
		return ProgressInfo{Step: "error", Error: "解压失败: " + err.Error()}
	}

	// ── 3. Configure python._pth (enable import site so pip works) ──
	a.emit("config", "正在配置 pip 支持...", 45, false, "")
	if err := configurePth(tmpExtract); err != nil {
		fmt.Printf("[pyenv] configurePth 警告: %v\n", err)
	}

	// ── 4. Download pip wheel and extract into site-packages ──
	a.emit("install-pip", "正在安装 pip...", 50, false, "")
	// Resolve the actual pip whl URL from PyPI simple index
	pipWhlURL := ""
	pipWhlName := ""
	for _, src := range PipWhlSources() {
		a.emit("install-pip", "正在从 "+src.Name+" 查找 pip 最新版本...", 50, false, "")
		url, err := ResolvePipWhlURL(src.URL)
		if err == nil && url != "" {
			pipWhlURL = url
			pipWhlName = url[strings.LastIndex(url, "/")+1:]
			if idx := strings.Index(pipWhlName, "#"); idx >= 0 {
				pipWhlName = pipWhlName[:idx]
			}
			break
		}
	}
	if pipWhlURL == "" {
		return ProgressInfo{Step: "error", Error: "无法获取 pip 下载地址"}
	}
	a.emit("install-pip", "正在下载 "+pipWhlName+" ...", 52, false, "")
	pipWhlPath := filepath.Join(userTempDir(), pipWhlName)
	if err := downloadFile(pipWhlURL, pipWhlPath); err != nil {
		return ProgressInfo{Step: "error", Error: "下载 pip 安装包失败: " + err.Error()}
	}
	// Validate it's a real zip
	if err := validateZip(pipWhlPath); err != nil {
		return ProgressInfo{Step: "error", Error: "下载的 pip 文件无效: " + err.Error()}
	}
	// Extract pip wheel directly into site-packages (no network needed, no SSL issues)
	sitePkgDir := filepath.Join(tmpExtract, "Lib", "site-packages")
	if err := extractZip(pipWhlPath, sitePkgDir); err != nil {
		return ProgressInfo{Step: "error", Error: "解压 pip 失败: " + err.Error()}
	}

	// ── 5. Install: copy to target dir + set PATH ──
	a.emit("install", "正在安装到 "+targetDir+" ...", 70, false, "")
	scriptsDir := filepath.Join(targetDir, "Scripts")

	if common.IsAdmin() {
		// Already elevated — do it directly, no UAC prompt
		os.MkdirAll(targetDir, 0755)
		exec.Command("cmd.exe", "/c", "xcopy", tmpExtract, targetDir, "/E", "/I", "/Y").Run()
		exec.Command("setx", "/M", "PATH", targetDir+";"+scriptsDir+";%PATH%").Run()
	} else {
		// Need elevation — run a batch via ShellExecuteExW
		batContent := fmt.Sprintf(`@echo off
if not exist "%s" mkdir "%s"
xcopy "%s" "%s" /E /I /Y >nul
setx /M PATH "%s;%s;%%PATH%%" >nul
`,
			targetDir, targetDir,
			tmpExtract, targetDir,
			targetDir, scriptsDir)

		batPath := filepath.Join(userTempDir(), "wintools-install-python.bat")
		if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
			return ProgressInfo{Step: "error", Error: "无法创建安装脚本: " + err.Error()}
		}
		if err := common.RunElevatedAndWait("cmd.exe", []string{"/c", batPath}); err != nil {
			return ProgressInfo{Step: "error", Error: "安装失败: " + err.Error()}
		}
		os.Remove(batPath)
	}

	// ── 6. Verify ──
	a.emit("verify", "正在验证安装...", 90, false, "")
	st := a.CheckStatus()
	if !st.Installed {
		return ProgressInfo{Step: "error", Error: "安装后未找到 Python，请检查 " + targetDir}
	}

	// ── 7. Clean up ──
	os.RemoveAll(tmpExtract)
	os.Remove(zipPath)

	a.emit("done", "Python "+fullVersion+" 安装成功！", 100, true, "")
	return ProgressInfo{Step: "done", Message: "Python 安装成功！", Percent: 100, Done: true}
}

// isPipPackageInstalled checks if a pip package is already installed.
func isPipPackageInstalled(pythonExe, pkg string) bool {
	cmd := exec.Command(pythonExe, "-m", "pip", "show", pkg)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

// InstallPackages installs the selected Python packages via pip (elevated).
func (a *InstallerAPI) InstallPackages(packages []string) ProgressInfo {
	st := a.CheckStatus()
	if !st.Installed {
		return ProgressInfo{Step: "error", Error: "请先安装 Python 环境"}
	}

	pythonExe := st.PythonExe
	total := len(packages)
	succeeded := 0

	for i, pkg := range packages {
		percent := float64(i) / float64(total) * 90
		msg := fmt.Sprintf("正在检查 %s (%d/%d)...", pkg, i+1, total)
		a.emit("install-package", msg, percent, false, "")

		// Skip if already installed
		if isPipPackageInstalled(pythonExe, pkg) {
			a.emit("install-package", pkg+" 已安装，跳过", percent, false, "")
			fmt.Printf("[pyenv] %s 已存在，跳过\n", pkg)
			succeeded++
			continue
		}

		msg = fmt.Sprintf("正在安装 %s (%d/%d)...", pkg, i+1, total)

		fmt.Printf("[pyenv] pip install %s\n", pkg)
		err := runWithPipMirrors(func(index string) error {
			a.emit("install-package", fmt.Sprintf("正在通过 %s 安装 %s...", index, pkg), percent, false, "")
			args := []string{
				"-m", "pip", "install", "--no-warn-script-location",
				"--disable-pip-version-check", "--timeout", "30", "--retries", "2",
				"--index-url", index, "--trusted-host", extractHost(index), pkg,
			}
			if common.IsAdmin() {
				cmd := exec.Command(pythonExe, args...)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				return cmd.Run()
			}
			return common.RunElevatedAndWait(pythonExe, args)
		})
		if err != nil {
			a.emit("install-package", pkg+" 安装失败", percent, false, err.Error())
			fmt.Printf("[pyenv] %s 失败: %v\n", pkg, err)
		} else {
			succeeded++
			fmt.Printf("[pyenv] %s 成功\n", pkg)
		}
	}

	if succeeded == total {
		a.emit("done", fmt.Sprintf("全部 %d 个包安装成功！", total), 100, true, "")
		return ProgressInfo{Step: "done", Message: "安装完成！", Percent: 100, Done: true}
	}
	msg := fmt.Sprintf("%d/%d 个包安装成功", succeeded, total)
	a.emit("done", msg, 100, true, "")
	return ProgressInfo{Step: "done", Message: msg, Percent: 100, Done: true}
}

// InstallWithElevation full install: Python + packages (kept for backward compat).
func (a *InstallerAPI) InstallWithElevation(packages []string) ProgressInfo {
	result := a.InstallPython()
	if result.Error != "" {
		return result
	}
	if len(packages) > 0 {
		return a.InstallPackages(packages)
	}
	return result
}

// ── Legacy helpers (kept for tests) ──────────────────────

func RunInstallWorker(req InstallRequest) TaskState {
	return runInstallWorker(req, defaultDeps())
}

func runInstallWorker(req InstallRequest, deps installDependencies) TaskState {
	step := func(s string) { deps.writeState(req.StatePath, TaskState{Step: s, Running: true}) }
	finish := func(s TaskState) TaskState {
		deps.writeState(req.StatePath, s)
		return s
	}

	state := TaskState{Step: "prepare", Message: "开始安装 Python " + req.PythonVersion, Percent: 0, Running: true}
	deps.writeState(req.StatePath, state)

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

	if out, err := deps.runCmd(installerPath, spec.Args...); err != nil {
		return finish(TaskState{Step: "error", Error: fmt.Sprintf("Python 安装程序退出错误: %v\n输出: %s", err, out)})
	}

	step("verify-python")
	deps.writeState(req.StatePath, TaskState{
		Step: "verify-python", Message: "正在验证 Python 安装...", Percent: 60, Running: true,
	})

	pythonExe, err := deps.findPython(req.PythonVersion)
	if err != nil {
		return finish(TaskState{Step: "error", Error: "安装后未找到 Python: " + err.Error()})
	}

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

		var out string
		err := runWithPipMirrors(func(index string) error {
			var runErr error
			out, runErr = deps.runCmd(pythonExe, "-m", "pip", "install", "--no-warn-script-location",
				"--disable-pip-version-check", "--timeout", "30", "--retries", "2", "--index-url", index, pkg)
			return runErr
		})
		if err != nil {
			state.Packages[i].Status = "failed"
			state.Packages[i].Error = err.Error()
			_ = out
			allSucceeded = false
		} else {
			state.Packages[i].Status = "success"
		}
		deps.writeState(req.StatePath, state)
	}

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
