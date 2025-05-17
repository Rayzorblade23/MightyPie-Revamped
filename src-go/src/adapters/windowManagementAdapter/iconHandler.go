package windowManagementAdapter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// --- Custom Errors ---
var (
	ErrIconSkipped      = errors.New("icon skipped, already exists")
	ErrIconNotFound     = errors.New("icon not found in executable")
	ErrIconNotProcessed = errors.New("icon could not be processed by any method")
)

// --- Shared State (Icon Directory) ---
var (
	iconDirOnce sync.Once
	iconDirPath string
	iconDirErr  error
)

// --- Icon Storage & Path Management ---

// getIconStorageDir finds or creates the directory for storing icons.
func getIconStorageDir() (string, error) {
	iconDirOnce.Do(func() {
		// Use the provided getRootDir function
		rootDir, err := getRootDir()
		if err != nil {
			iconDirErr = fmt.Errorf("failed to determine project root using getRootDir: %w", err)
			return
		}

		dir := filepath.Join(rootDir, appDataIconSubdir)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			iconDirErr = fmt.Errorf("failed to create icon directory '%s': %w", dir, err)
			return
		}
		iconDirPath = dir
	})
	return iconDirPath, iconDirErr
}

// getRootDir returns the project root directory. (User's original version was kept as it was stated to be correct for their setup)
func getRootDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Dir(dir), nil
		}
		if filepath.Base(dir) == "src-go" {
			return filepath.Dir(filepath.Dir(dir)), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root directory from %s", wd)
		}
		dir = parent
	}
}

// Updated to include space replacement
var iconFilenameReplacer = strings.NewReplacer(
	" ", "_",
	":", "_",
	`\`, "_",
	`/`, "_",
	`*`, "_",
	`?`, "_",
	`"`, "_",
	`<`, "_",
	`>`, "_",
	`|`, "_",
)

func generateIconFilename(exePath string) string {
	baseName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	safeBaseName := iconFilenameReplacer.Replace(baseName)
	if len(safeBaseName) > maxIconBaseNameLength {
		safeBaseName = safeBaseName[:maxIconBaseNameLength]
	}
	hasher := sha256.New()
	hasher.Write([]byte(exePath))
	hashString := hex.EncodeToString(hasher.Sum(nil)[:iconHashLength])
	return fmt.Sprintf("%s_%s.png", safeBaseName, hashString)
}

// --- File Saving & Orchestration ---

// extractAndSaveIcon checks existence, extracts, and saves a single icon.
func extractAndSaveIcon(exePath string, storageDir string) error {
	targetPngPath := filepath.Join(storageDir, generateIconFilename(exePath))

	if _, err := os.Stat(targetPngPath); err == nil {
		return ErrIconSkipped
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking icon file %s: %w", targetPngPath, err)
	}

	extractedIcon, extractErr := extractIconFromExe(exePath, defaultIconSize)
	if extractErr != nil {
		if errors.Is(extractErr, ErrIconNotFound) || errors.Is(extractErr, ErrIconNotProcessed) {
			// Log these as debug/info, not necessarily hard errors for the whole batch
			// log.Printf("Info: No usable icon for %s: %v", filepath.Base(exePath), extractErr)
			return nil // Treat as skippable for batch processing
		}
		return extractErr // Actual error during extraction
	}
	if extractedIcon == nil { // Should be caught by extractErr usually
		return fmt.Errorf("internal: extractIconFromExe nil image without error for %q", exePath)
	}

	outFile, err := os.Create(targetPngPath)
	if err != nil {
		return fmt.Errorf("creating icon file %q: %w", targetPngPath, err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, extractedIcon); err != nil {
		_ = os.Remove(targetPngPath) // Best effort cleanup
		return fmt.Errorf("encoding icon to PNG %q: %w", targetPngPath, err)
	}
	return nil
}

// generateSummaryReport creates a formatted string summarizing the extraction results.
func generateSummaryReport(total, skipped, failures uint64, skippedNames, failedNames []string) string {
	var report strings.Builder
	reportSeparator := "\n---\n"
	sort.Strings(skippedNames)
	sort.Strings(failedNames)

	if skipped > 0 {
		report.WriteString(fmt.Sprintf("Icons skipped (already exist) (%d):", skipped))
		for _, name := range skippedNames {
			report.WriteString(fmt.Sprintf("\n  - %s", name))
		}
		if failures > 0 || total > 0 {
			report.WriteString(reportSeparator)
		}
	}
	if failures > 0 {
		report.WriteString(fmt.Sprintf("Executables with extraction/saving failures (%d):", failures))
		for _, name := range failedNames {
			report.WriteString(fmt.Sprintf("\n  - %s", name))
		}
		// No separator needed after the last section
	}
	report.WriteString(reportSeparator) // Separator before final summary line
	report.WriteString(fmt.Sprintf("Icon extraction complete. Processed: %d, Skipped: %d, Failures: %d",
		total, skipped, failures))
	return report.String()
}

// ExtractAndSaveIcons concurrently processes a map of applications to extract and save their icons,
// updating appMap with the new icon paths.
func ExtractAndSaveIcons(appMap map[string]core.AppInfo) error {
	if len(appMap) == 0 {
		log.Println("App map is empty, icon extraction skipped.")
		return nil
	}
	iconStorageDir, err := getIconStorageDir() // Assume defined
	if err != nil {
		return fmt.Errorf("icon storage directory unavailable: %w", err)
	}
	// Log statement slightly changed from original to reflect map size not attempts yet.
	log.Printf("Processing up to %d apps for icon extraction to %s...", len(appMap), iconStorageDir)

	var wg sync.WaitGroup
	// Counters align with generateSummaryReport:
	// totalAttempted: Goroutines launched for an extraction attempt.
	// failureCount: Hard errors or unprocessable states.
	// skippedCount: Icon already known (in appMap.IconPath) or found on disk by extractAndSaveIcon (ErrIconSkipped).
	var totalAttempted, failureCount, skippedCount atomic.Uint64

	var reportMutex sync.Mutex // Guards appMap writes AND the report slices
	failedExeNames := make([]string, 0)
	skippedExeNames := make([]string, 0)

	for appNameKey, appInfo := range appMap {
		extractionPath := appInfo.ExePath
		// Determine a representative name for reporting this entry
		reportName := filepath.Base(extractionPath)
		if reportName == "." || reportName == "" || extractionPath == "" { // Handle empty or unusual paths
			reportName = appNameKey // Fallback to appNameKey if path isn't useful
		}

		// --- PRE-CHECKS before launching goroutine ---

		// 1. IconPath already set in appMap: Considered "skipped (already exist)".
		if appInfo.IconPath != "" {
			skippedCount.Add(1)
			reportMutex.Lock()
			skippedExeNames = append(skippedExeNames, reportName)
			reportMutex.Unlock()
			continue
		}

		// 2. No ExePath for extraction: Cannot attempt. Considered a "failure" for reporting.
		if extractionPath == "" {
			failureCount.Add(1)
			reportMutex.Lock()
			failedExeNames = append(failedExeNames, fmt.Sprintf("%s (no ExePath)", appNameKey)) // Use appNameKey as reportName is also appNameKey here
			reportMutex.Unlock()
			continue
		}

		// If we reach here, an extraction attempt will be made via goroutine.
		totalAttempted.Add(1)
		wg.Add(1)

		go func(currentAppName string, currentExtractionPath string, currentReportName string) {
			defer wg.Done()

			errExtract := extractAndSaveIcon(currentExtractionPath, iconStorageDir)

			// We *always* need to construct the potential disk/web path to check/use it
			expectedDiskFilename := generateIconFilename(currentExtractionPath)
			targetPngPathOnDisk := filepath.Join(iconStorageDir, expectedDiskFilename)
			potentialWebIconPath := path.Join(webIconPathPrefix, expectedDiskFilename)

			successfullyProcessedAndIconAvailable := false

			if errExtract == nil {
				// extractAndSaveIcon returned nil. Check if the file was actually created/exists.
				if _, statErr := os.Stat(targetPngPathOnDisk); statErr == nil {
					// File exists! Successful extraction.
					successfullyProcessedAndIconAvailable = true
					// log.Printf("DEBUG: App %s, Exe %s: extractAndSaveIcon nil error, file found at %s", currentAppName, currentExtractionPath, targetPngPathOnDisk)
				} else {
					// File does not exist. extractAndSaveIcon returned nil due to internal skip (e.g., ErrIconNotFound).
					// Not a failure for reporting, but no icon path to update.
					// log.Printf("DEBUG: App %s, Exe %s: extractAndSaveIcon nil error, but file NOT found at %s (internal skip)", currentAppName, currentExtractionPath, targetPngPathOnDisk)
				}
			} else if errors.Is(errExtract, ErrIconSkipped) {
				// extractAndSaveIcon reported ErrIconSkipped, meaning file was already there.
				skippedCount.Add(1)
				reportMutex.Lock()
				skippedExeNames = append(skippedExeNames, currentReportName)
				reportMutex.Unlock()
				successfullyProcessedAndIconAvailable = true // Icon is available
				// log.Printf("DEBUG: App %s, Exe %s: ErrIconSkipped, icon assumed at %s", currentAppName, currentExtractionPath, targetPngPathOnDisk)
			} else {
				// Hard failure
				failureCount.Add(1)
				reportMutex.Lock()
				failedExeNames = append(failedExeNames, currentReportName)
				reportMutex.Unlock()
				// log.Printf("DEBUG: App %s, Exe %s: Hard failure: %v", currentAppName, currentExtractionPath, errExtract)
			}

			if successfullyProcessedAndIconAvailable {
				reportMutex.Lock()
				if entryToUpdate, ok := appMap[currentAppName]; ok {
					if entryToUpdate.IconPath != potentialWebIconPath { // Update if different or was empty
						// log.Printf("DEBUG: UPDATING App %s: Old IconPath: '%s', New IconPath: '%s'", currentAppName, entryToUpdate.IconPath, potentialWebIconPath)
						entryToUpdate.IconPath = potentialWebIconPath
						appMap[currentAppName] = entryToUpdate // Put the modified copy back
					} else {
						// log.Printf("DEBUG: App %s: IconPath already correctly set to '%s', no update needed.", currentAppName, potentialWebIconPath)
					}
				} else {
					log.Printf("CRITICAL (DEBUG): App '%s' not found in appMap for update. Desired IconPath: '%s'", currentAppName, potentialWebIconPath)
				}
				reportMutex.Unlock()
			}
		}(appNameKey, extractionPath, reportName)
	}

	wg.Wait()

	// Use the original generateSummaryReport function with the populated counters and slices.
	// log.Println(generateSummaryReport(totalAttempted.Load(), skippedCount.Load(), failureCount.Load(), skippedExeNames, failedExeNames))
	return nil
}

// CleanOrphanedIcons removes icon files from the storage directory that are not
// referenced by any application in the provided appMap.
func CleanOrphanedIcons(appMap map[string]core.AppInfo) error {
	log.Println("Starting orphaned icon cleanup...")
	iconStorageDir, err := getIconStorageDir() // Assume getIconStorageDir is defined
	if err != nil {
		return fmt.Errorf("could not get icon storage directory for cleanup: %w", err)
	}

	// Populate a set of expected icon filenames from the appMap.
	expectedIconFiles := make(map[string]struct{}, len(appMap))
	for _, appInfo := range appMap { // Iterate over the values (AppInfo structs)
		if appInfo.IconPath != "" {
			// appInfo.IconPath is a web-servable path like "/icons/actual_icon.png".
			// We need the base filename, e.g., "actual_icon.png".
			// Use path.Base for slash-separated paths.
			iconFilename := path.Base(appInfo.IconPath)

			// path.Base returns "." for an empty path or ".." for paths ending in "..".
			// It returns "/" for "/". Ensure we only add actual filenames.
			if iconFilename != "" && iconFilename != "." && iconFilename != "/" {
				expectedIconFiles[iconFilename] = struct{}{}
			}
		}
	}
	log.Printf("Expecting %d unique icon files based on current application map.", len(expectedIconFiles))

	// Read the contents of the icon storage directory.
	dirEntries, err := os.ReadDir(iconStorageDir)
	if err != nil {
		// If the directory doesn't exist, there's nothing to clean.
		if os.IsNotExist(err) {
			log.Printf("Icon storage directory '%s' does not exist. No cleanup needed.", iconStorageDir)
			return nil
		}
		return fmt.Errorf("failed to read icon storage directory '%s': %w", iconStorageDir, err)
	}

	// Iterate through files in the storage directory and delete orphans.
	actualFilesChecked, iconsDeleted := 0, 0
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		filenameOnDisk := entry.Name()

		// Process only files with the expected icon extension (e.g., .png).
		// Adjust this if you support other icon formats.
		if !strings.HasSuffix(strings.ToLower(filenameOnDisk), ".png") {
			continue
		}
		actualFilesChecked++

		if _, isExpected := expectedIconFiles[filenameOnDisk]; !isExpected {
			orphanPath := filepath.Join(iconStorageDir, filenameOnDisk)
			// log.Printf("Deleting orphaned icon: %s", filenameOnDisk) // More verbose logging if needed
			if err := os.Remove(orphanPath); err != nil {
				// Log the error but continue trying to clean other files.
				log.Printf("Warning: Failed to delete orphaned icon '%s': %v", orphanPath, err)
			} else {
				iconsDeleted++
			}
		}
	}

	log.Printf("Orphaned icon cleanup complete. Checked %d relevant files on disk, deleted %d orphaned icons.", actualFilesChecked, iconsDeleted)
	return nil
}

// GetIconPathForExe returns the web-servable path for an executable's icon.
func GetIconPathForExe(exePath string) (string, error) {
	if exePath == "" {
		return "", nil
	}
	iconFilename := generateIconFilename(exePath)
	iconDir, err := getIconStorageDir()
	if err != nil {
		return "", err
	} // Propagate error

	fullPath := filepath.Join(iconDir, iconFilename)
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		} // Icon doesn't exist, no error, just no path
		return "", err // Filesystem error
	}
	return path.Join(webIconPathPrefix, iconFilename), nil
}

// ProcessIcons orchestrates icon extraction and cleanup.
func ProcessIcons() {
	if installedAppsInfo == nil { // Defensive check for nil map
		log.Println("No discovered apps for icon processing.")
		return
	}
	log.Println("Starting background icon processing...")
	go func() {
		if err := ExtractAndSaveIcons(installedAppsInfo); err != nil {
			log.Printf("CRITICAL: Icon extraction process failed: %v", err)
		} else {
			log.Println("Background icon extraction finished.")
		}
		if err := CleanOrphanedIcons(installedAppsInfo); err != nil {
			log.Printf("Error during icon cleanup: %v", err)
		} else {
			log.Println("Icon cleanup finished.")
		}
	}()
}
