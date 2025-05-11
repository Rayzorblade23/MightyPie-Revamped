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

var iconFilenameReplacer = strings.NewReplacer(":", "_", `\`, "_", `/`, "_", `*`, "_", `?`, "_", `"`, "_", `<`, "_", `>`, "_", `|`, "_")

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

// ExtractAndSaveIcons concurrently processes a map of applications.
func ExtractAndSaveIcons(appMap map[string]AppLaunchInfo) error {
	if len(appMap) == 0 {
		log.Println("App map is empty, icon extraction skipped.")
		return nil
	}
	iconStorageDir, err := getIconStorageDir()
	if err != nil {
		return fmt.Errorf("icon storage directory unavailable: %w", err) // Critical setup error
	}
	log.Printf("Starting icon extraction for %d apps to %s...", len(appMap), iconStorageDir)

	var wg sync.WaitGroup
	var totalAttempted, failureCount, skippedCount atomic.Uint64
	failedExeNames := make([]string, 0) // Use channels for concurrent-safe appends or mutex
	skippedExeNames := make([]string, 0)
	var reportMutex sync.Mutex

	for exePath := range appMap {
		if !strings.HasSuffix(strings.ToLower(exePath), ".exe") {
			// log.Printf("Skipping non-exe: %s", exePath)
			continue
		}
		totalAttempted.Add(1)
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			err := extractAndSaveIcon(p, iconStorageDir)
			baseName := filepath.Base(p)
			if err != nil {
				if errors.Is(err, ErrIconSkipped) {
					skippedCount.Add(1)
					reportMutex.Lock()
					skippedExeNames = append(skippedExeNames, baseName)
					reportMutex.Unlock()
				} else {
					failureCount.Add(1)
					reportMutex.Lock()
					failedExeNames = append(failedExeNames, baseName)
					reportMutex.Unlock()
				}
			}
		}(exePath)
	}
	wg.Wait()
	// log.Println(generateSummaryReport(totalAttempted.Load(), skippedCount.Load(), failureCount.Load(), skippedExeNames, failedExeNames))
	return nil
}

// CleanOrphanedIcons removes icons not in the current appMap.
func CleanOrphanedIcons(appMap map[string]AppLaunchInfo) error {
	log.Println("Starting orphaned icon cleanup...")
	iconStorageDir, err := getIconStorageDir()
	if err != nil {
		return fmt.Errorf("icon storage directory for cleanup: %w", err)
	}

	expectedIconFiles := make(map[string]struct{}, len(appMap))
	for exePath := range appMap {
		if strings.HasSuffix(strings.ToLower(exePath), ".exe") {
			expectedIconFiles[generateIconFilename(exePath)] = struct{}{}
		}
	}
	// log.Printf("Expecting %d icons.", len(expectedIconFiles))

	dirEntries, err := os.ReadDir(iconStorageDir)
	if err != nil {
		return fmt.Errorf("reading icon storage dir '%s': %w", iconStorageDir, err)
	}

	foundCount, deletedCount := 0, 0
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".png") {
			continue
		} // Process only .png files
		foundCount++
		if _, expected := expectedIconFiles[filename]; !expected {
			orphanPath := filepath.Join(iconStorageDir, filename)
			// log.Printf("Deleting orphaned icon: %s", filename)
			if err := os.Remove(orphanPath); err != nil {
				log.Printf("Warning: Failed to delete orphaned icon '%s': %v", orphanPath, err)
			} else {
				deletedCount++
			}
		}
	}
	log.Printf("Orphaned icon cleanup: Checked %d, Deleted %d.", foundCount, deletedCount)
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
	if discoveredApps == nil { // Defensive check for nil map
		log.Println("No discovered apps for icon processing.")
		return
	}
	log.Println("Starting background icon processing...")
	go func() {
		if err := ExtractAndSaveIcons(discoveredApps); err != nil {
			log.Printf("CRITICAL: Icon extraction process failed: %v", err)
		} else {
			log.Println("Background icon extraction finished.")
		}
		if err := CleanOrphanedIcons(discoveredApps); err != nil {
			log.Printf("Error during icon cleanup: %v", err)
		} else {
			log.Println("Icon cleanup finished.")
		}
	}()
}
