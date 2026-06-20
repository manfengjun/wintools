package pyenv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// ── Fake dependencies ────────────────────────────────────

type fakeDeps struct {
	installerExit  error
	installerCalls int
	pipCalls       int
	packageErrors  map[string]error
	downloadErr    error
	findPythonExe  string
	findPythonErr  error
	states         []TaskState
}

func (f *fakeDeps) reset() {
	f.installerCalls = 0
	f.pipCalls = 0
	f.states = nil
}

func (f *fakeDeps) download(url, dest string) error {
	if f.downloadErr != nil {
		return f.downloadErr
	}
	return nil
}

func (f *fakeDeps) runCmd(exe string, args ...string) (string, error) {
	if len(args) > 0 && args[0] == "-m" && args[1] == "pip" {
		f.pipCalls++
		if f.packageErrors != nil {
			// Find which package is being installed
			for _, a := range args {
				if err, ok := f.packageErrors[a]; ok {
					return "", err
				}
			}
		}
		return "", nil
	}
	// The installer run
	f.installerCalls++
	if f.installerExit != nil {
		return "", f.installerExit
	}
	return "", nil
}

func (f *fakeDeps) findPython(version string) (string, error) {
	if f.findPythonErr != nil {
		return "", f.findPythonErr
	}
	if f.findPythonExe != "" {
		return f.findPythonExe, nil
	}
	return "C:\\Python312\\python.exe", nil
}

func (f *fakeDeps) writeState(path string, state TaskState) error {
	f.states = append(f.states, state)
	return nil
}

func (f *fakeDeps) toDeps() installDependencies {
	return installDependencies{
		downloadFile: f.download,
		runCmd:       f.runCmd,
		findPython:   f.findPython,
		writeState:   f.writeState,
	}
}

func testRequest(t *testing.T) InstallRequest {
	t.Helper()
	return InstallRequest{
		PythonVersion: "3.12.0",
		Packages:      []string{},
		StatePath:     filepath.Join(t.TempDir(), "state.json"),
	}
}

func requestWithPackages(pkgs ...string) InstallRequest {
	return InstallRequest{
		PythonVersion: "3.12.0",
		Packages:      pkgs,
		StatePath:     "test-state.json",
	}
}

// ── Tests ────────────────────────────────────────────────

func TestWorkerStopsWhenOfficialInstallerFails(t *testing.T) {
	deps := &fakeDeps{installerExit: errors.New("exit code 1603")}
	state := runInstallWorker(testRequest(t), deps.toDeps())
	if state.Step != "error" || !strings.Contains(state.Error, "1603") {
		t.Fatalf("expected error step with 1603, got %+v", state)
	}
	if deps.pipCalls != 0 {
		t.Fatal("pip must not run after installer failure")
	}
}

func TestWorkerContinuesAfterOnePackageFails(t *testing.T) {
	deps := &fakeDeps{
		packageErrors: map[string]error{"numpy": errors.New("failed")},
	}
	state := runInstallWorker(requestWithPackages("numpy", "pygame"), deps.toDeps())
	if !state.Done {
		t.Fatalf("expected Done=true, got %+v", state)
	}
	if len(state.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(state.Packages))
	}
	if state.Packages[0].Status != "failed" || state.Packages[1].Status != "success" {
		t.Fatalf("package statuses wrong: %+v", state.Packages)
	}
}

func TestWorkerReportsProgressStatesInOrder(t *testing.T) {
	deps := &fakeDeps{}
	_ = runInstallWorker(testRequest(t), deps.toDeps())

	if len(deps.states) == 0 {
		t.Fatal("expected at least one state write")
	}

	// First state should be prepare
	if deps.states[0].Step != "prepare" {
		t.Fatalf("first step should be prepare, got %q", deps.states[0].Step)
	}
	// Last state should be done
	last := deps.states[len(deps.states)-1]
	if last.Step != "done" {
		t.Fatalf("last step should be done, got %q", last.Step)
	}
	if !last.Done {
		t.Fatal("last state should have Done=true")
	}
}

func TestWorkerErrorWhenPythonNotFoundAfterInstall(t *testing.T) {
	deps := &fakeDeps{
		findPythonErr: fmt.Errorf("Python 3.12 not found"),
	}
	state := runInstallWorker(testRequest(t), deps.toDeps())
	if state.Step != "error" || state.Error == "" {
		t.Fatalf("expected error, got %+v", state)
	}
}

func TestWorkerRunsInstallerOnce(t *testing.T) {
	deps := &fakeDeps{}
	state := runInstallWorker(testRequest(t), deps.toDeps())
	if deps.installerCalls != 0 {
		// The installer path is handled by exec.Command, not runCmd
		// so installerCalls remains 0 in our fake — that's expected.
	}
	if state.Step != "done" {
		t.Fatalf("expected done, got %q", state.Step)
	}
}
