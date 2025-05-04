package windowManagementAdapter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
)

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

// renderIconToBGRA draws the icon onto a compatible bitmap and returns its raw BGRA pixel data.
// It handles GDI resource creation and cleanup internally.
func renderIconToBGRA(hIcon w32.HICON, size int) ([]byte, error) {
	// 1. Get Screen DC (needed for compatibility)
	screenDC := w32.GetDC(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC(0) failed: %v", syscall.GetLastError())
	}
	defer w32.ReleaseDC(0, screenDC)

	// 2. Create Compatible DC
	memDC := w32.CreateCompatibleDC(screenDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed: %v", syscall.GetLastError())
	}
	defer w32.DeleteDC(memDC)

	// 3. Create Compatible Bitmap
	bitmap := w32.CreateCompatibleBitmap(screenDC, size, size)
	if bitmap == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed: %v", syscall.GetLastError())
	}
	defer w32.DeleteObject(w32.HGDIOBJ(bitmap))

	// 4. Select Bitmap into DC
	oldBitmap := w32.SelectObject(memDC, w32.HGDIOBJ(bitmap))
	if oldBitmap == 0 {
		return nil, errors.New("SelectObject failed") // GetLastError often not set here
	}
	defer w32.SelectObject(memDC, oldBitmap) // Select back before deleting DC

	// 5. Draw the Icon
	success := w32.DrawIconEx(memDC, 0, 0, hIcon, size, size, 0, 0, diNormal)
	if !success {
		return nil, fmt.Errorf("DrawIconEx failed: %w", syscall.GetLastError())
	}

	// 6. Prepare to get pixel data (BITMAPINFO)
	var bmi w32.BITMAPINFO
	hdr := &bmi.BmiHeader
	hdr.BiSize = uint32(unsafe.Sizeof(*hdr))
	hdr.BiWidth = int32(size)
	hdr.BiHeight = int32(-size) // Negative height for top-down DIB
	hdr.BiPlanes = 1
	hdr.BiBitCount = 32 // 32 bits per pixel (BGRA)
	hdr.BiCompression = w32.BI_RGB

	// 7. Allocate buffer for pixel data
	pixelDataSize := size * size * 4 // BGRA format
	pixelData := make([]byte, pixelDataSize)

	// 8. Get Pixel Data using GetDIBits
	scanLinesCopied := w32.GetDIBits(memDC, bitmap, 0, uint(size), unsafe.Pointer(&pixelData[0]), &bmi, w32.DIB_RGB_COLORS)
	if scanLinesCopied == 0 {
		return nil, fmt.Errorf("GetDIBits failed: %w", syscall.GetLastError())
	}
	if int(scanLinesCopied) != size {
		log.Printf("Warning: GetDIBits copied %d scanlines, expected %d", scanLinesCopied, size)
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

// extractIconFromExe orchestrates the icon extraction using helper functions.
func extractIconFromExe(exePath string, size int) (image.Image, error) {
	if size <= 0 {
		size = defaultIconSize
	}

	// 1. Extract Icon Handles
	hIconLarge, hIconSmall, err := extractIconHandles(exePath)
	if err != nil {
		// Handle ErrIconNotFound specifically, wrap other errors
		if errors.Is(err, ErrIconNotFound) {
			return nil, ErrIconNotFound
		}
		return nil, fmt.Errorf("failed to extract icon handles for %q: %w", exePath, err)
	}

	// 2. Select Best Handle and Manage Cleanup
	selectedIcon := hIconLarge
	if selectedIcon == 0 {
		selectedIcon = hIconSmall
	}

	// Clean up the *unused* handle immediately if both were valid.
	// The selected handle cleanup is deferred.
	if hIconLarge != 0 && hIconSmall != 0 {
		if selectedIcon == hIconLarge {
			w32.DestroyIcon(hIconSmall)
		} else { // selectedIcon == hIconSmall
			w32.DestroyIcon(hIconLarge)
		}
	}
	// Ensure the selected icon handle is eventually destroyed.
	defer w32.DestroyIcon(selectedIcon)

	// 3. Render Icon to BGRA Pixel Buffer
	bgraData, err := renderIconToBGRA(selectedIcon, size)
	if err != nil {
		return nil, fmt.Errorf("failed to render icon to bitmap for %q: %w", exePath, err)
	}

	// 4. Convert BGRA data to Go image.Image (RGBA)
	img := bgraToGoImage(bgraData, size, size)

	return img, nil
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
					var osErr syscall.Errno
					isOsExtractionError := errors.As(err, &osErr)

					if !isOsExtractionError {
						log.Printf("Error processing icon for %s: %v", baseName, err)
					}
					// Note: Even if not logged individually here, OS errors are counted
					// and listed in the final summary report.
				}
			}
			// If err is nil, it means success OR ErrIconNotFound (which is handled)
		}(exePath)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Generate and log the summary report
	// finalReport := generateSummaryReport(
	// 	totalAttempted.Load(),
	// 	skippedCount.Load(),
	// 	failureCount.Load(),
	// 	skippedExeNames, // Pass slice copies implicitly
	// 	failedExeNames,
	// )
	// log.Println(finalReport)

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
        return "", nil  // Icon doesn't exist, return empty string
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
