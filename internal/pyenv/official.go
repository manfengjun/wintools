package pyenv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ── Installer spec ────────────────────────────────────────

// InstallerSpec describes the download URL and silent arguments
// for the official Python Windows installer.
type InstallerSpec struct {
	URL  string
	Args []string
}

// officialInstallerSpec returns the download URL and silent arguments
// for the given Python version (e.g. "3.12.10" or "3.12" → "3.12.0").
func officialInstallerSpec(version string) InstallerSpec {
	fullVersion := version
	if strings.Count(version, ".") == 1 {
		fullVersion = version + ".0"
	}
	url := fmt.Sprintf("https://www.python.org/ftp/python/%s/python-%s-amd64.exe",
		fullVersion, fullVersion)
	args := []string{
		"/quiet",
		"InstallAllUsers=1",
		"PrependPath=1",
		"Include_pip=1",
		"Include_launcher=1",
	}
	return InstallerSpec{URL: url, Args: args}
}

// ── Python discovery ──────────────────────────────────────

// findInstalledPython searches common locations for a Python installation
// matching the requested version.
func findInstalledPython(version string) (string, error) {
	roots := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		"C:\\",
	}
	return findPythonInRoots(version, roots)
}

// findPythonInRoots searches for python.exe under ProgramFiles-style roots
// using the pattern <root>/Python<major><minor>/python.exe.
func findPythonInRoots(version string, roots []string) (string, error) {
	// Normalise "3.12" or "3.12.10" → "312"
	tag := strings.ReplaceAll(version, ".", "")
	if len(tag) > 3 {
		tag = tag[:3]
	}

	for _, root := range roots {
		if root == "" {
			continue
		}
		candidate := filepath.Join(root, "Python"+tag, "python.exe")
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Python %s not found in any standard location", version)
}

// ── Command execution ─────────────────────────────────────

// runCommand executes the given executable with arguments and returns
// combined stdout+stderr.
func runCommand(exe string, args ...string) (string, error) {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
