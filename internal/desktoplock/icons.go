package desktoplock

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"syscall"
	"unsafe"
)

func iconDataURL(path string) string {
	data, err := getFileIcon(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

func getFileIcon(filePath string) ([]byte, error) {
	pathPtr, err := syscall.UTF16PtrFromString(filePath)
	if err != nil {
		return nil, err
	}

	var shfi SHFILEINFOW
	ret, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&shfi)),
		unsafe.Sizeof(shfi),
		0x100, // SHGFI_ICON
	)
	if ret == 0 || shfi.hIcon == 0 {
		return nil, fmt.Errorf("SHGetFileInfoW returned no icon")
	}
	defer procDestroyIcon.Call(shfi.hIcon)

	return hiconToPNG(shfi.hIcon)
}

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

type ICONINFO struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type SHFILEINFOW struct {
	hIcon         uintptr
	iIcon         int32
	dwAttributes  uint32
	szDisplayName [260]uint16
	szTypeName    [80]uint16
}

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	gdi32             = syscall.NewLazyDLL("gdi32.dll")
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procSHGetFileInfo = shell32.NewProc("SHGetFileInfoW")
	procDestroyIcon   = user32.NewProc("DestroyIcon")
)

func hiconToPNG(hIcon uintptr) ([]byte, error) {
	getIconInfo := user32.NewProc("GetIconInfo")
	getObject := gdi32.NewProc("GetObjectW")
	getDC := user32.NewProc("GetDC")
	releaseDC := user32.NewProc("ReleaseDC")
	createDC := gdi32.NewProc("CreateCompatibleDC")
	createBitmap := gdi32.NewProc("CreateCompatibleBitmap")
	selectObject := gdi32.NewProc("SelectObject")
	getDIBits := gdi32.NewProc("GetDIBits")
	deleteDC := gdi32.NewProc("DeleteDC")
	deleteObject := gdi32.NewProc("DeleteObject")
	drawIcon := user32.NewProc("DrawIconEx")

	var iconInfo ICONINFO
	if ret, _, _ := getIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&iconInfo))); ret == 0 {
		return nil, fmt.Errorf("GetIconInfo failed")
	}
	if iconInfo.hbmColor != 0 {
		defer deleteObject.Call(iconInfo.hbmColor)
	}
	if iconInfo.hbmMask != 0 {
		defer deleteObject.Call(iconInfo.hbmMask)
	}

	var bitmap BITMAP
	if ret, _, _ := getObject.Call(iconInfo.hbmColor, unsafe.Sizeof(bitmap), uintptr(unsafe.Pointer(&bitmap))); ret == 0 {
		return nil, fmt.Errorf("GetObjectW failed")
	}
	width, height := bitmap.bmWidth, bitmap.bmHeight
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid icon bitmap size")
	}

	screenDC, _, _ := getDC.Call(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer releaseDC.Call(0, screenDC)

	memoryDC, _, _ := createDC.Call(screenDC)
	if memoryDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer deleteDC.Call(memoryDC)

	colorBitmap, _, _ := createBitmap.Call(screenDC, uintptr(width), uintptr(height))
	if colorBitmap == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer deleteObject.Call(colorBitmap)

	oldObject, _, _ := selectObject.Call(memoryDC, colorBitmap)
	if oldObject == 0 {
		return nil, fmt.Errorf("SelectObject failed")
	}
	drawResult, _, _ := drawIcon.Call(memoryDC, 0, 0, hIcon, uintptr(width), uintptr(height), 0, 0, 3)
	selectObject.Call(memoryDC, oldObject)
	if drawResult == 0 {
		return nil, fmt.Errorf("DrawIconEx failed")
	}

	pixels := make([]byte, int(width)*int(height)*4)
	header := BITMAPINFOHEADER{
		biSize:     uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		biWidth:    width,
		biHeight:   -height,
		biPlanes:   1,
		biBitCount: 32,
	}
	info := BITMAPINFO{bmiHeader: header}
	if ret, _, _ := getDIBits.Call(memoryDC, colorBitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0); ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	rgba := make([]byte, len(pixels))
	for offset := 0; offset < len(pixels); offset += 4 {
		rgba[offset] = pixels[offset+2]
		rgba[offset+1] = pixels[offset+1]
		rgba[offset+2] = pixels[offset]
		rgba[offset+3] = pixels[offset+3]
	}
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	img.Pix = rgba

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
