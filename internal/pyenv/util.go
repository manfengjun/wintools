package pyenv

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// extractZip extracts a zip file to the target directory.
// If target exists, it's removed first for a clean install.
func extractZip(zipPath, target string) error {
	if _, err := os.Stat(target); err == nil {
		os.RemoveAll(target)
	}
	os.MkdirAll(target, 0755)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(target, f.Name)
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(fpath)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// configurePth modifies python._pth to enable site-packages and add DLLs/Lib paths.
func configurePth(targetDir string) error {
	pthFiles, err := filepath.Glob(filepath.Join(targetDir, "*._pth"))
	if err != nil {
		return err
	}

	var pthPath string
	if len(pthFiles) > 0 {
		pthPath = pthFiles[0]
	} else {
		pthPath = filepath.Join(targetDir, "python._pth")
	}

	var lines []string
	if data, err := os.ReadFile(pthPath); err == nil {
		lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	} else {
		lines = []string{
			"python3.zip",
			".",
			"DLLs",
			"Lib",
			"Lib\\site-packages",
			"import site",
		}
	}

	hasDLLs := false
	hasLib := false
	importEnabled := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "DLLs") {
			hasDLLs = true
		}
		if strings.EqualFold(trimmed, "Lib") {
			hasLib = true
		}
		if trimmed == "import site" {
			importEnabled = true
		}
	}

	// Rebuild with proper structure
	var newLines []string
	added := map[string]bool{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		newLines = append(newLines, trimmed)
		added[trimmed] = true
		// After ".", insert DLLs and Lib if missing
		if trimmed == "." {
			if !hasDLLs && !added["DLLs"] {
				newLines = append(newLines, "DLLs")
				added["DLLs"] = true
			}
			if !hasLib && !added["Lib"] {
				newLines = append(newLines, "Lib")
				added["Lib"] = true
			}
		}
	}
	if !importEnabled {
		newLines = append(newLines, "import site")
	}
	content := strings.Join(newLines, "\r\n") + "\r\n"
	return os.WriteFile(pthPath, []byte(content), 0644)
}

// installPip downloads and installs pip for the target Python.
func installPip(pythonExe, mirror string) error {
	getPipPath := filepath.Join(filepath.Dir(pythonExe), "get-pip.py")
	if err := downloadFile(GetPipURL, getPipPath); err != nil {
		return fmt.Errorf("下载 get-pip.py 失败: %w", err)
	}
	args := []string{getPipPath, "--no-warn-script-location"}
	if mirror != "" {
		args = append(args, "-i", mirror, "--trusted-host", extractHost(mirror))
	}
	cmd := exec.Command(pythonExe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// installPackages installs the given Python packages via pip from mirror.
func installPackages(pythonExe, mirror string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}
	args := []string{"-m", "pip", "install"}
	if mirror != "" {
		args = append(args, "-i", mirror, "--trusted-host", extractHost(mirror))
	}
	args = append(args, packages...)
	cmd := exec.Command(pythonExe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// addToSystemPath adds Python directories to system PATH via setx.
func addToSystemPath(installDir string) error {
	paths := []string{installDir, filepath.Join(installDir, "Scripts")}
	for _, p := range paths {
		exec.Command("setx", "/M", "PATH", p+";%PATH%").Run()
	}
	return nil
}

func extractHost(mirrorURL string) string {
	mirrorURL = strings.TrimPrefix(mirrorURL, "https://")
	mirrorURL = strings.TrimPrefix(mirrorURL, "http://")
	parts := strings.Split(mirrorURL, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return mirrorURL
}
