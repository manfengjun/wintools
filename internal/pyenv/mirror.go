package pyenv

// Mirror defines a Python package mirror source.
type Mirror struct {
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
	{Name: "华为云", Base: "https://repo.huaweicloud.com/python"},
	{Name: "官方 python.org", Base: "https://www.python.org/ftp/python"},
}

// PythonDownloadURL returns the download URL for a given Python version using the first available mirror.
func PythonDownloadURL(version string) string {
	return "https://www.python.org/ftp/python/" + version + "/python-" + version + "-embed-amd64.zip"
}

// GetPipURL returns the get-pip.py download URL.
const GetPipURL = "https://bootstrap.pypa.io/get-pip.py"
