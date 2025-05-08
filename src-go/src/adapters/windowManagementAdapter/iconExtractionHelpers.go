package windowManagementAdapter

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"unsafe"

	w32 "github.com/gonutz/w32/v2"
)

// extractIconHandles extracts large and small icon handles from an executable.
func extractIconHandles(exePath string) (hIconLarge, hIconSmall w32.HICON, err error) {
	// shell32 and extractIconEx could be package-level vars if loaded once
	shell32 := syscall.MustLoadDLL("shell32.dll")           // Consider error handling for MustLoadDLL
	extractIconEx := shell32.MustFindProc("ExtractIconExW") // Consider error handling for MustFindProc

	exePathPtr, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return 0, 0, fmt.Errorf("UTF16PtrFromString for %q failed: %w", exePath, err)
	}

	_, _, lastErr := extractIconEx.Call(
		uintptr(unsafe.Pointer(exePathPtr)),
		0, // nIconIndex
		uintptr(unsafe.Pointer(&hIconLarge)),
		uintptr(unsafe.Pointer(&hIconSmall)),
		1, // nIcons
	)

	if hIconLarge == 0 && hIconSmall == 0 {
		errno := syscall.Errno(0)
		if lastErr != nil && !errors.Is(lastErr, syscall.Errno(0)) { // syscall.Errno(0) is ERROR_SUCCESS
			if osErr, ok := lastErr.(syscall.Errno); ok {
				errno = osErr
			} else {
				return 0, 0, fmt.Errorf("ExtractIconExW for %q returned non-errno: %v", exePath, lastErr)
			}
		}
		if errno == 0 { // No OS error, but no icons found
			return 0, 0, ErrIconNotFound
		}
		return 0, 0, fmt.Errorf("ExtractIconExW for %q failed: %w", exePath, errno)
	}
	return hIconLarge, hIconSmall, nil
}

// renderIconToBGRA draws an HICON to a 32-bit BGRA DIB section.
func renderIconToBGRA(hIcon w32.HICON, size int) ([]byte, error) {
	screenDC := w32.GetDC(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC(0) failed: %w", syscall.GetLastError())
	}
	defer w32.ReleaseDC(0, screenDC)

	memDC := w32.CreateCompatibleDC(screenDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed: %w", syscall.GetLastError())
	}
	defer w32.DeleteDC(memDC)

	var bi w32.BITMAPINFO
	hdr := &bi.BmiHeader
	hdr.BiSize = uint32(unsafe.Sizeof(*hdr))
	hdr.BiWidth = int32(size)
	hdr.BiHeight = int32(-size) // Top-down
	hdr.BiPlanes = 1
	hdr.BiBitCount = 32
	hdr.BiCompression = w32.BI_RGB

	var dibDataPtr unsafe.Pointer
	bitmap := w32.CreateDIBSection(memDC, &bi, w32.DIB_RGB_COLORS, &dibDataPtr, 0, 0)
	if bitmap == 0 {
		return nil, fmt.Errorf("CreateDIBSection failed: %w", syscall.GetLastError())
	}
	defer w32.DeleteObject(w32.HGDIOBJ(bitmap))

	oldBitmap := w32.SelectObject(memDC, w32.HGDIOBJ(bitmap))
	if oldBitmap == 0 {
		return nil, errors.New("SelectObject for DIBSection failed")
	}
	defer w32.SelectObject(memDC, oldBitmap)

	if !w32.DrawIconEx(memDC, 0, 0, hIcon, size, size, 0, 0, gdiDrawIconNormal) {
		return nil, fmt.Errorf("DrawIconEx failed: %w", syscall.GetLastError())
	}

	pixelData := make([]byte, size*size*4)
	if w32.GetDIBits(memDC, bitmap, 0, uint(size), unsafe.Pointer(&pixelData[0]), &bi, w32.DIB_RGB_COLORS) == 0 {
		return nil, fmt.Errorf("GetDIBits after DrawIconEx failed: %w", syscall.GetLastError())
	}
	return pixelData, nil
}

// --- GetIconInfoEx Path Helpers ---

// getBitmapPixelData extracts pixel data from an HBITMAP.
// Returns BGRA data, width, height, bitsPerPixel, and error.
func getBitmapPixelData(hBitmap w32.HBITMAP, debugNameForCtx string) ([]byte, int, int, int, error) {
	if hBitmap == 0 {
		return nil, 0, 0, 0, errors.New("getBitmapPixelData: hBitmap is nil")
	}

	var bmp w32.BITMAP
	if w32.GetObject(w32.HGDIOBJ(hBitmap), unsafe.Sizeof(bmp), unsafe.Pointer(&bmp)) == 0 {
		return nil, 0, 0, 0, fmt.Errorf("GetObject for %s failed: %w", debugNameForCtx, syscall.GetLastError())
	}
	// log.Printf("Debug GIIEX %s Details: W:%d, H:%d, Planes:%d, BPP:%d", debugNameForCtx, bmp.BmWidth, bmp.BmHeight, bmp.BmPlanes, bmp.BmBitsPixel)

	width, height := int(bmp.BmWidth), int(bmp.BmHeight)
	bpp := int(bmp.BmBitsPixel)
	if width == 0 || height == 0 {
		return nil, width, height, bpp, fmt.Errorf("%s has zero dimension (W:%d, H:%d)", debugNameForCtx, width, height)
	}

	var bi w32.BITMAPINFO
	hdr := &bi.BmiHeader
	hdr.BiSize = uint32(unsafe.Sizeof(*hdr))
	hdr.BiWidth = bmp.BmWidth
	hdr.BiHeight = -bmp.BmHeight // Read top-down
	hdr.BiPlanes = 1
	hdr.BiBitCount = uint16(bpp) // Use actual BPP from GetObject
	hdr.BiCompression = w32.BI_RGB

	// For non-32bpp, GetDIBits might convert, or we might need specific handling.
	// The common case we're interested in for color is 32bpp, for mask is 1bpp.
	// If HbmColor is 24bpp, GetDIBits with a 32bpp BITMAPINFO target might work or pad.

	memDC := w32.CreateCompatibleDC(0)
	if memDC == 0 {
		return nil, width, height, bpp, fmt.Errorf("CreateCompatibleDC for %s GetDIBits failed: %w", debugNameForCtx, syscall.GetLastError())
	}
	defer w32.DeleteDC(memDC)

	// Calculate stride and pixel data size
	// For 1bpp mask: stride = ((width * 1 + 31) / 32) * 4
	// For 32bpp color: stride = width * 4 (already 4-byte aligned per pixel)
	var stride int
	if bpp == 1 {
		stride = ((width + 31) &^ 31) / 8
	} else if bpp == 32 {
		stride = width * 4
	} else {
		// For other BPPs, a full DIB conversion might be needed if GetDIBits doesn't handle it well.
		// This simplified version might only work well for 1bpp and 32bpp sources.
		// For now, we assume GetDIBits will work for 32bpp targets.
		// If source is 24bpp and target BITMAPINFO is 32bpp, GetDIBits might expand it.
		hdr.BiBitCount = 32 // Request 32bpp output from GetDIBits
		stride = width * 4
		log.Printf("Info GIIEX %s: Source BPP is %d, requesting 32bpp output from GetDIBits.", debugNameForCtx, bpp)
	}

	pixelData := make([]byte, stride*abs(height)) // abs(height) because BiHeight can be negative

	if w32.GetDIBits(memDC, hBitmap, 0, uint(abs(height)), unsafe.Pointer(&pixelData[0]), &bi, w32.DIB_RGB_COLORS) == 0 {
		return nil, width, height, bpp, fmt.Errorf("GetDIBits for %s (W:%d,H:%d,BPP:%d->req32) failed: %w", debugNameForCtx, width, height, bpp, syscall.GetLastError())
	}

	// If we requested 32bpp output, the data is now BGRA.
	return pixelData, width, height, 32, nil // Return 32 as the effective BPP of pixelData
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// applyAlphaFromMask applies a 1bpp mask to 32bpp BGRA color data.
// colorBGRA is modified in place.
func applyAlphaFromMask(colorBGRA []byte, colorWidth, colorHeight int, maskData []byte, maskWidth, maskHeight int) error {
	if len(colorBGRA) != colorWidth*colorHeight*4 {
		return fmt.Errorf("colorBGRA data size mismatch (expected %d, got %d)", colorWidth*colorHeight*4, len(colorBGRA))
	}
	// Mask dimensions must match color dimensions for this simple application
	if maskWidth != colorWidth || (maskHeight != colorHeight && maskHeight != colorHeight*2) {
		return fmt.Errorf("mask dimensions (W%d,H%d) incompatible with color (W%d,H%d)", maskWidth, maskHeight, colorWidth, colorHeight)
	}

	maskStride := ((maskWidth + 31) &^ 31) / 8
	if len(maskData) < maskStride*colorHeight { // Only need up to colorHeight of the mask
		return fmt.Errorf("maskData size too small (expected at least %d for W%dH%d, got %d)", maskStride*colorHeight, maskWidth, colorHeight, len(maskData))
	}

	idx := 0 // Index for colorBGRA
	for y := range colorHeight {
		for x := range colorWidth {
			maskByte := maskData[y*maskStride+x/8]
			maskBit := (maskByte >> (7 - (x % 8))) & 1 // 0 is opaque, 1 is transparent
			if maskBit == 0 {
				colorBGRA[idx+3] = 255 // Set Alpha to Opaque
			} else {
				colorBGRA[idx+3] = 0 // Set Alpha to Transparent
			}
			idx += 4
		}
	}
	return nil
}

// isBGRADataFullyTransparent checks if 32bpp BGRA data is all zeros in alpha channel.
func isBGRADataFullyTransparent(bgraData []byte, width, height int) bool {
	if len(bgraData) != width*height*4 {
		return true
	} // Invalid data considered transparent
	for i := 3; i < len(bgraData); i += 4 {
		if bgraData[i] != 0 {
			return false
		}
	}
	return true
}