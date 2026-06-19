package common

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	MirrorURL    string `json:"mirror_url"`
	PyVersion    string `json:"py_version"`
	PyInstallDir string `json:"py_install_dir"`
	UpdateURL    string `json:"update_url"`
}

var DefaultConfig = Config{
	MirrorURL:    "https://pypi.tuna.tsinghua.edu.cn/simple",
	PyVersion:    "3.12.0",
	PyInstallDir: "C:\\Python\\3.12",
	UpdateURL:    "https://gitee.com/api/v5/repos/manfengjun/wintools/releases/latest",
}

func ConfigDir() string {
	d := os.Getenv("APPDATA")
	if d == "" {
		d = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(d, "DesktopSuite")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func LoadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		c := DefaultConfig
		return &c
	}
	json.Unmarshal(data, cfg)
	return cfg
}

func SaveConfig(cfg *Config) error {
	os.MkdirAll(ConfigDir(), 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(ConfigPath(), data, 0644)
}
