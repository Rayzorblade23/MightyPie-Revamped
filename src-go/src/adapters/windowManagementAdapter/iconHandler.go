package windowManagementAdapter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/image/draw" // For resizing
	"image"
	"image/png"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	w32 "github.com/gonutz/w32/v2"
)

// --- Constants ---
const (
	AppDataIconSubdir     = "static/appIcons"
	WebIconPath           = "/appIcons"
	defaultIconSize       = 32
	iconHashLength        = 8
	maxIconBaseNameLength = 50
	diNormal              = 0x0003 // Flag for DrawIconEx: standard drawing

	iconInfoExSize = uint32(unsafe.Sizeof(ICONINFOEXW{})) // Placeholder if struct defined below

)

// ICONINFOEXW structure, as per Windows API.
// Ensure this matches the expected layout for GetIconInfoExW.
// Check if gonutz/w32 already provides this or a similar structure.
// If it does, use the library's version.
type ICONINFOEXW struct {
	CbSize          uint32
	FIcon           bool
	XHotspot        uint32
	YHotspot        uint32
	HbmMask         w32.HBITMAP
	HbmColor        w32.HBITMAP
	WResID          uint16
	SzModName       [w32.MAX_PATH]uint16
	SzResName       [w32.MAX_PATH]uint16
	// Note: The actual C struct might have different packing or alignment.
	// This is a common Go representation. If issues persist, meticulous
	// checking against the C struct definition and gonutz/w32 is needed.
}

// --- Custom Errors ---
var (
	ErrIconSkipped  = errors.New("icon skipped, already exists")
	ErrIconNotFound = errors.New("icon not found in executable")
)

// --- Shared State (Icon Directory) ---
var (
	iconDirOnce sync.Once
	iconDirPath string
	iconDirErr  error
)

// getIconStorageDir finds or creates the directory for storing icons.
func getIconStorageDir() (string, error) {
	iconDirOnce.Do(func() {
		rootDir, err := getRootDir()
		if err != nil {
			iconDirErr = fmt.Errorf("failed to determine project root: %w", err)
			return
		}

		dir := filepath.Join(rootDir, AppDataIconSubdir)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			iconDirErr = fmt.Errorf("failed to create icon directory '%s': %w", dir, err)
			return
		}
		iconDirPath = dir
	})
	return iconDirPath, iconDirErr
}

// getRootDir returns the project root directory by going up from the current working directory
func getRootDir() (string, error) {
	// First try working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up from working directory
	dir := wd
	for {
		// Check for markers of project root (like go.mod or the src-go directory)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found go.mod, go up one more level
			return filepath.Dir(dir), nil
		}
		if filepath.Base(dir) == "src-go" {
			// Found src-go directory, go up two levels
			return filepath.Dir(filepath.Dir(dir)), nil
		}

		// Go up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding project
			return "", fmt.Errorf("could not find project root directory from %s", wd)
		}
		dir = parent
	}
}

// --- Icon Filename Generation ---

var iconFilenameReplacer = strings.NewReplacer(
	":", "_", `\`, "_", `/`, "_", `*`, "_", `?`, "_", `"`, "_", `<`, "_", `>`, "_", `|`, "_",
)

// generateIconFilename creates a unique and safe filename for the icon PNG.
func generateIconFilename(exePath string) string {
	baseName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	safeBaseName := iconFilenameReplacer.Replace(baseName)
	if len(safeBaseName) > maxIconBaseNameLength {
		safeBaseName = safeBaseName[:maxIconBaseNameLength]
	}
	hasher := sha256.New()
	hasher.Write([]byte(exePath)) // Hash the full path for uniqueness
	hashString := hex.EncodeToString(hasher.Sum(nil)[:iconHashLength])
	return fmt.Sprintf("%s_%s.png", safeBaseName, hashString)
}

// --- Platform-Specific Icon Extraction Helpers (Windows) ---

// extractIconHandles uses ExtractIconExW to get icon handles from an executable.
func extractIconHandles(exePath string) (hIconLarge, hIconSmall w32.HICON, err error) {
	shell32 := syscall.MustLoadDLL("shell32.dll")
	extractIconEx := shell32.MustFindProc("ExtractIconExW")

	exePathPtr, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert path %q to UTF-16: %w", exePath, err)
	}

	// Call ExtractIconExW via syscall.
	// r1: number of icons in the file (we don't strictly need it, checking handles is better)
	// r2: not used by this function
	// lastErr: syscall error result
	_, _, lastErr := extractIconEx.Call(
		uintptr(unsafe.Pointer(exePathPtr)),  // lpszFile
		uintptr(0),                           // nIconIndex (0 for first group)
		uintptr(unsafe.Pointer(&hIconLarge)), // phiconLarge
		uintptr(unsafe.Pointer(&hIconSmall)), // phiconSmall
		uintptr(1),                           // nIcons (request 1 pair)
	)

	// Check if any icon handle was actually returned.
	if hIconLarge == 0 && hIconSmall == 0 {
		// No handles returned. Check if it was due to an OS error or just no icon found.
		errno := syscall.Errno(0)                                    // Assume success but no icon by default
		if lastErr != nil && !errors.Is(lastErr, syscall.Errno(0)) { // Ignore ERROR_SUCCESS
			if osErr, ok := lastErr.(syscall.Errno); ok {
				errno = osErr // It was a specific OS error
			} else {
				// Unexpected error type from syscall.Call
				return 0, 0, fmt.Errorf("ExtractIconExW returned non-errno error for %q: %v", exePath, lastErr)
			}
		}

		// Determine final error based on errno
		if errno == 0 {
			// No OS error, so the icon wasn't found at index 0.
			return 0, 0, ErrIconNotFound
		} else {
			// A specific OS error occurred (e.g., file not found, access denied).
			return 0, 0, fmt.Errorf("ExtractIconExW failed for %q: %w", exePath, errno)
		}
	}

	// At least one handle was extracted successfully.
	return hIconLarge, hIconSmall, nil
}

// renderIconToBGRA draws the icon onto a 32-bit DIBSection and returns its raw BGRA pixel data.
// This method provides better and more explicit alpha channel handling.
func renderIconToBGRA(hIcon w32.HICON, size int) ([]byte, error) {
	screenDC := w32.GetDC(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC(0) failed: %v", syscall.GetLastError())
	}
	defer w32.ReleaseDC(0, screenDC)

	memDC := w32.CreateCompatibleDC(screenDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed: %v", syscall.GetLastError())
	}
	defer w32.DeleteDC(memDC)

	// Prepare BITMAPINFO for a 32-bit top-down DIB (BGRA)
	var bi w32.BITMAPINFO
	// Correctly access the BmiHeader field of w32.BITMAPINFO
	hdr := &bi.BmiHeader

	hdr.BiSize = uint32(unsafe.Sizeof(*hdr)) // Size of the BITMAPINFOHEADER
	hdr.BiWidth = int32(size)
	hdr.BiHeight = int32(-size) // Negative height for top-down DIB
	hdr.BiPlanes = 1
	hdr.BiBitCount = 32            // 32 bits per pixel (BGRA)
	hdr.BiCompression = w32.BI_RGB // Uncompressed

	// Create the DIB Section. ppvBits can be nil if we use GetDIBits later.
	var ppvBits unsafe.Pointer
	// Pass the address of the BITMAPINFO struct (bi)
	bitmap := w32.CreateDIBSection(memDC, &bi, w32.DIB_RGB_COLORS, &ppvBits, 0, 0)
	if bitmap == 0 {
		err := syscall.GetLastError()
		return nil, fmt.Errorf("CreateDIBSection failed (size: %dx%d, bitCount: %d): %v", size, size, hdr.BiBitCount, err)
	}
	defer w32.DeleteObject(w32.HGDIOBJ(bitmap))

	oldBitmap := w32.SelectObject(memDC, w32.HGDIOBJ(bitmap))
	if oldBitmap == 0 {
		return nil, errors.New("SelectObject failed selecting DIBSection")
	}
	defer w32.SelectObject(memDC, oldBitmap)

	success := w32.DrawIconEx(memDC, 0, 0, hIcon, size, size, 0, 0, diNormal)
	if !success {
		err := syscall.GetLastError()
		return nil, fmt.Errorf("DrawIconEx failed for HICON %p (size: %dx%d): %w", hIcon, size, size, err)
	}

	pixelDataSize := size * size * 4
	pixelData := make([]byte, pixelDataSize)

	// Get Pixel Data using GetDIBits. Use the same BITMAPINFO (bi) as for creation.
	scanLinesCopied := w32.GetDIBits(memDC, bitmap, 0, uint(size), unsafe.Pointer(&pixelData[0]), &bi, w32.DIB_RGB_COLORS)
	if scanLinesCopied == 0 {
		err := syscall.GetLastError()
		return nil, fmt.Errorf("GetDIBits failed for HICON %p (size: %dx%d, expected scanlines: %d): %w", hIcon, size, size, size, err)
	}
	if int(scanLinesCopied) != size {
		// This is not necessarily a fatal error if some lines were copied, but it's a warning.
		log.Printf("Warning: GetDIBits copied %d scanlines, expected %d for HICON %p (size: %dx%d)", scanLinesCopied, size, hIcon, size, size)
	}

	return pixelData, nil
}

// bgraToGoImage converts raw BGRA pixel data to a Go image.RGBA.
func bgraToGoImage(bgraData []byte, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pixels := img.Pix
	j := 0                                  // Index for RGBA buffer
	for i := 0; i < len(bgraData); i += 4 { // Index for BGRA buffer
		pixels[j+0] = bgraData[i+2] // R <- B
		pixels[j+1] = bgraData[i+1] // G <- G
		pixels[j+2] = bgraData[i+0] // B <- R
		pixels[j+3] = bgraData[i+3] // A <- A
		j += 4
	}
	return img
}

// getIconInfoEx wraps the GetIconInfoExW syscall.
// Call this instead of w32.GetIconInfoEx if w32.GetIconInfoEx is not available or not working.
func getIconInfoEx(hIcon w32.HICON, piconinfo *ICONINFOEXW) bool {
	ret, _, _ := procGetIconInfoExW.Call(
		uintptr(hIcon),
		uintptr(unsafe.Pointer(piconinfo)),
	)
	return ret != 0
}


// localGetIconInfoEx wraps the GetIconInfoExW syscall using our local ICONINFOEXW struct.
func localGetIconInfoEx(hIcon w32.HICON, piconinfo *ICONINFOEXW) bool {
	// Set CbSize before calling
	piconinfo.CbSize = uint32(unsafe.Sizeof(*piconinfo))
	ret, _, _ := procGetIconInfoExW.Call(
		uintptr(hIcon),
		uintptr(unsafe.Pointer(piconinfo)),
	)
	return ret != 0
}


func (lbd *LoggedBitmapDetails) Populate(exeName, bitmapName string, bmp *w32.BITMAP) {
	lbd.ExeName = exeName
	lbd.BitmapName = bitmapName
	lbd.Width = bmp.BmWidth
	lbd.Height = bmp.BmHeight
	lbd.Planes = bmp.BmPlanes
	lbd.BitsPixel = bmp.BmBitsPixel
	lbd.Valid = true
}

func (lbd *LoggedBitmapDetails) Log() {
	if !lbd.Valid {
		log.Printf("Debug GIIEX for %s: %s details not populated.", lbd.ExeName, lbd.BitmapName)
		return
	}
	log.Printf("Debug GIIEX for %s: %s Details - Width:%d, Height:%d, Planes:%d, BitsPixel:%d",
		lbd.ExeName, lbd.BitmapName, lbd.Width, lbd.Height, lbd.Planes, lbd.BitsPixel)
}
func (lbd *LoggedBitmapDetails) Summary() string {
	if !lbd.Valid { return "N/A" }
	return fmt.Sprintf("W:%d, H:%d, P:%d, BPP:%d", lbd.Width, lbd.Height, lbd.Planes, lbd.BitsPixel)
}

// getIconImageViaGetIconInfoEx attempts to get icon image data using our local GetIconInfoExW call.
func getIconImageViaGetIconInfoEx(exePathForDebug string, hIcon w32.HICON) (img *image.RGBA, nativeWidth, nativeHeight int, err error) {
	var ii ICONINFOEXW
	if !localGetIconInfoEx(hIcon, &ii) {
		return nil, 0, 0, fmt.Errorf("localGetIconInfoEx (GetIconInfoExW) failed for %s: %w", exePathForDebug, syscall.GetLastError())
	}
	if ii.HbmColor != 0 { defer w32.DeleteObject(w32.HGDIOBJ(ii.HbmColor)) }
	if ii.HbmMask != 0 { defer w32.DeleteObject(w32.HGDIOBJ(ii.HbmMask)) }

	// ... (logging of ICONINFOEXW details as before) ...
	modNameEnd := 0; for modNameEnd < len(ii.SzModName) && ii.SzModName[modNameEnd] != 0 { modNameEnd++ }
	modNameStr := syscall.UTF16ToString(ii.SzModName[:modNameEnd])
	resNameEnd := 0; for resNameEnd < len(ii.SzResName) && ii.SzResName[resNameEnd] != 0 { resNameEnd++ }
	resNameStr := syscall.UTF16ToString(ii.SzResName[:resNameEnd])
	log.Printf("Debug GIIEX for %s: CbSize:%d, IsIcon:%v, Hotspot:(%d,%d), HbmColor:%p, HbmMask:%p, ResID:%d, ModName:'%s', ResName:'%s'",
		exePathForDebug, ii.CbSize, ii.FIcon, ii.XHotspot, ii.YHotspot, ii.HbmColor, ii.HbmMask, ii.WResID, modNameStr, resNameStr)


	if ii.HbmColor == 0 && ii.HbmMask == 0 {
		return nil, 0, 0, fmt.Errorf("GetIconInfoEx for %s: both HbmColor and HbmMask are nil", exePathForDebug)
	}

	var colorBitmapDetails LoggedBitmapDetails
	var hbmColorData []byte // BGRA
	var colorWidth, colorHeight int
	isHbmColor32bpp := false
	isHbmColorDataInitiallyTransparent := true // Assume transparent until proven otherwise

	// --- Process HbmColor ---
	if ii.HbmColor != 0 {
		var bmpColor w32.BITMAP
		if w32.GetObject(w32.HGDIOBJ(ii.HbmColor), unsafe.Sizeof(bmpColor), unsafe.Pointer(&bmpColor)) == 0 {
			log.Printf("Warning GIIEX for %s: GetObject for HbmColor failed: %v.", exePathForDebug, syscall.GetLastError())
		} else {
			colorBitmapDetails.Populate(exePathForDebug, "HbmColor", &bmpColor)
			colorBitmapDetails.Log()
			colorWidth = int(bmpColor.BmWidth)
			colorHeight = int(bmpColor.BmHeight)
			nativeWidth, nativeHeight = colorWidth, colorHeight

			if bmpColor.BmBitsPixel == 32 && colorWidth > 0 && colorHeight > 0 {
				isHbmColor32bpp = true
				var biColor w32.BITMAPINFO
				hdrColor := &biColor.BmiHeader
				hdrColor.BiSize = uint32(unsafe.Sizeof(*hdrColor)); hdrColor.BiWidth = bmpColor.BmWidth
				hdrColor.BiHeight = -bmpColor.BmHeight; hdrColor.BiPlanes = 1
				hdrColor.BiBitCount = 32; hdrColor.BiCompression = w32.BI_RGB
				
				memDC := w32.CreateCompatibleDC(0)
				if memDC == 0 {
					log.Printf("Error GIIEX for %s: CreateCompatibleDC for HbmColor GetDIBits failed: %v", exePathForDebug, syscall.GetLastError())
				} else {
					defer w32.DeleteDC(memDC)
					hbmColorData = make([]byte, colorWidth*colorHeight*4)
					scanLinesCopied := w32.GetDIBits(memDC, ii.HbmColor, 0, uint(colorHeight), unsafe.Pointer(&hbmColorData[0]), &biColor, w32.DIB_RGB_COLORS)
					if scanLinesCopied == 0 {
						log.Printf("Error GIIEX for %s: GetDIBits on 32bpp HbmColor FAILED: %v.", exePathForDebug, syscall.GetLastError())
						hbmColorData = nil
					} else {
						if int(scanLinesCopied) != colorHeight {
							log.Printf("Warning GIIEX for %s: GetDIBits (HbmColor) copied %d, expected %d", exePathForDebug, scanLinesCopied, colorHeight)
						}
						// Check if this 32bpp HbmColor data has any non-zero alpha
						for i := 3; i < len(hbmColorData); i += 4 {
							if hbmColorData[i] != 0 {
								isHbmColorDataInitiallyTransparent = false
								break
							}
						}
						if isHbmColorDataInitiallyTransparent {
							log.Printf("Debug GIIEX for %s: HbmColor is 32bpp but its original alpha channel is fully transparent.", exePathForDebug)
						} else {
							log.Printf("Debug GIIEX for %s: HbmColor is 32bpp and has existing non-zero alpha values. Will prioritize this.", exePathForDebug)
						}
					}
				}
			} else {
				log.Printf("Debug GIIEX for %s: HbmColor is not 32bpp (is %d bpp). Mask will be required for transparency.", exePathForDebug, bmpColor.BmBitsPixel)
				// If HbmColorData isn't 32bpp, we'd need to convert it to 32bpp BGRA before applying a mask.
				// This is more complex. For now, if it's not 32bpp, we might need to fall back or implement BitBlt-based drawing.
				// For simplicity in this step, we'll assume if it's not 32bpp, DrawIconEx is a better bet,
				// unless we explicitly handle drawing it to a 32bpp surface and then applying the mask.
				// Let's set hbmColorData to nil to force reliance on mask or fallback.
				hbmColorData = nil 
			}
		}
	} else { log.Printf("Debug GIIEX for %s: HbmColor is nil. Mask is essential.", exePathForDebug) }


	// Decision point: Do we use HbmColor data directly (if it's 32bpp with good alpha) OR blend with mask?
	if isHbmColor32bpp && !isHbmColorDataInitiallyTransparent && hbmColorData != nil {
		// Case 1: HbmColor is 32bpp and ALREADY has good alpha. Use it directly.
		log.Printf("Debug GIIEX for %s: Using HbmColor (32bpp with existing alpha) directly.", exePathForDebug)
		rgbaOutputImg := image.NewRGBA(image.Rect(0, 0, colorWidth, colorHeight))
		idx := 0
		for y := 0; y < colorHeight; y++ {
			for x := 0; x < colorWidth; x++ {
				rgbaOutputImg.Pix[idx+0] = hbmColorData[idx+2] // R
				rgbaOutputImg.Pix[idx+1] = hbmColorData[idx+1] // G
				rgbaOutputImg.Pix[idx+2] = hbmColorData[idx+0] // B
				rgbaOutputImg.Pix[idx+3] = hbmColorData[idx+3] // A
				idx += 4
			}
		}
		return rgbaOutputImg, colorWidth, colorHeight, nil
	}

	// Case 2: HbmColor was nil, not 32bpp, or 32bpp but fully transparent.
	// We need to try blending with HbmMask if available and HbmColor data exists (even if its alpha was 0).
	// If HbmColor was nil or not 32bpp, we'd need a more complex path to even get color data
	// to blend with the mask. For this iteration, we focus on the case where HbmColor IS 32bpp
	// (so hbmColorData is populated) but was initially transparent, requiring the mask.

	var hbmMaskData []byte
	var maskBitmapDetails LoggedBitmapDetails // Declare here for scope
	// ... (Process HbmMask as in your previous working version to populate hbmMaskData and maskBitmapDetails)
	if ii.HbmMask != 0 {
		var bmpMask w32.BITMAP
		if w32.GetObject(w32.HGDIOBJ(ii.HbmMask), unsafe.Sizeof(bmpMask), unsafe.Pointer(&bmpMask)) == 0 {
			log.Printf("Error GIIEX for %s: GetObject for HbmMask failed: %v.", exePathForDebug, syscall.GetLastError())
		} else {
			maskBitmapDetails.Populate(exePathForDebug, "HbmMask", &bmpMask)
			maskBitmapDetails.Log()
			maskWidth := int(bmpMask.BmWidth)
			maskHeight := int(bmpMask.BmHeight)

			if bmpMask.BmBitsPixel == 1 && maskWidth > 0 && maskHeight > 0 {
				if maskHeight == colorHeight*2 && colorHeight > 0 {
					log.Printf("Debug GIIEX for %s: HbmMask height (%d) is 2x HbmColor height (%d). Assuming AND/XOR type mask.", exePathForDebug, maskHeight, colorHeight)
				} else if maskHeight != colorHeight && colorHeight > 0 {
					log.Printf("Warning GIIEX for %s: HbmMask height (%d) inconsistent with HbmColor height (%d).", exePathForDebug, maskHeight, colorHeight)
				}

				var biMask w32.BITMAPINFO
				hdrMask := &biMask.BmiHeader
				hdrMask.BiSize = uint32(unsafe.Sizeof(*hdrMask)); hdrMask.BiWidth = bmpMask.BmWidth
				hdrMask.BiHeight = -bmpMask.BmHeight; hdrMask.BiPlanes = 1; hdrMask.BiBitCount = 1; hdrMask.BiCompression = w32.BI_RGB
				
				memDC := w32.CreateCompatibleDC(0)
				if memDC == 0 {
					log.Printf("Error GIIEX for %s: CreateCompatibleDC for HbmMask GetDIBits failed: %v", exePathForDebug, syscall.GetLastError())
				} else {
					defer w32.DeleteDC(memDC)
					stride := ( (maskWidth + 31) &^ 31 ) / 8
					hbmMaskData = make([]byte, stride*maskHeight)
					
					scanLinesCopied := w32.GetDIBits(memDC, ii.HbmMask, 0, uint(maskHeight), unsafe.Pointer(&hbmMaskData[0]), &biMask, w32.DIB_RGB_COLORS)
					if scanLinesCopied == 0 {
						log.Printf("Error GIIEX for %s: GetDIBits on 1bpp HbmMask FAILED: %v.", exePathForDebug, syscall.GetLastError())
						hbmMaskData = nil
					} else if int(scanLinesCopied) != maskHeight {
						log.Printf("Warning GIIEX for %s: GetDIBits (HbmMask) copied %d, expected %d", exePathForDebug, scanLinesCopied, maskHeight)
					}
				}
			} else {
				log.Printf("Debug GIIEX for %s: HbmMask is not 1bpp (is %d bpp). Cannot use for alpha blending.", exePathForDebug, bmpMask.BmBitsPixel)
			}
		}
	} else { log.Printf("Debug GIIEX for %s: HbmMask is nil.", exePathForDebug) }


	// Attempt to construct final image with manual mask blending
	// This path is now taken if HbmColor was not 32bpp with good alpha, OR if it was 32bpp but fully transparent
	if hbmColorData != nil && colorWidth > 0 && colorHeight > 0 && hbmMaskData != nil && maskBitmapDetails.Width == int32(colorWidth) && (maskBitmapDetails.Height == int32(colorHeight) || maskBitmapDetails.Height == int32(colorHeight*2)) {
		// We have 32bpp color data (whose original alpha might have been all zero) and a matching mask
		log.Printf("Debug GIIEX for %s: Applying HbmMask data to HbmColor data for alpha.", exePathForDebug)
		
		rgbaOutputImg := image.NewRGBA(image.Rect(0, 0, colorWidth, colorHeight))
		maskStride := ( (colorWidth + 31) &^ 31 ) / 8 // Mask stride based on colorWidth

		idx := 0
		for y := 0; y < colorHeight; y++ {
			for x := 0; x < colorWidth; x++ {
				// Color from hbmColorData (BGRA)
				rgbaOutputImg.Pix[idx+0] = hbmColorData[idx+2] // R
				rgbaOutputImg.Pix[idx+1] = hbmColorData[idx+1] // G
				rgbaOutputImg.Pix[idx+2] = hbmColorData[idx+0] // B

				// Alpha from hbmMaskData
				maskByteIndex := y*maskStride + x/8
				if maskByteIndex < len(hbmMaskData) { // Boundary check for safety
					maskByte := hbmMaskData[maskByteIndex]
					maskBit := (maskByte >> (7 - (x % 8))) & 1
					if maskBit == 0 { // Opaque in mask
						rgbaOutputImg.Pix[idx+3] = 255
					} else { // Transparent in mask
						rgbaOutputImg.Pix[idx+3] = 0
					}
				} else { // Should not happen if dimensions match
					rgbaOutputImg.Pix[idx+3] = 0 // Default to transparent if out of bounds
				}
				idx += 4
			}
		}
		
		isFullyTransparentAfterBlend := true
		for i := 3; i < len(rgbaOutputImg.Pix); i += 4 { if rgbaOutputImg.Pix[i] != 0 { isFullyTransparentAfterBlend = false; break } }
		if isFullyTransparentAfterBlend {
			log.Printf("Warning GIIEX for %s: Image is STILL FULLY TRANSPARENT after applying HbmMask.", exePathForDebug)
		} else {
			log.Printf("Debug GIIEX for %s: Image has OPAQUE pixels after applying HbmMask.", exePathForDebug)
		}
		return rgbaOutputImg, colorWidth, colorHeight, nil
	}

	// If HbmColor was not 32bpp or its data was nil, and we couldn't blend with mask...
	// Or if HbmColor was 32bpp with good alpha but something went wrong before returning.
	// Or if blending failed due to missing data or dimension mismatch.
	log.Printf("Debug GIIEX for %s: Conditions for using HbmColor directly or blending with mask not met. Color data nil: %v. Mask data nil: %v. Color 32bpp: %v. Color initially transparent: %v",
		exePathForDebug, hbmColorData == nil, hbmMaskData == nil, isHbmColor32bpp, isHbmColorDataInitiallyTransparent)

	return nil, int(nativeWidth), int(nativeHeight), fmt.Errorf("GetIconInfoEx for %s: Unable to produce valid image from HbmColor/HbmMask. Color: %s. Mask: %s. Falling back.",
		exePathForDebug, colorBitmapDetails.Summary(), maskBitmapDetails.Summary())
}

// Helper struct and methods for logging bitmap details
type LoggedBitmapDetails struct {
	ExeName   string
	BitmapName string
	Width     int32
	Height    int32
	Planes    uint16
	BitsPixel uint16
	Valid     bool
}


// extractIconFromExe orchestrates the icon extraction.
// NEW STRATEGY:
// 1. Try DrawIconEx first (via renderIconToBGRA). This often handles complex icons well.
// 2. If DrawIconEx results in a fully transparent image, then try GetIconInfoEx + manual blending.
func extractIconFromExe(exePath string, targetSize int) (image.Image, error) {
	if targetSize <= 0 {
		targetSize = defaultIconSize
	}
	baseName := filepath.Base(exePath)

	// 1. Extract Icon Handles
	hIconLarge, hIconSmall, err := extractIconHandles(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract icon handles for %q: %w", baseName, err)
	}

	selectedIcon := hIconLarge
	if selectedIcon == 0 { selectedIcon = hIconSmall }
	if selectedIcon == 0 {
		if hIconLarge != 0 { w32.DestroyIcon(hIconLarge) }
		if hIconSmall != 0 { w32.DestroyIcon(hIconSmall) }
		return nil, ErrIconNotFound
	}
	// Cleanup unused handle
	if hIconLarge != 0 && hIconSmall != 0 {
		if selectedIcon == hIconLarge { w32.DestroyIcon(hIconSmall)
		} else { w32.DestroyIcon(hIconLarge) }
	} else if hIconLarge != 0 && selectedIcon != hIconLarge { w32.DestroyIcon(hIconLarge)
	} else if hIconSmall != 0 && selectedIcon != hIconSmall { w32.DestroyIcon(hIconSmall) }
	defer w32.DestroyIcon(selectedIcon)

	var finalImage image.Image

	// --- Attempt 1: DrawIconEx (via renderIconToBGRA) ---
	log.Printf("Debug Extractor for %s: Attempting DrawIconEx path (renderIconToBGRA).", baseName)
	bgraData, errRender := renderIconToBGRA(selectedIcon, targetSize) // Ensure this uses CreateDIBSection
	if errRender != nil {
		log.Printf("Warning Extractor for %s: renderIconToBGRA (DrawIconEx) failed: %v. Will attempt GetIconInfoEx path.", baseName, errRender)
		// Proceed to Attempt 2
	} else if bgraData == nil {
		log.Printf("Warning Extractor for %s: renderIconToBGRA (DrawIconEx) returned nil data without error. Will attempt GetIconInfoEx path.", baseName)
		// Proceed to Attempt 2
	} else {
		imgFromDrawIconEx := bgraToGoImage(bgraData, targetSize, targetSize)
		isTransparentDrawIconEx := true
		for i := 3; i < len(imgFromDrawIconEx.Pix); i += 4 {
			if imgFromDrawIconEx.Pix[i] != 0 {
				isTransparentDrawIconEx = false
				break
			}
		}

		if !isTransparentDrawIconEx {
			log.Printf("Debug Extractor for %s: DrawIconEx path successful and produced a non-transparent image.", baseName)
			finalImage = imgFromDrawIconEx
		} else {
			log.Printf("Debug Extractor for %s: DrawIconEx path produced a transparent image. Attempting GetIconInfoEx path.", baseName)
			// Proceed to Attempt 2
		}
	}

	// --- Attempt 2: GetIconInfoEx + Manual Blending (if DrawIconEx failed or produced transparent) ---
	if finalImage == nil {
		log.Printf("Debug Extractor for %s: Attempting GetIconInfoEx path.", baseName)
		// getIconImageViaGetIconInfoEx should be your version that does conditional blending (prioritizing 32bpp HbmColor if its alpha is good, else mask blend)
		imgFromGetIconInfo, nativeW, nativeH, errGetIconInfo := getIconImageViaGetIconInfoEx(baseName, selectedIcon)

		if errGetIconInfo != nil {
			log.Printf("Error Extractor for %s: GetIconInfoEx path also failed: %v. No usable icon.", baseName, errGetIconInfo)
			return nil, fmt.Errorf("both DrawIconEx and GetIconInfoEx paths failed for %q (GetIconInfoEx error: %w)", baseName, errGetIconInfo)
		} else if imgFromGetIconInfo == nil {
            log.Printf("Error Extractor for %s: GetIconInfoEx path returned nil image without error. No usable icon.", baseName)
			return nil, fmt.Errorf("GetIconInfoEx path returned nil image without error for %q", baseName)
        } else {
			// Check transparency of image from GetIconInfoEx path
			isTransparentGetIconInfoEx := true
			for i := 3; i < len(imgFromGetIconInfo.Pix); i += 4 {
				if imgFromGetIconInfo.Pix[i] != 0 {
					isTransparentGetIconInfoEx = false
					break
				}
			}
			if isTransparentGetIconInfoEx {
				log.Printf("Error Extractor for %s: GetIconInfoEx path also produced a transparent image. No usable icon.", baseName)
				return nil, fmt.Errorf("both DrawIconEx and GetIconInfoEx paths produced transparent images for %q", baseName)
			}

			// Image from GetIconInfoEx is good (non-transparent)
			log.Printf("Debug Extractor for %s: GetIconInfoEx path successful (Native: %dx%d).", baseName, nativeW, nativeH)
			if nativeW == targetSize && nativeH == targetSize {
				finalImage = imgFromGetIconInfo
			} else if nativeW > 0 && nativeH > 0 {
				resizedImg := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
				draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), imgFromGetIconInfo, imgFromGetIconInfo.Bounds(), draw.Over, nil)
				finalImage = resizedImg
			} else {
				log.Printf("Error Extractor for %s: GetIconInfoEx path returned 0 dimensions (%dx%d). No usable icon.", baseName, nativeW, nativeH)
				return nil, fmt.Errorf("GetIconInfoEx path returned 0 dimensions for %q", baseName)
			}
		}
	}

	if finalImage == nil {
		// This means DrawIconEx path failed/transparent AND GetIconInfoEx path failed/transparent
		return nil, fmt.Errorf("failed to extract a non-transparent icon for %q after all attempts", baseName)
	}

	return finalImage, nil
}

// --- Icon Extraction and Saving Logic (Minor Refinements) ---

// extractAndSaveIcon checks existence, extracts, and saves a single icon.
// Returns nil on success, ErrIconSkipped, or ErrIconNotFound.
// Returns specific errors on failure (filesystem, extraction, encoding).
func extractAndSaveIcon(exePath string, storageDir string) error {
	targetPngPath := filepath.Join(storageDir, generateIconFilename(exePath))

	// 1. Check if icon already exists
	if _, err := os.Stat(targetPngPath); err == nil {
		return ErrIconSkipped // Already exists, skip processing
	} else if !errors.Is(err, os.ErrNotExist) {
		// Filesystem error other than "not found"
		log.Printf("Warning: Error checking icon file %s: %v\n", targetPngPath, err)
		return fmt.Errorf("failed to check icon file status for %q: %w", targetPngPath, err)
	}

	// 2. Extract the icon image
	extractedIcon, extractErr := extractIconFromExe(exePath, defaultIconSize)
	if extractErr != nil {
		if errors.Is(extractErr, ErrIconNotFound) {
			// Not a failure, just no icon present in the exe.
			// log.Printf("Debug: No icon found in %s", filepath.Base(exePath))
			return nil // Handled case, return nil to signify no action needed/failed
		}
		// Actual extraction error (permissions, API failure, rendering error, etc.)
		// The error from platformExtractIcon already includes context.
		return extractErr // Return the wrapped error directly
	}
	// Defensive check - should not happen if platformExtractIcon works correctly
	if extractedIcon == nil {
		return fmt.Errorf("internal error: platformExtractIcon returned nil image and nil error for %q", exePath)
	}

	// 3. Save the extracted icon as PNG
	outFile, err := os.Create(targetPngPath)
	if err != nil {
		log.Printf("Error: Failed to create icon file %s: %v\n", targetPngPath, err)
		return fmt.Errorf("failed to create icon file %q: %w", targetPngPath, err)
	}
	defer outFile.Close() // Ensure file is closed

	err = png.Encode(outFile, extractedIcon)
	if err != nil {
		// Best effort to clean up potentially partial/corrupt file on encode error
		outFile.Close()              // Close handle before removing
		_ = os.Remove(targetPngPath) // Ignore error during cleanup attempt
		return fmt.Errorf("failed to encode icon to PNG %q: %w", targetPngPath, err)
	}

	return nil // Success!
}

// --- Main Orchestration Function (Refactored for Clarity) ---

// generateSummaryReport creates a formatted string summarizing the extraction results.
func generateSummaryReport(total, skipped, failures uint64, skippedNames, failedNames []string) string {
	var report strings.Builder
	reportSeparator := "\n---\n"

	// Sort names for consistent output
	sort.Strings(skippedNames)
	sort.Strings(failedNames)

	if skipped > 0 {
		report.WriteString(fmt.Sprintf("Icons skipped (already exist) (%d):", skipped))
		for _, name := range skippedNames {
			report.WriteString(fmt.Sprintf("\n  - %s", name))
		}
		if failures > 0 || total > 0 { // Add separator if there's more to report
			report.WriteString(reportSeparator)
		}
	}

	if failures > 0 {
		report.WriteString(fmt.Sprintf("Executables with extraction/saving failures (%d):", failures))
		for _, name := range failedNames {
			report.WriteString(fmt.Sprintf("\n  - %s", name))
		}
		if skipped > 0 || total > 0 { // Add separator if there's more to report
			report.WriteString(reportSeparator)
		}
	}

	// Add summary line
	report.WriteString(fmt.Sprintf("Icon extraction complete. Processed: %d .exe files, Skipped: %d, Actual Failures: %d",
		total, skipped, failures))

	return report.String()
}

// ExtractAndSaveIcons concurrently processes a map of applications to extract and save icons.
func ExtractAndSaveIcons(appMap map[string]AppLaunchInfo) error {
	if len(appMap) == 0 {
		log.Println("App map is empty, skipping icon extraction.")
		return nil
	}

	// Critical: Ensure storage directory exists first.
	iconStorageDir, err := getIconStorageDir()
	if err != nil {
		log.Printf("CRITICAL: Failed to get/create icon storage directory: %v. Icon extraction disabled.", err)
		return err // Cannot proceed without storage.
	}

	log.Printf("Starting icon extraction for up to %d applications to %s...", len(appMap), iconStorageDir)

	var wg sync.WaitGroup
	var totalAttempted atomic.Uint64
	var failureCount atomic.Uint64
	var skippedCount atomic.Uint64

	// Slices for reporting (protected by mutex)
	var failedExeNames []string
	var skippedExeNames []string
	var reportMutex sync.Mutex // Renamed mutex for clarity

	for exePath := range appMap {
		if !strings.HasSuffix(strings.ToLower(exePath), ".exe") {
			continue // Skip non-exe files
		}

		totalAttempted.Add(1)
		wg.Add(1)

		go func(p string) { // Pass path as argument
			defer wg.Done()

			err := extractAndSaveIcon(p, iconStorageDir)

			if err != nil {
				baseName := filepath.Base(p)
				if errors.Is(err, ErrIconSkipped) {
					skippedCount.Add(1)
					reportMutex.Lock()
					skippedExeNames = append(skippedExeNames, baseName)
					reportMutex.Unlock()
				} else {
					// This is a real failure (not skipped, not icon-not-found)
					failureCount.Add(1)
					reportMutex.Lock()
					failedExeNames = append(failedExeNames, baseName)
					reportMutex.Unlock()

					// Conditionally log individual errors - avoid flooding with common OS errors
					// from extraction if they were already wrapped (e.g., access denied),
					// but log other errors (file create, encode).
					// We check if the error *is* a syscall.Errno OR if it *wraps* one
					// originating from platformExtractIcon/renderIconToBGRA/extractIconHandles.

					log.Printf("ERROR_DETAILS: Failed to process icon for %s (Path: %s): %v", baseName, p, err)

					// var osErr syscall.Errno
					// isOsExtractionError := errors.As(err, &osErr)

					// if !isOsExtractionError {
					// 	log.Printf("Error processing icon for %s: %v", baseName, err)
					// }
					// Note: Even if not logged individually here, OS errors are counted
					// and listed in the final summary report.
				}
			}
			// If err is nil, it means success OR ErrIconNotFound (which is handled)
		}(exePath)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Generate and log the summary report
	finalReport := generateSummaryReport(
		totalAttempted.Load(),
		skippedCount.Load(),
		failureCount.Load(),
		skippedExeNames, // Pass slice copies implicitly
		failedExeNames,
	)
	log.Println(finalReport)

	return nil // Only setup errors are returned from this function
}

// CleanOrphanedIcons scans the icon storage directory and removes any icon files
// that do not correspond to an executable path currently listed in the provided appMap.
// It logs the number of icons checked and deleted.
//
// Args:
//
//	appMap (map[string]FinalAppOutput): The map of currently known/discovered applications,
//	                                    where keys are the full executable paths.
//
// Returns:
//
//	error: An error if the icon storage directory cannot be accessed or read,
//	       otherwise nil (even if individual file deletions fail, which are logged).
func CleanOrphanedIcons(appMap map[string]AppLaunchInfo) error {
	log.Println("Starting cleanup of orphaned icons...")

	// 1. Get the icon storage directory path.
	iconStorageDir, err := getIconStorageDir()
	if err != nil {
		// This is critical, can't proceed without the directory.
		return fmt.Errorf("failed to get icon storage directory for cleanup: %w", err)
	}

	// 2. Build a set of *expected* icon filenames based on the current appMap.
	// Using a map[string]struct{} acts as a lightweight set for efficient lookups.
	expectedIconFiles := make(map[string]struct{}, len(appMap))
	for exePath := range appMap {
		// Only consider .exe files for generating expected icon names, mirroring extraction logic.
		if strings.HasSuffix(strings.ToLower(exePath), ".exe") {
			expectedFilename := generateIconFilename(exePath)
			expectedIconFiles[expectedFilename] = struct{}{} // Add filename to the set
		}
	}
	log.Printf("Expecting icons for %d known executables.", len(expectedIconFiles))

	// 3. Read the contents of the actual icon storage directory.
	dirEntries, err := os.ReadDir(iconStorageDir)
	if err != nil {
		// If we can't read the directory, we can't clean it.
		return fmt.Errorf("failed to read icon storage directory '%s': %w", iconStorageDir, err)
	}

	// 4. Iterate through actual files and delete orphans.
	foundCount := 0
	deletedCount := 0
	for _, entry := range dirEntries {
		// Skip subdirectories, focus on files.
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Optional: Only consider .png files if you might have other file types there.
		// if !strings.HasSuffix(strings.ToLower(filename), ".png") {
		//  continue
		// }
		foundCount++

		// Check if this filename exists in our set of expected icons.
		if _, expected := expectedIconFiles[filename]; !expected {
			// This icon file is not associated with any current app in the map. It's an orphan.
			orphanPath := filepath.Join(iconStorageDir, filename)
			log.Printf("Deleting orphaned icon: %s", filename)
			err := os.Remove(orphanPath)
			if err != nil {
				// Log deletion errors but continue the process for other files.
				log.Printf("Warning: Failed to delete orphaned icon '%s': %v", orphanPath, err)
				// You might want to add a counter for failed deletions if needed.
			} else {
				deletedCount++
			}
		}
	}

	log.Printf("Orphaned icon cleanup complete. Checked: %d files, Deleted: %d orphans.", foundCount, deletedCount)
	return nil // Return nil as the primary operation (reading dir) succeeded.
}

// Example of how you might call it (e.g., during startup or periodically):
func RunOrphanedIconsCleanup() {
	if len(discoveredApps) > 0 { // Or some other condition to trigger cleanup
		err := CleanOrphanedIcons(discoveredApps)
		if err != nil {
			log.Printf("Error during icon cleanup process: %v", err)
		}
	} else {
		log.Println("Skipping icon cleanup as the application map is empty.")
	}
}

// --- Helper Function to Get Icon Path (Unchanged) ---
func GetIconPathForExe(exePath string) (string, error) {
	if exePath == "" {
		return "", nil
	}

	// Generate the expected icon filename
	iconFilename := generateIconFilename(exePath)

	// Check if the icon file actually exists
	iconDir, err := getIconStorageDir()
	if err != nil {
		return "", err
	}

	// Check if the icon file exists
	fullPath := filepath.Join(iconDir, iconFilename)
	if _, err := os.Stat(fullPath); err != nil {
		return "", nil // Icon doesn't exist, return empty string
	}

	// Icon exists, return the web path
	return path.Join(WebIconPath, iconFilename), nil
}

func ProcessIcons() {

	if len(discoveredApps) == 0 {
		log.Println("No discovered apps to process icons for.")
		return
	}

	log.Println("Starting background icon extraction...")
	go func() {
		err := ExtractAndSaveIcons(discoveredApps)
		if err != nil {
			log.Printf("Icon extraction failed critically during setup: %v", err)
		} else {
			log.Println("Background icon extraction goroutine finished.")
		}
	}()

	RunOrphanedIconsCleanup()
}
