package pyenv

import (
	"strings"
	"testing"
)

func sourceNames(sources []DownloadSource) string {
	names := make([]string, len(sources))
	for i, source := range sources {
		names[i] = source.Name
	}
	return strings.Join(names, ",")
}

func TestPythonDownloadSourcesOrder(t *testing.T) {
	got := sourceNames(PythonDownloadSources("3.12.0"))
	want := "清华 Tuna,中科大 USTC,华为云,官方 python.org"
	if got != want {
		t.Fatalf("order=%q, want %q", got, want)
	}
}

func TestGetPipSourcesOrder(t *testing.T) {
	got := sourceNames(GetPipSources())
	want := "阿里云,官方 PyPA"
	if got != want {
		t.Fatalf("order=%q, want %q", got, want)
	}
}
