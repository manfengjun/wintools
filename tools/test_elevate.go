package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	// 使用 ShellExecuteExW 来提权并等待完成
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteEx := shell32.NewProc("ShellExecuteExW")

	type SHELLEXECUTEINFO struct {
		Size       uint32
		Mask       uint32
		Hwnd       uintptr
		Verb       *uint16
		File       *uint16
		Parameters *uint16
		Directory  *uint16
		Show       int32
		InstApp    uintptr
		IDList     uintptr
		ClassName  *uint16
		HkeyClass  uintptr
		HotKey     uint32
		Icon       uintptr
		Process    uintptr
	}

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString("cmd.exe")
	param, _ := syscall.UTF16PtrFromString("/c echo test")

	info := SHELLEXECUTEINFO{
		Size:       uint32(unsafe.Sizeof(SHELLEXECUTEINFO{})),
		Mask:       0x00000040, // SEE_MASK_NOCLOSEPROCESS
		Verb:       verb,
		File:       file,
		Parameters: param,
		Show:       1, // SW_SHOWNORMAL
	}

	ret, _, err := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		fmt.Printf("ShellExecuteExW failed: %v\n", err)
		return
	}

	fmt.Printf("Process handle: %v\n", info.Process)
	if info.Process != 0 {
		windows.WaitForSingleObject(info.Process, windows.INFINITE)
		windows.CloseHandle(info.Process)
		fmt.Println("Process completed")
	}
}
