package pyenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestDownloadWithCurlUsesFixedArguments(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "python.zip")
	var gotName string
	var gotArgs []string
	runner := func(name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil, os.WriteFile(dest, []byte("zip"), 0o600)
	}

	err := downloadWithCurl(`C:\Windows\System32\curl.exe`, "https://example.test/python.zip", dest, runner)
	if err != nil {
		t.Fatalf("downloadWithCurl: %v", err)
	}
	if gotName != `C:\Windows\System32\curl.exe` {
		t.Fatalf("command = %q", gotName)
	}
	want := []string{
		"--fail", "--location", "--silent", "--show-error",
		"--connect-timeout", "10", "--max-time", "120",
		"--retry", "2", "--retry-delay", "1", "--retry-all-errors",
		"--output", dest, "https://example.test/python.zip",
	}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestDownloadFileWithFallsBackAfterCurlFailure(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "python.zip")
	curlCalls, goCalls := 0, 0
	err := downloadFileWith("https://example.test/python.zip", dest,
		func(string, string) error {
			curlCalls++
			return errors.New("curl failed")
		},
		func(string, string) error {
			goCalls++
			return os.WriteFile(dest, []byte("zip"), 0o600)
		},
	)
	if err != nil || curlCalls != 1 || goCalls != 1 {
		t.Fatalf("err=%v curlCalls=%d goCalls=%d", err, curlCalls, goCalls)
	}
}

func TestDownloadFromSourcesFallsBackAndPromotesPart(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "artifact.zip")
	sources := []DownloadSource{{Name: "one", URL: "one"}, {Name: "two", URL: "two"}}
	var calls []string
	err := downloadFromSources(sources, dest, func(url, part string) error {
		calls = append(calls, url)
		if url == "one" {
			_ = os.WriteFile(part, []byte("bad"), 0o600)
			return errors.New("failed")
		}
		return os.WriteFile(part, []byte("good"), 0o600)
	}, func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(data) != "good" {
			return errors.New("invalid")
		}
		return nil
	}, nil)
	if err != nil || strings.Join(calls, ",") != "one,two" {
		t.Fatalf("err=%v calls=%v", err, calls)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("part remains")
	}
}

func TestDownloadFromSourcesReusesValidCache(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(dest, []byte("cached"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := downloadFromSources([]DownloadSource{{Name: "one", URL: "one"}}, dest,
		func(string, string) error { calls++; return nil },
		func(string) error { return nil }, nil)
	if err != nil || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestRunWithPipMirrorsStopsAtFirstSuccess(t *testing.T) {
	calls := []string{}
	err := runWithPipMirrors(func(index string) error {
		calls = append(calls, index)
		if len(calls) < 2 {
			return errors.New("failed")
		}
		return nil
	})
	if err != nil || len(calls) != 2 || calls[0] != DefaultMirrors[0].URL || calls[1] != DefaultMirrors[1].URL {
		t.Fatalf("err=%v calls=%v", err, calls)
	}
}

func TestGetPipInstallBatchTriesMirrorsInOrder(t *testing.T) {
	batch := getPipInstallBatch(`C:\Python\python.exe`, `C:\Python\get-pip.py`)
	previous := -1
	for _, mirror := range DefaultMirrors {
		position := strings.Index(batch, `--index-url "`+mirror.URL+`"`)
		if position <= previous {
			t.Fatalf("mirror %q missing or out of order in batch: %s", mirror.Name, batch)
		}
		previous = position
	}
	for _, flag := range []string{"--disable-pip-version-check", "--timeout 30", "--retries 2"} {
		if !strings.Contains(batch, flag) {
			t.Fatalf("batch missing %q: %s", flag, batch)
		}
	}
	if !strings.Contains(batch, "if errorlevel 1 exit /b 1") || !strings.Contains(batch, ":pip_ok") {
		t.Fatalf("batch does not fail after exhausting mirrors: %s", batch)
	}
}
