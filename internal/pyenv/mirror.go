package pyenv

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Mirror defines a Python package mirror source.
type Mirror struct {
	Name string
	URL  string
}

// DownloadSource identifies one candidate URL for an installation artifact.
type DownloadSource struct {
	Name string
	URL  string
}

// DefaultMirrors lists domestic mirrors in priority order.
var DefaultMirrors = []Mirror{
	{Name: "清华 Tuna", URL: "https://pypi.tuna.tsinghua.edu.cn/simple"},
	{Name: "阿里云", URL: "https://mirrors.aliyun.com/pypi/simple"},
	{Name: "官方 PyPI", URL: "https://pypi.org/simple"},
}

// PythonDownloadMirrors lists mirrors for downloading Python embeddable releases.
var PythonDownloadMirrors = []struct {
	Name string
	Base string
}{
	{Name: "清华 Tuna", Base: "https://mirrors.tuna.tsinghua.edu.cn/python"},
	{Name: "中科大 USTC", Base: "https://mirrors.ustc.edu.cn/python"},
	{Name: "华为云", Base: "https://repo.huaweicloud.com/python"},
	{Name: "官方 python.org", Base: "https://www.python.org/ftp/python"},
}

// PythonDownloadSources returns every Python artifact source in failover order.
func PythonDownloadSources(version string) []DownloadSource {
	path := version + "/python-" + version + "-embed-amd64.zip"
	out := make([]DownloadSource, 0, len(PythonDownloadMirrors))
	for _, mirror := range PythonDownloadMirrors {
		out = append(out, DownloadSource{Name: mirror.Name, URL: mirror.Base + "/" + path})
	}
	return out
}

// PythonDownloadURL returns the download URL for a given Python version using mirrors.
// The first available mirror in PythonDownloadMirrors has the highest priority.
func PythonDownloadURL(version string) string {
	sources := PythonDownloadSources(version)
	if len(sources) > 0 {
		return sources[0].URL
	}
	return ""
}

// GetPipSources returns get-pip.py sources in failover order.
func GetPipSources() []DownloadSource {
	return []DownloadSource{
		{Name: "阿里云", URL: "https://mirrors.aliyun.com/pypi/get-pip.py"},
		{Name: "官方 PyPA", URL: "https://bootstrap.pypa.io/get-pip.py"},
	}
}

// PipWhlSources returns pip wheel sources (direct install, no SSL needed).
func PipWhlSources() []DownloadSource {
	return []DownloadSource{
		{Name: "清华 Tuna", URL: "https://pypi.tuna.tsinghua.edu.cn/simple/pip/"},
		{Name: "阿里云", URL: "https://mirrors.aliyun.com/pypi/simple/pip/"},
		{Name: "官方 PyPI", URL: "https://pypi.org/simple/pip/"},
	}
}

// ResolvePipWhlURL fetches a PyPI simple index page and extracts the latest
// wheel URL (py3-none-any.whl) from the HTML. Returns the full download URL.
func ResolvePipWhlURL(indexURL string) (string, error) {
	req, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return "", err
	}
	// Some mirrors require a proper User-Agent
	req.Header.Set("User-Agent", "Wintools/1.0")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	html := string(body)
	// Find the last whl link (newest version typically appears last)
	re := regexp.MustCompile(`href="([^"]*pip-[\d.]+-py3-none-any\.whl[^"]*)"`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("在 %s 中未找到 pip whl 链接", indexURL)
	}
	last := matches[len(matches)-1][1]
	// Handle relative URLs like "../../packages/ab/cd/.../pip.whl"
	if strings.HasPrefix(last, "../../") {
		// indexURL is like "https://pypi.tuna.tsinghua.edu.cn/simple/pip/"
		// We need to resolve "../../" relative to it:
		// /simple/pip/ → go up 2 levels → /
		// Then append "packages/.../pip.whl"
		parts := strings.Split(strings.TrimRight(indexURL, "/"), "/")
		if len(parts) >= 3 {
			// Keep scheme + host, drop the last 2 path components
			base := strings.Join(parts[:len(parts)-2], "/")
			rel := strings.TrimPrefix(last, "../../")
			return base + "/" + rel, nil
		}
	}
	if strings.HasPrefix(last, "http") {
		return last, nil
	}
	return last, nil
}

// GetPipURL is kept for legacy callers; new installation code uses GetPipSources.
const GetPipURL = "https://bootstrap.pypa.io/get-pip.py"
