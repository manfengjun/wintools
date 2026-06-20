package pyenv

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOfficialInstallerSpec(t *testing.T) {
	spec := officialInstallerSpec("3.12.10")
	if spec.URL != "https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe" {
		t.Fatalf("unexpected URL: %s", spec.URL)
	}
	want := []string{"/quiet", "InstallAllUsers=1", "PrependPath=1", "Include_pip=1", "Include_launcher=1"}
	for _, flag := range want {
		if !slices.Contains(spec.Args, flag) {
			t.Fatalf("missing silent argument %q in %v", flag, spec.Args)
		}
	}
}

func TestFindInstalledPythonPrefersRequestedVersion(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "Python312", "python.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, nil, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := findPythonInRoots("3.12", []string{root})
	if err != nil {
		t.Fatalf("findPythonInRoots failed: %v", err)
	}
	if got != exe {
		t.Fatalf("got %q, want %q", got, exe)
	}
}

func TestFindInstalledPythonSkipsMissingVersion(t *testing.T) {
	root := t.TempDir()
	// Create Python311 but request 3.12
	exe := filepath.Join(root, "Python311", "python.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, nil, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := findPythonInRoots("3.12", []string{root})
	if err == nil {
		t.Fatal("expected error for missing Python version")
	}
}

func TestOfficialInstallerSpecVersionFormat(t *testing.T) {
	spec := officialInstallerSpec("3.12")
	if spec.URL != "https://www.python.org/ftp/python/3.12.0/python-3.12.0-amd64.exe" {
		t.Fatalf("unexpected URL for short version: %s", spec.URL)
	}
}
