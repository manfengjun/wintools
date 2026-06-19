package desktoplock

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func iconCacheDir() string {
	return filepath.Join(backupDir(), ".iconcache")
}

// ── Windows API types ──────────────────────────────────────

type (
	HICON   uintptr
	HBITMAP uintptr
	HDC     uintptr
)

type BITMAP struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type BITMAPINFOHEADER struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type BITMAPINFO struct {
	bmiHeader BITMAPINFOHEADER
	bmiColors [4]uint32
}

type SHFILEINFOW struct {
	hIcon        uintptr
	iIcon        int32
	dwAttributes uint32
	szDisplayName [260]uint16
	szTypeName   [80]uint16
}

type ICONINFO struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	gdi32   = syscall.NewLazyDLL("gdi32.dll")
	shell32 = syscall.NewLazyDLL("shell32.dll")
)

// GetBackupIcons 批量获取所有备份快捷方式的图标。
// 纯 Windows API（无 PowerShell 依赖），优先读缓存，缺失则 GDI 提取。
func (a *API) GetBackupIcons() map[string]string {
	files := scanBackupDir()
	if len(files) == 0 {
		return map[string]string{}
	}

	os.MkdirAll(iconCacheDir(), 0755)
	result := make(map[string]string, len(files))
	var uncached []string

	for _, name := range files {
		cachePath := filepath.Join(iconCacheDir(), name+".png")
		data, err := os.ReadFile(cachePath)
		if err == nil {
			result[name] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
		} else {
			result[name] = ""
			uncached = append(uncached, name)
		}
	}

	if len(uncached) > 0 {
		batchExtractGDI(uncached, result)
	}

	return result
}

// batchExtractGDI 用 GDI 批量提取图标（无 PowerShell），每个文件 goroutine + 超时。
func batchExtractGDI(names []string, result map[string]string) {
	type taskResult struct {
		name string
		png  []byte
	}

	ch := make(chan taskResult, len(names))

	for _, name := range names {
		go func(n string) {
			defer func() { recover() }()
			path := filepath.Join(backupDir(), n)
			png, err := extractIconGDI(path)
			if err == nil && len(png) > 0 {
				ch <- taskResult{n, png}
			}
		}(name)
	}

	// 等待所有 goroutine 完成或超时（每个最多 3 秒）
	timeout := time.After(5 * time.Second)
	for i := 0; i < len(names); i++ {
		select {
		case r := <-ch:
			cachePath := filepath.Join(iconCacheDir(), r.name+".png")
			os.WriteFile(cachePath, r.png, 0644)
			result[r.name] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(r.png)
		case <-timeout:
			return
		}
	}
}

// extractIconGDI 用纯 GDI 从文件提取图标，返回 PNG 字节。
func extractIconGDI(path string) ([]byte, error) {
	done := make(chan []byte, 1)
	go func() {
		defer func() { recover(); done <- nil }()
		if png, err := gdiExtract(path); err == nil {
			done <- png
		} else {
			done <- nil
		}
	}()
	select {
	case png := <-done:
		if png == nil {
			return nil, fmt.Errorf("extract failed")
		}
		return png, nil
	case <-time.After(3 * time.Second):
		return nil, fmt.Errorf("timeout")
	}
}

func gdiExtract(path string) ([]byte, error) {
	procSHGetFileInfo := shell32.NewProc("SHGetFileInfoW")
	procDestroyIcon := user32.NewProc("DestroyIcon")
	procGetIconInfo := user32.NewProc("GetIconInfoW")
	procGetObject := gdi32.NewProc("GetObjectW")
	procGetDC := user32.NewProc("GetDC")
	procReleaseDC := user32.NewProc("ReleaseDC")
	procCreateComDC := gdi32.NewProc("CreateCompatibleDC")
	procCreateBitmap := gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObj := gdi32.NewProc("SelectObject")
	procGetDIBits := gdi32.NewProc("GetDIBits")
	procDeleteDC := gdi32.NewProc("DeleteDC")
	procDeleteObject := gdi32.NewProc("DeleteObject")
	procDrawIconEx := user32.NewProc("DrawIconEx")

	// 1. SHGetFileInfoW → HICON
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	var shfi SHFILEINFOW
	ret, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0x80, // FILE_ATTRIBUTE_NORMAL
		uintptr(unsafe.Pointer(&shfi)),
		unsafe.Sizeof(shfi),
		0x100, // SHGFI_ICON
	)
	if ret == 0 || shfi.hIcon == 0 {
		return nil, fmt.Errorf("SHGetFileInfo no icon")
	}
	hIcon := shfi.hIcon
	defer procDestroyIcon.Call(hIcon)

	// 2. GetIconInfo → hbmColor
	var ii ICONINFO
	if ret, _, _ := procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&ii))); ret == 0 {
		return nil, fmt.Errorf("GetIconInfo failed")
	}
	if ii.hbmColor != 0 {
		defer procDeleteObject.Call(ii.hbmColor)
	}
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(ii.hbmMask)
	}

	// 3. BITMAP → 尺寸
	var bm BITMAP
	procGetObject.Call(ii.hbmColor, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))
	w, h := bm.bmWidth, bm.bmHeight
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid bitmap size")
	}

	// 4. GetDC(0) 获取屏幕 DC（比 CreateCompatibleDC(0) 更可靠）
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, screenDC)

	dc, _, _ := procCreateComDC.Call(screenDC)
	if dc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(dc)

	hBmp, _, _ := procCreateBitmap.Call(dc, uintptr(w), uintptr(h))
	if hBmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer procDeleteObject.Call(hBmp)

	procSelectObj.Call(dc, hBmp)
	procDrawIconEx.Call(dc, 0, 0, hIcon, uintptr(w), uintptr(h), 0, 0, 3)

	// 5. GetDIBits → 像素数据
	pixels := make([]byte, w*h*4)
	bih := BITMAPINFOHEADER{
		biSize:    uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		biWidth:   w,
		biHeight:  -h,
		biPlanes:  1,
		biBitCount: 32,
	}
	bmi := BITMAPINFO{bmiHeader: bih}
	if ret, _, _ := procGetDIBits.Call(dc, hBmp, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)), 0); ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	// 6. BGRA → RGBA → PNG
	rgba := make([]byte, w*h*4)
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			s := (y*int(w) + x) * 4
			rgba[s] = pixels[s+2]
			rgba[s+1] = pixels[s+1]
			rgba[s+2] = pixels[s]
			rgba[s+3] = pixels[s+3]
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	img.Pix = rgba

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
