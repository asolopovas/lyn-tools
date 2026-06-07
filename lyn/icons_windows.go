//go:build windows

package lyn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	shgfiIcon      = 0x000000100
	shgfiLargeIcon = 0x000000000
	dibRGBColors   = 0
	biRGB          = 0
)

type windowsSHFileInfo struct {
	icon        syscall.Handle
	iconIndex   int32
	attributes  uint32
	displayName [260]uint16
	typeName    [80]uint16
}

type windowsIconInfo struct {
	icon     int32
	xHotspot uint32
	yHotspot uint32
	mask     syscall.Handle
	color    syscall.Handle
}

type windowsBitmap struct {
	typ        int32
	width      int32
	height     int32
	widthBytes int32
	planes     uint16
	bitsPixel  uint16
	bits       uintptr
}

type windowsBitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

type windowsRGBQuad struct {
	blue     byte
	green    byte
	red      byte
	reserved byte
}

type windowsBitmapInfo struct {
	header windowsBitmapInfoHeader
	colors [1]windowsRGBQuad
}

var (
	windowsShell32       = syscall.NewLazyDLL("shell32.dll")
	windowsUser32        = syscall.NewLazyDLL("user32.dll")
	windowsGdi32         = syscall.NewLazyDLL("gdi32.dll")
	windowsSHGetFileInfo = windowsShell32.NewProc("SHGetFileInfoW")
	windowsDestroyIcon   = windowsUser32.NewProc("DestroyIcon")
	windowsGetIconInfo   = windowsUser32.NewProc("GetIconInfo")
	windowsGetDC         = windowsUser32.NewProc("GetDC")
	windowsReleaseDC     = windowsUser32.NewProc("ReleaseDC")
	windowsGetObject     = windowsGdi32.NewProc("GetObjectW")
	windowsGetDIBits     = windowsGdi32.NewProc("GetDIBits")
	windowsDeleteObject  = windowsGdi32.NewProc("DeleteObject")
)

func init() {
	windowsAppIcon = resolveWindowsAppIcon
}

func resolveWindowsAppIcon(ctx context.Context, cacheDir string, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	out := iconCachePath(cacheDir, path)
	if _, err := os.Stat(out); err == nil {
		return iconDataURI(out)
	}
	icon, ok := windowsAssociatedIcon(path)
	if !ok {
		return "", nil
	}
	defer windowsDestroyIcon.Call(uintptr(icon))
	img, ok := windowsIconImage(icon)
	if !ok {
		return "", nil
	}
	file, err := os.Create(out)
	if err != nil {
		return "", err
	}
	err = png.Encode(file, img)
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return iconDataURI(out)
}

func iconCachePath(dir string, path string) string {
	info, _ := os.Stat(path)
	key := path
	if info != nil {
		key = key + info.ModTime().UTC().String() + info.Name()
	}
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".png")
}

func windowsAssociatedIcon(path string) (syscall.Handle, bool) {
	target, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	var info windowsSHFileInfo
	ret, _, _ := windowsSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(target)),
		0,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
		uintptr(shgfiIcon|shgfiLargeIcon),
	)
	return info.icon, ret != 0 && info.icon != 0
}

func windowsIconImage(icon syscall.Handle) (*image.RGBA, bool) {
	var iconInfo windowsIconInfo
	ret, _, _ := windowsGetIconInfo.Call(uintptr(icon), uintptr(unsafe.Pointer(&iconInfo)))
	if ret == 0 {
		return nil, false
	}
	defer deleteWindowsObject(iconInfo.color)
	defer deleteWindowsObject(iconInfo.mask)
	if iconInfo.color == 0 {
		return nil, false
	}
	var bitmap windowsBitmap
	ret, _, _ = windowsGetObject.Call(
		uintptr(iconInfo.color),
		unsafe.Sizeof(bitmap),
		uintptr(unsafe.Pointer(&bitmap)),
	)
	if ret == 0 || bitmap.width <= 0 || bitmap.height <= 0 {
		return nil, false
	}
	width := int(bitmap.width)
	height := int(bitmap.height)
	pixels := make([]byte, width*height*4)
	info := windowsBitmapInfo{
		header: windowsBitmapInfoHeader{
			size:        uint32(unsafe.Sizeof(windowsBitmapInfoHeader{})),
			width:       bitmap.width,
			height:      -bitmap.height,
			planes:      1,
			bitCount:    32,
			compression: biRGB,
			sizeImage:   uint32(len(pixels)),
		},
	}
	hdcRet, _, _ := windowsGetDC.Call(0)
	if hdcRet == 0 {
		return nil, false
	}
	defer windowsReleaseDC.Call(0, hdcRet)
	ret, _, _ = windowsGetDIBits.Call(
		hdcRet,
		uintptr(iconInfo.color),
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&info)),
		uintptr(dibRGBColors),
	)
	if ret == 0 {
		return nil, false
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	hasAlpha := false
	for i := 0; i < width*height; i++ {
		b := pixels[i*4]
		g := pixels[i*4+1]
		r := pixels[i*4+2]
		a := pixels[i*4+3]
		if a != 0 {
			hasAlpha = true
		}
		img.Pix[i*4] = r
		img.Pix[i*4+1] = g
		img.Pix[i*4+2] = b
		img.Pix[i*4+3] = a
	}
	if !hasAlpha {
		for i := 0; i < width*height; i++ {
			img.Pix[i*4+3] = 255
		}
	}
	return img, true
}

func deleteWindowsObject(handle syscall.Handle) {
	if handle != 0 {
		windowsDeleteObject.Call(uintptr(handle))
	}
}
