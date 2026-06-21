package updater

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckReleaseFindsNewVersionAndSetupAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name":"v1.0.3",
			"body":"修复更新检测",
			"assets":[
				{"name":"Wintools_Windows_x86_64.exe","browser_download_url":"https://example.test/setup.exe"}
			]
		}`))
	}))
	defer server.Close()

	info := checkRelease(server.URL, "1.0.2")
	if info.Error != "" {
		t.Fatalf("unexpected error: %s", info.Error)
	}
	if !info.HasUpdate || info.Version != "1.0.3" {
		t.Fatalf("expected update 1.0.3, got %+v", info)
	}
	if info.DownloadURL != "https://example.test/setup.exe" {
		t.Fatalf("expected setup asset, got %q", info.DownloadURL)
	}
	if info.ReleaseNotes != "修复更新检测" {
		t.Fatalf("unexpected release notes: %q", info.ReleaseNotes)
	}
}

func TestCheckReleaseReportsCurrentVersionAsLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.2","assets":[]}`))
	}))
	defer server.Close()

	info := checkRelease(server.URL, "1.0.2")
	if info.Error != "" || info.HasUpdate || info.Version != "1.0.2" {
		t.Fatalf("expected current version, got %+v", info)
	}
}

func TestBuildInstallerBatchRunsSetupInsteadOfReplacingExecutable(t *testing.T) {
	batch := buildInstallerBatch(`C:\Temp\Wintools Setup.exe`)
	if strings.Contains(strings.ToLower(batch), "copy /y") {
		t.Fatalf("installer update must not replace the application with the setup executable: %s", batch)
	}
	if !strings.Contains(batch, `start "" /wait "C:\Temp\Wintools Setup.exe" /S`) {
		t.Fatalf("expected silent NSIS installer launch, got: %s", batch)
	}
}

func TestParseWindowsProxy(t *testing.T) {
	proxyURL := parseWindowsProxy("127.0.0.1:7890")
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected proxy URL: %v", proxyURL)
	}

	proxyURL = parseWindowsProxy("http=127.0.0.1:8080;https=127.0.0.1:8443")
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:8443" {
		t.Fatalf("expected HTTPS proxy entry, got: %v", proxyURL)
	}
}
