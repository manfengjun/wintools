package pyenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTaskStateJSONProtocol(t *testing.T) {
	request := InstallRequest{
		PythonVersion: "3.12.9",
		Packages:      []string{"numpy", "pygame"},
		StatePath:     `C:\\temp\\python-state.json`,
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal install request: %v", err)
	}

	jsonText := string(data)
	for _, field := range []string{"python_version", "packages", "state_path"} {
		if !strings.Contains(jsonText, `"`+field+`"`) {
			t.Errorf("request JSON %s does not contain snake_case field %q", jsonText, field)
		}
	}
}

func TestTaskStateRoundTripCreatesParentAndRemovesTemp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	want := TaskState{
		Step:      "packages",
		Message:   "Installing packages",
		Percent:   72.5,
		Running:   true,
		PythonExe: `C:\\Python\\python.exe`,
		Packages: []PackageState{
			{Name: "numpy", Status: "done"},
			{Name: "pygame", Status: "failed", Error: "network error"},
		},
	}

	if err := writeTaskState(path, want); err != nil {
		t.Fatalf("write state: %v", err)
	}
	got, err := readTaskState(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains after write: %v", err)
	}
}

func TestTaskStateRejectsMissingAndCorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if _, err := readTaskState(path); err == nil {
		t.Fatal("read missing state returned nil error")
	}

	if err := os.WriteFile(path, []byte(`{"step":`), 0o600); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}
	if _, err := readTaskState(path); err == nil {
		t.Fatal("read corrupt state returned nil error")
	}
}

func TestTaskStateConsecutiveWritesReplaceExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	first := TaskState{Step: "download", Running: true, Percent: 10}
	last := TaskState{Step: "complete", Message: "Done", Percent: 100, Done: true}

	if err := writeTaskState(path, first); err != nil {
		t.Fatalf("write first state: %v", err)
	}
	if err := writeTaskState(path, last); err != nil {
		t.Fatalf("replace state: %v", err)
	}
	got, err := readTaskState(path)
	if err != nil {
		t.Fatalf("read replaced state: %v", err)
	}
	if !reflect.DeepEqual(got, last) {
		t.Fatalf("replaced state mismatch: got %#v, want %#v", got, last)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains after replacement: %v", err)
	}
}
