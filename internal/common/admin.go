package common

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

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

// RunElevated starts a subprocess with administrator rights via ShellExecute "runas".
// It waits for the process to finish and returns its exit code.
func RunElevated(exePath string, args []string) (int, error) {
	verb, _ := syscall.UTF16PtrFromString("runas")
	exe, _ := syscall.UTF16PtrFromString(exePath)

	// Build argument string
	argStr := ""
	for i, a := range args {
		if i > 0 {
			argStr += " "
		}
		argStr += "\"" + a + "\""
	}
	argv, _ := syscall.UTF16PtrFromString(argStr)
	dir, _ := syscall.UTF16PtrFromString("")

	// ShellExecuteW via Shell32
	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW := shell32.NewProc("ShellExecuteW")

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exe)),
		uintptr(unsafe.Pointer(argv)),
		uintptr(unsafe.Pointer(dir)),
		1, // SW_SHOWNORMAL
	)

	// If ret > 32, it's a success (returns instance handle)
	// If ret <= 32, it's an error
	if int(ret) <= 32 {
		return 0, fmt.Errorf("ShellExecuteW failed with code %d", int(ret))
	}
	return 0, nil
}

// RunCommand executes a command and returns stdout + stderr.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
