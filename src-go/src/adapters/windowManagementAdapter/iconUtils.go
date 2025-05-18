package windowManagementAdapter

import (
	"image"
	"log"
	"unsafe"

	w32 "github.com/gonutz/w32/v2"
)

// --- Constants ---
const (
	appDataIconSubdir     = "appIcons" // Relative to project root
	webIconPathPrefix     = "/appIcons"       // URL path prefix for serving icons
	defaultIconSize       = 32
	iconHashLength        = 8
	maxIconBaseNameLength = 50
	gdiDrawIconNormal     = 0x0003 // Flag for DrawIconEx: DI_NORMAL
)

// --- Win32 API Structures & Functions (Local definitions if not in w32 or for clarity) ---

// ICONINFOEXW structure for GetIconInfoExW.
type ICONINFOEXW struct {
	CbSize    uint32
	FIcon     bool
	XHotspot  uint32
	YHotspot  uint32
	HbmMask   w32.HBITMAP
	HbmColor  w32.HBITMAP
	WResID    uint16
	SzModName [w32.MAX_PATH]uint16
	SzResName [w32.MAX_PATH]uint16
}

// localGetIconInfoEx wraps the GetIconInfoExW syscall.
func localGetIconInfoEx(hIcon w32.HICON, piconinfo *ICONINFOEXW) bool {
	piconinfo.CbSize = uint32(unsafe.Sizeof(*piconinfo))
	ret, _, _ := procGetIconInfoExW.Call(uintptr(hIcon), uintptr(unsafe.Pointer(piconinfo)))
	return ret != 0
}

// bgraToGoImage converts raw BGRA pixel data to a Go image.RGBA.
// Ensure this is defined or imported.
func bgraToGoImage(bgraData []byte, width, height int) *image.RGBA {
	if len(bgraData) < width*height*4 {
		log.Printf("Error bgraToGoImage: not enough data. Expected %d, got %d for %dx%d", width*height*4, len(bgraData), width, height)
		// Return an empty/black image or handle error appropriately
		return image.NewRGBA(image.Rect(0,0,width,height))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pixels := img.Pix
	j := 0
	for i := 0; i < width*height*4; i += 4 {
		pixels[j+0] = bgraData[i+2] // R
		pixels[j+1] = bgraData[i+1] // G
		pixels[j+2] = bgraData[i+0] // B
		pixels[j+3] = bgraData[i+3] // A
		j += 4
	}
	return img
}