package desktoplock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildBackupItemsIncludesIcon(t *testing.T) {
	dir := t.TempDir()
	name := "Example.lnk"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("shortcut"), 0o644); err != nil {
		t.Fatal(err)
	}

	items := buildBackupItems(dir, []string{name}, func(path string) string {
		if path != filepath.Join(dir, name) {
			t.Fatalf("icon reader received %q", path)
		}
		return "data:image/png;base64,AAAA"
	})

	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].IconBase64 != "data:image/png;base64,AAAA" {
		t.Fatalf("IconBase64 = %q", items[0].IconBase64)
	}
}

func TestBuildBackupItemsKeepsItemWhenIconFails(t *testing.T) {
	dir := t.TempDir()
	name := "Example.url"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("shortcut"), 0o644); err != nil {
		t.Fatal(err)
	}

	items := buildBackupItems(dir, []string{name}, func(string) string { return "" })

	if len(items) != 1 || items[0].Name != name || items[0].IconBase64 != "" {
		t.Fatalf("unexpected item: %#v", items)
	}
}
