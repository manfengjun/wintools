package common

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

// Windows ShellExecuteExW constants
const (
	seeMaskNoCloseProcess = 0x00000040
	swShownormal          = 1
	swHide                = 0
)

type shellExecuteInfoW struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hMonitor     uintptr
	hProcess     uintptr
}

// IsAdmin checks if the current process is running as administrator
// by attempting to open the physical drive.
func IsAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// RunElevated starts a process with administrator rights via ShellExecuteW "runas".
// Non-blocking — returns immediately after launching.
func RunElevated(exePath string, args []string) (int, error) {
	verb, _ := syscall.UTF16PtrFromString("runas")
	exe, _ := syscall.UTF16PtrFromString(exePath)

	argStr := strings.Join(args, " ")
	argv, _ := syscall.UTF16PtrFromString(argStr)

	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW := shell32.NewProc("ShellExecuteW")

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exe)),
		uintptr(unsafe.Pointer(argv)),
		0,
		swShownormal,
	)

	if int(ret) <= 32 {
		return 0, fmt.Errorf("ShellExecuteW failed with code %d", int(ret))
	}
	return 0, nil
}

// RunElevatedAndWait starts a process with administrator rights via ShellExecuteExW "runas",
// waits for it to finish, and returns an error if the exit code is non‑zero.
// It is the preferred blocking-elevation helper — no PowerShell nesting, no encoding issues.
func RunElevatedAndWait(exePath string, args []string) error {
	verb, _ := syscall.UTF16PtrFromString("runas")
	exe, _ := syscall.UTF16PtrFromString(exePath)

	// Quote each argument to handle spaces
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = `"` + a + `"`
	}
	argStr := strings.Join(quoted, " ")
	argv, _ := syscall.UTF16PtrFromString(argStr)

	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecuteExW := shell32.NewProc("ShellExecuteExW")

	info := shellExecuteInfoW{
		cbSize:       uint32(unsafe.Sizeof(shellExecuteInfoW{})),
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verb,
		lpFile:       exe,
		lpParameters: argv,
		nShow:        swHide,
	}

	ret, _, _ := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return fmt.Errorf("提权启动失败")
	}

	// Wait for the process to finish
	if info.hProcess != 0 {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		procWait := kernel32.NewProc("WaitForSingleObject")
		const infinite = 0xFFFFFFFF
		procWait.Call(info.hProcess, infinite)

		var exitCode uint32
		procGetExitCode := kernel32.NewProc("GetExitCodeProcess")
		r, _, _ := procGetExitCode.Call(info.hProcess, uintptr(unsafe.Pointer(&exitCode)))

		procClose := kernel32.NewProc("CloseHandle")
		procClose.Call(info.hProcess)

		if r != 0 && exitCode != 0 && exitCode != 3010 {
			return fmt.Errorf("进程退出码: %d", exitCode)
		}
	}

	return nil
}

// RunCommand runs an executable and returns combined stdout+stderr.
// The child window is hidden so no console flashes.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
