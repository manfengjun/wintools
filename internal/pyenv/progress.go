package pyenv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ── Types ──────────────────────────────────────────────────

// InstallRequest describes a Python installation request
// passed from the normal process to the elevated worker.
type InstallRequest struct {
	PythonVersion string   `json:"python_version"`
	Packages      []string `json:"packages"`
	StatePath     string   `json:"state_path"`
}

// TaskState represents the current progress of a Python installation task.
type TaskState struct {
	Step      string         `json:"step"`
	Message   string         `json:"message"`
	Percent   float64        `json:"percent"`
	Running   bool           `json:"running"`
	Done      bool           `json:"done"`
	Error     string         `json:"error,omitempty"`
	PythonExe string         `json:"python_exe,omitempty"`
	Packages  []PackageState `json:"packages,omitempty"`
}

// PackageState describes the installation status of a single Python package.
type PackageState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ── Atomic file-state storage ─────────────────────────────

// writeTaskState atomically writes task state to the given path.
// It marshals the state to JSON, writes to a temporary file,
// then renames to the destination to ensure atomicity.
func writeTaskState(path string, state TaskState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal task state: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state dir %s: %w", dir, err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp state %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Best-effort cleanup of orphaned temp file
		os.Remove(tmpPath)
		return fmt.Errorf("rename state %s -> %s: %w", tmpPath, path, err)
	}

	return nil
}

// readTaskState reads and unmarshals task state from the given path.
// Returns a typed error when the file is missing or malformed.
func readTaskState(path string) (TaskState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TaskState{}, fmt.Errorf("read state %s: %w", path, err)
	}

	var state TaskState
	if err := json.Unmarshal(data, &state); err != nil {
		return TaskState{}, fmt.Errorf("parse state %s: %w", path, err)
	}

	return state, nil
}
