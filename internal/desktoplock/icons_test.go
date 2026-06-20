package desktoplock

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetFileIconReturnsPNG(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows icon extraction test")
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	data, err := getFileIcon(executable)
	if err != nil {
		t.Fatalf("getFileIcon(%q): %v", executable, err)
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("getFileIcon did not return PNG data")
	}

	assertVisiblePNG(t, data)
}

func TestBackupShortcutsYieldVisiblePNG(t *testing.T) {
	dir := os.Getenv("WINTOOLS_BACKUP_ICON_TEST_DIR")
	if dir == "" {
		t.Skip("set WINTOOLS_BACKUP_ICON_TEST_DIR to run against real backup shortcuts")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if entry.IsDir() || (ext != ".lnk" && ext != ".url") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			data, err := getFileIcon(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			assertVisiblePNG(t, data)
		})
	}
}

func assertVisiblePNG(t *testing.T, data []byte) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode extracted icon: %v", err)
	}
	visible := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y && !visible; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha != 0 {
				visible = true
				break
			}
		}
	}
	if !visible {
		t.Fatal("extracted icon PNG is fully transparent")
	}
}
