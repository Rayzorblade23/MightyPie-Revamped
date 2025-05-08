package windowManagementAdapter

import (
	"fmt"
	"image"
	"log"
	"path/filepath"
	"syscall"
	"unsafe"

	w32 "github.com/gonutz/w32/v2"
	"golang.org/x/image/draw" // For resizing
)

// getIconImageViaGetIconInfoEx attempts to extract an icon using GetIconInfoExW,
// prioritizing existing alpha in HbmColor, then attempting mask blending.
func getIconImageViaGetIconInfoEx(exeName string, hIcon w32.HICON) (*image.RGBA, int, int, error) {
	var ii ICONINFOEXW
	if !localGetIconInfoEx(hIcon, &ii) {
		return nil, 0, 0, fmt.Errorf("localGetIconInfoEx for %s failed: %w", exeName, syscall.GetLastError())
	}
	if ii.HbmColor != 0 {
		defer w32.DeleteObject(w32.HGDIOBJ(ii.HbmColor))
	}
	if ii.HbmMask != 0 {
		defer w32.DeleteObject(w32.HGDIOBJ(ii.HbmMask))
	}

	// log.Printf("Debug GIIEX for %s: HbmColor:%p, HbmMask:%p", exeName, ii.HbmColor, ii.HbmMask)

	if ii.HbmColor == 0 {
		// TODO: Handle monochrome icons if HbmColor is nil but HbmMask exists.
		// This would involve creating a 32bpp image, colorizing based on mask,
		// then applying mask for transparency. For now, require HbmColor.
		return nil, 0, 0, fmt.Errorf("GetIconInfoEx for %s: HbmColor is nil", exeName)
	}

	// Get HbmColor data, requesting 32bpp output
	colorBGRA, cW, cH, cBPP, err := getBitmapPixelData(ii.HbmColor, exeName+"_HbmColor")
	if err != nil {
		return nil, 0, 0, fmt.Errorf("processing HbmColor for %s failed: %w", exeName, err)
	}
	if cBPP != 32 || colorBGRA == nil { // getBitmapPixelData now aims to return 32bpp data
		return nil, cW, cH, fmt.Errorf("HbmColor for %s not processed into 32bpp BGRA (srcBPP was different or error)", exeName)
	}

	// Check if the 32bpp HbmColor (as returned by getBitmapPixelData) has its own alpha
	if !isBGRADataFullyTransparent(colorBGRA, cW, cH) {
		log.Printf("Debug GIIEX for %s: Using direct 32bpp HbmColor with existing alpha.", exeName)
		return bgraToGoImage(colorBGRA, cW, cH), cW, cH, nil
	}

	// HbmColor is 32bpp but fully transparent (or source was not 32bpp and converted), try mask
	log.Printf("Debug GIIEX for %s: HbmColor is 32bpp but transparent, or needs mask. Attempting to apply HbmMask.", exeName)
	if ii.HbmMask == 0 {
		log.Printf("Debug GIIEX for %s: HbmColor transparent/needs mask, but HbmMask is nil. Returning transparent HbmColor.", exeName)
		// Return the (transparent) image from HbmColor; extractIconFromExe will decide if it's usable
		return bgraToGoImage(colorBGRA, cW, cH), cW, cH, nil
	}

	// getBitmapPixelData for mask should have returned 1bpp data as its sourceBPP, but the []byte might be larger
	// if we forced a 32bpp intermediate in getBitmapPixelData, which we don't for mask
	// Let's re-fetch mask data specifically as 1bpp
	// For simplicity, the previous mask fetching logic was more direct:
	var bmpMask w32.BITMAP
	var maskData1bpp []byte
	if w32.GetObject(w32.HGDIOBJ(ii.HbmMask), unsafe.Sizeof(bmpMask), unsafe.Pointer(&bmpMask)) == 0 {
		log.Printf("Warning GIIEX for %s: GetObject for HbmMask (2nd attempt) failed: %v.", exeName, syscall.GetLastError())
	} else if bmpMask.BmBitsPixel == 1 {
		var biMask w32.BITMAPINFO
		hdrMask := &biMask.BmiHeader
		hdrMask.BiSize = uint32(unsafe.Sizeof(*hdrMask))
		hdrMask.BiWidth = bmpMask.BmWidth
		hdrMask.BiHeight = -bmpMask.BmHeight
		hdrMask.BiPlanes = 1
		hdrMask.BiBitCount = 1
		hdrMask.BiCompression = w32.BI_RGB
		memDC := w32.CreateCompatibleDC(0)
		if memDC != 0 {
			defer w32.DeleteDC(memDC)
			stride1bpp := ((int(bmpMask.BmWidth) + 31) &^ 31) / 8
			maskData1bpp = make([]byte, stride1bpp*int(bmpMask.BmHeight))
			if w32.GetDIBits(memDC, ii.HbmMask, 0, uint(bmpMask.BmHeight), unsafe.Pointer(&maskData1bpp[0]), &biMask, w32.DIB_RGB_COLORS) == 0 {
				maskData1bpp = nil
				log.Printf("Warning GIIEX for %s: GetDIBits for 1bpp HbmMask failed.", exeName)
			}
		}
	}

	if maskData1bpp == nil {
		log.Printf("Warning GIIEX for %s: Could not get 1bpp HbmMask data. Using (transparent) HbmColor.", exeName)
		return bgraToGoImage(colorBGRA, cW, cH), cW, cH, nil
	}

	if err := applyAlphaFromMask(colorBGRA, cW, cH, maskData1bpp, int(bmpMask.BmWidth), int(bmpMask.BmHeight)); err != nil {
		log.Printf("Warning GIIEX for %s: Failed to apply alpha from mask (%v). Using HbmColor as-is.", exeName, err)
		return bgraToGoImage(colorBGRA, cW, cH), cW, cH, nil // Return (transparent) HbmColor
	}

	log.Printf("Debug GIIEX for %s: Successfully applied HbmMask to HbmColor.", exeName)
	return bgraToGoImage(colorBGRA, cW, cH), cW, cH, nil
}

// extractIconFromExe attempts to extract an icon, trying DrawIconEx first,
// then GetIconInfoEx with mask blending if needed.
func extractIconFromExe(exePath string, targetSize int) (image.Image, error) {
	baseName := filepath.Base(exePath)
	if targetSize <= 0 {
		targetSize = defaultIconSize
	}

	hIconLarge, hIconSmall, err := extractIconHandles(exePath)
	if err != nil {
		return nil, err
	} // Already context-aware

	selectedIcon := hIconLarge
	if selectedIcon == 0 {
		selectedIcon = hIconSmall
	}
	if selectedIcon == 0 {
		// Ensure handles are destroyed if one was non-zero but not selected (should not happen with current logic)
		if hIconLarge != 0 {
			w32.DestroyIcon(hIconLarge)
		}
		if hIconSmall != 0 {
			w32.DestroyIcon(hIconSmall)
		}
		return nil, ErrIconNotFound
	}

	// Cleanup unused handle
	if hIconLarge != 0 && hIconSmall != 0 {
		if selectedIcon == hIconLarge {
			w32.DestroyIcon(hIconSmall)
		} else {
			w32.DestroyIcon(hIconLarge)
		}
	} // Single handle case, selectedIcon is the one to destroy via defer
	defer w32.DestroyIcon(selectedIcon)

	// Attempt 1: DrawIconEx (renderIconToBGRA)
	// log.Printf("Debug Extractor for %s: Attempt 1: DrawIconEx path.", baseName)
	bgraDataDrawIconEx, errRender := renderIconToBGRA(selectedIcon, targetSize)
	if errRender == nil && bgraDataDrawIconEx != nil {
		if !isBGRADataFullyTransparent(bgraDataDrawIconEx, targetSize, targetSize) {
			// log.Printf("Debug Extractor for %s: DrawIconEx path successful (non-transparent).", baseName)
			return bgraToGoImage(bgraDataDrawIconEx, targetSize, targetSize), nil
		}
		log.Printf("Debug Extractor for %s: DrawIconEx produced transparent image. Trying GetIconInfoEx.", baseName)
	} else if errRender != nil {
		log.Printf("Warning Extractor for %s: DrawIconEx failed (%v). Trying GetIconInfoEx.", baseName, errRender)
	} else { // bgraDataDrawIconEx is nil without error
		log.Printf("Warning Extractor for %s: DrawIconEx returned nil data. Trying GetIconInfoEx.", baseName)
	}

	// Attempt 2: GetIconInfoEx path
	// log.Printf("Debug Extractor for %s: Attempt 2: GetIconInfoEx path.", baseName)
	imgGIIEX, nativeW, nativeH, errGIIEX := getIconImageViaGetIconInfoEx(baseName, selectedIcon)
	if errGIIEX != nil {
		return nil, fmt.Errorf("all extraction paths failed for %s (DrawIconEx err: %v; GetIconInfoEx err: %w)", baseName, errRender, errGIIEX)
	}
	if imgGIIEX == nil { // Should be caught by errGIIEX, but defensive
		return nil, fmt.Errorf("GetIconInfoEx for %s returned nil image without error", baseName)
	}

	// Check if image from GetIconInfoEx is non-transparent
	isTransparentGIIEX := true
	for i := 3; i < len(imgGIIEX.Pix); i += 4 {
		if imgGIIEX.Pix[i] != 0 {
			isTransparentGIIEX = false
			break
		}
	}
	if isTransparentGIIEX {
		log.Printf("Warning Extractor for %s: GetIconInfoEx path also resulted in a transparent image.", baseName)
		return nil, ErrIconNotProcessed // Or a more specific error
	}

	// log.Printf("Debug Extractor for %s: GetIconInfoEx path successful (non-transparent). Native: %dx%d", baseName, nativeW, nativeH)
	if nativeW == targetSize && nativeH == targetSize {
		return imgGIIEX, nil
	}
	if nativeW > 0 && nativeH > 0 { // Ensure valid dimensions
		resizedImg := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
		draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), imgGIIEX, imgGIIEX.Bounds(), draw.Over, nil)
		return resizedImg, nil
	}
	return nil, fmt.Errorf("GetIconInfoEx for %s yielded zero dimensions (%dx%d) after processing", baseName, nativeW, nativeH)
}
