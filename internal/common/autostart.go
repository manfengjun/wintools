package common

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const autoStartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autoStartValueName = "Wintools"

// IsAutoStart 检查是否已注册开机自启动。
func IsAutoStart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue(autoStartValueName)
	if err != nil || val == "" {
		return false
	}

	currentExe, err := os.Executable()
	if err != nil {
		return false
	}
	// 确认指向的是当前程序（避免手动修改或残留引用）
	return filepath.Clean(val) == filepath.Clean(currentExe)
}

// EnableAutoStart 注册开机自启动。
func EnableAutoStart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(autoStartValueName, exe)
}

// DisableAutoStart 取消开机自启动注册。
func DisableAutoStart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.DeleteValue(autoStartValueName); err != nil {
		if err == registry.ErrNotExist {
			return nil // 未注册也算是"取消成功"
		}
		return err
	}
	return nil
}
