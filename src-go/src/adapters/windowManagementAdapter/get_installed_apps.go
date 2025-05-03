package windowManagementAdapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	lnk "github.com/parsiya/golnk"
)

// --- Constants ---

const (
	getUwpLocationsPsCommand = `Get-AppxPackage | Where-Object {$_.InstallLocation} | Select-Object -Property PackageFamilyName, InstallLocation | ConvertTo-Json -Depth 2 -Compress`
	getStartAppsPsCommand    = `Get-StartApps | Select-Object Name, AppID | Where-Object { $_.AppID -ne $null -and $_.AppID -ne '' -and $_.AppID -notlike '*SystemSettings*' -and $_.AppID -notlike '*Search*' } | ForEach-Object { ($_.Name -replace '\t',' ') + "` + "\t" + `" + $_.AppID }`
)

// Filtering lists
var (
	unwantedKeywords = []string{
		"uninstall", "uninst", "remove", "setup", "install", "update", "updater", "patch", "config",
		"configure", "report", "crash", "debug", "eula", "readme", "license", "help", "support",
		"wizard", "register", "activate", "bootstrapper", "dotnet", "vcredist", "redist", "driver",
		"service", "agent", "sync", "verifier", "manual", "documentation", "docs", "guide",
		"keymap", "shortcuts", "website", "homepage", "link", "url", "example", "demo",
		"appvlp", "hxtsr", "searchhost", "createdump", "apphost", "dllhost", "migration",
		"devhome", "gamebar", "hxcalendarappimm", "hxoutlook",
	}
	nonExecExtensions = []string{
		".txt", ".pdf", ".html", ".htm", ".url", ".lnk", ".log", ".ini", ".xml", ".chm", ".msi",
		".msp", ".msu", ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".zip", ".rar", ".7z",
		".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".wav", ".mp3", ".mp4", ".avi",
		".mov", ".inf", ".sys", ".dll", ".md",
	}
	whitelistKeywords = []string{"sleepington"} // Example: Keep "Sleepington Updater"

	systemApps = map[string]string{
		"explorer.exe": "Windows Explorer",
		"taskmgr.exe":  "Task Manager",
		"cmd.exe":      "Command Prompt",
	}
)

// --- Structs ---

type AppEntry struct {
	Name string
	Path string // Resolved executable path
}

// FinalAppOutput defines the structure of the VALUE in the final JSON map sent to stdout.
type FinalAppOutput struct {
	Name             string `json:"name"`                       // The original display name
	WorkingDirectory string `json:"workingDirectory,omitempty"` // Working directory from LNK
	Args             string `json:"args,omitempty"`             // Command line args from LNK
}

type UwpPackageInfo struct {
	PackageFamilyName string `json:"PackageFamilyName"`
	InstallLocation   string `json:"InstallLocation"`
}

// --- Core Functions ---

// resolveLnkTarget retrieves the target path from a Windows shortcut (.lnk) file.
// It prioritizes checks, expands environment variables, cleans the path,
// and attempts to resolve relative paths based on the LNK file's directory.
func resolveLnkTarget(lnkPath string) (string, error) {
	linkFile, err := lnk.File(lnkPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse lnk file '%s': %w", lnkPath, err)
	}

	// Determine the initial target path using prioritized fallback
	var targetPath string
	if linkFile.LinkInfo.LocalBasePath != "" {
		targetPath = linkFile.LinkInfo.LocalBasePath
	} else if linkFile.LinkInfo.CommonPathSuffix != "" {
		targetPath = linkFile.LinkInfo.CommonPathSuffix
	} else {
		targetPath = linkFile.StringData.RelativePath
	}

	if targetPath == "" {
		return "", nil
	} // No path found

	// Expand environment variables first
	expandedPath := os.ExpandEnv(targetPath)

	// Attempt to make paths absolute if they still look relative
	finalPath := expandedPath
	// Check if path is not absolute OR explicitly starts with '.' or '..' separators
	// (covers cases like '.\file.exe' or '..\dir\file.exe' that IsAbs might miss on Windows if they lack drive letter)
	if !filepath.IsAbs(expandedPath) || strings.HasPrefix(expandedPath, ".") || strings.HasPrefix(expandedPath, string(filepath.Separator)+".") {
		// Assume relative paths are relative to the directory containing the .lnk file
		lnkDir := filepath.Dir(lnkPath)
		absCandidate := filepath.Join(lnkDir, expandedPath)
		finalPath = absCandidate // Use this potentially better path
	}

	// Clean the potentially now-absolute path
	cleanedPath := filepath.Clean(finalPath)

	return cleanedPath, nil
}

func getStartMenuDirs() []string {
	// Consider platform APIs for robustness if needed.
	return []string{
		`C:\ProgramData\Microsoft\Windows\Start Menu\Programs`,
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`),
	}
}

// Add system executables as hardcoded entries
func addSystemApps(apps []AppEntry) []AppEntry {
	systemPaths := map[string]string{
		"explorer.exe": `C:\Windows\explorer.exe`,
		"taskmgr.exe":  `C:\Windows\System32\taskmgr.exe`,
		"cmd.exe":      `C:\Windows\System32\cmd.exe`,
	}

	for exeName, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			apps = append(apps, AppEntry{
				Name: systemApps[exeName],
				Path: path,
			})
		}
	}
	return apps
}

// --- Filtering Logic ---

func isWhitelisted(lowerName string, components []string) bool {
	for _, w := range whitelistKeywords {
		if strings.Contains(lowerName, w) {
			return true
		}
		for _, component := range components {
			if component != "" && strings.Contains(component, w) {
				return true
			}
		}
	}
	return false
}

func containsUnwantedKeyword(text string) bool {
	for _, k := range unwantedKeywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func hasUnwantedExtensionOrPattern(filename string) bool {
	if filename == "" {
		return false
	}
	if strings.HasPrefix(filename, "unins") && strings.HasSuffix(filename, ".exe") {
		return true
	}
	for _, ext := range nonExecExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

// isUnwantedEntry orchestrates filtering checks.
func isUnwantedEntry(name, path string) bool {
	if filename := filepath.Base(strings.ToLower(path)); systemApps[filename] != "" {
		return false
	}

	lowerName := strings.ToLower(name)
	lowerPath := strings.ToLower(path)

	if path == "" { // Cannot proceed with empty path
		return true
	}

	var components []string
	if lowerPath != "" {
		components = strings.Split(lowerPath, string(filepath.Separator))
	}

	// Check whitelist first
	if isWhitelisted(lowerName, components) {
		return false // Keep if whitelisted
	}

	// Check name keywords
	if containsUnwantedKeyword(lowerName) {
		return true
	}

	// Check path keywords
	for _, component := range components {
		if component != "" && containsUnwantedKeyword(component) {
			return true
		}
	}

	// Check filename patterns/extensions
	var lowerFilename string
	if len(components) > 0 {
		lowerFilename = components[len(components)-1]
	}
	if hasUnwantedExtensionOrPattern(lowerFilename) {
		return true
	}

	return false // Not unwanted
}

// --- Normalization ---

func normalizeAppName(name string) string {
	lower := strings.ToLower(name)
	lower = strings.TrimSuffix(lower, " (64-bit)")
	lower = strings.TrimSuffix(lower, " (x64)")
	lower = strings.TrimSuffix(lower, " (32-bit)")
	lower = strings.TrimSuffix(lower, " (x86)")
	lower = strings.TrimSuffix(lower, ".exe")
	lower = strings.Join(strings.Fields(lower), " ") // Consolidate whitespace
	return strings.TrimSpace(lower)
}

// --- Executable Selection Heuristic ---

func selectPrimaryExecutable(appName string, exePaths []string) string {
	if len(exePaths) == 0 {
		return ""
	}

	// Filter out unwanted candidates first
	candidates := make([]string, 0, len(exePaths))
	for _, p := range exePaths {
		if !isUnwantedEntry(appName, p) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	normalizedAppNameBase := normalizeAppName(appName)
	bestCandidate := ""
	shortestMatchLen := -1

	// 1. Prefer exact name match
	for _, p := range candidates {
		exeNameOnly := strings.TrimSuffix(strings.ToLower(filepath.Base(p)), ".exe")
		if exeNameOnly == normalizedAppNameBase {
			return p // Found best match
		}
	}

	// 2. Prefer name containment (shortest path wins tie)
	for _, p := range candidates {
		exeNameOnly := strings.TrimSuffix(strings.ToLower(filepath.Base(p)), ".exe")
		if strings.Contains(exeNameOnly, normalizedAppNameBase) {
			if bestCandidate == "" || len(p) < shortestMatchLen {
				bestCandidate = p
				shortestMatchLen = len(p)
			}
		}
	}

	// 3. Fallback to shortest path overall if no name match found
	if bestCandidate == "" {
		bestCandidate = candidates[0]
		for _, p := range candidates[1:] {
			if len(p) < len(bestCandidate) {
				bestCandidate = p
			}
		}
	}
	return bestCandidate
}

// --- Application Discovery (.lnk) ---

// processLnkEntry resolves, validates, and filters a single LNK file entry.
// It ensures the final path is absolute before checking existence and storing.
func processLnkEntry(linkPath string, info os.FileInfo, seenTargets map[string]bool) *AppEntry {
	linkName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
	resolvedPathFromLnk, err := resolveLnkTarget(linkPath)
	if err != nil || resolvedPathFromLnk == "" {
		return nil
	}

	// Convert the path from resolver to Absolute using CWD context (as a final guarantee).
	absPath, err := filepath.Abs(resolvedPathFromLnk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning (processLnkEntry): Failed to get final absolute path for '%s': %v. Skipping entry '%s'.\n", resolvedPathFromLnk, err, linkName)
		return nil
	}

	// Use the final absolute path for all subsequent checks and storage
	lowerAbsPath := strings.ToLower(absPath)

	if seenTargets[lowerAbsPath] {
		return nil
	} // Duplicate target path check
	if isUnwantedEntry(linkName, absPath) {
		return nil
	} // Filtering check

	targetExt := strings.ToLower(filepath.Ext(absPath)) // Extension check
	if targetExt != ".exe" && targetExt != ".bat" && targetExt != ".com" {
		return nil
	}

	// Final check for existence and type using the absolute path
	statInfo, err := os.Stat(absPath)
	if err != nil || statInfo.IsDir() {
		return nil
	}

	// Mark seen and return valid entry using the final absolute path
	seenTargets[lowerAbsPath] = true
	return &AppEntry{Name: linkName, Path: absPath} // Store the absolute path
}

// getExeApps finds applications by scanning .lnk files in standard Start Menu directories.
// It resolves LNK targets, filters unwanted entries, verifies target existence, and deduplicates.
// Returns:
// - []AppEntry: List of valid applications found via LNK files.
// - map[string]bool: Set of lowercase absolute target paths seen (for deduplication).
// - map[string]string: Map from lowercase absolute target path to the original LNK file path.
func getExeApps() ([]AppEntry, map[string]bool, map[string]string) {
	var apps []AppEntry
	seenTargets := make(map[string]bool)
	// Map from lowercase absolute target path -> original LNK file path
	lnkFilePaths := make(map[string]string)
	dirs := getStartMenuDirs()

	for _, dir := range dirs {
		_ = filepath.Walk(dir, func(linkPath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				// Handle walk errors silently in production unless specific logging is needed
				if info != nil && info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() || !strings.EqualFold(filepath.Ext(linkPath), ".lnk") {
				return nil // Not a LNK file
			}

			// Process the LNK file using the helper
			if appEntry := processLnkEntry(linkPath, info, seenTargets); appEntry != nil {
				apps = append(apps, *appEntry)
				// Store the mapping from the resolved absolute path back to the LNK file
				lnkFilePaths[strings.ToLower(appEntry.Path)] = linkPath
			}
			return nil
		})
	}
	// Return all three results
	return apps, seenTargets, lnkFilePaths
}

// processLnkEntry remains unchanged from the previous cleaned version
// It still returns *AppEntry containing the ABSOLUTE path if successful.
// resolveLnkTarget also remains unchanged.

// --- Application Discovery (UWP) ---

func getUwpPackageLocations() (map[string]string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", getUwpLocationsPsCommand)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getUwpPackageLocations): PowerShell command failed. Error: %v\nStderr:\n%s\n", err, stderr.String())
		return nil, fmt.Errorf("failed to get UWP package locations: %w", err)
	}

	output := out.Bytes()
	if len(output) == 0 {
		return make(map[string]string), nil
	} // No packages found is ok

	var packages []UwpPackageInfo
	if err := json.Unmarshal(output, &packages); err != nil {
		var singlePackage UwpPackageInfo
		if errSingle := json.Unmarshal(output, &singlePackage); errSingle != nil {
			fmt.Fprintf(os.Stderr, "ERROR (getUwpPackageLocations): Failed to parse PowerShell JSON output.\nOutput:\n%s\nError: %v\n", string(output), err)
			return nil, fmt.Errorf("failed to parse UWP package location JSON: %w", err)
		}
		packages = append(packages, singlePackage)
	}

	locationsMap := make(map[string]string, len(packages))
	for _, pkg := range packages {
		if pkg.PackageFamilyName != "" && pkg.InstallLocation != "" {
			locationsMap[strings.ToLower(pkg.PackageFamilyName)] = pkg.InstallLocation
		}
	}
	return locationsMap, nil
}

// processUwpEntry resolves, validates, and filters a single UWP entry from Get-StartApps.
// Returns an AppEntry pointer if valid, nil otherwise.
func processUwpEntry(name, appid string, packageLocations map[string]string) *AppEntry {
	primaryExePath := ""
	var packageExes []string

	// Attempt to find executables if AppID suggests a packaged app
	if strings.Contains(appid, "!") {
		familyName := strings.ToLower(strings.SplitN(appid, "!", 2)[0])
		installLocation, found := packageLocations[familyName]

		if found && installLocation != "" {
			globPattern := filepath.Join(installLocation, "*.exe")
			foundExes, err := filepath.Glob(globPattern)
			if err == nil && len(foundExes) > 0 { // Ignore glob errors, proceed if exes found
				packageExes = foundExes
			}
		}
	}

	// Select primary executable if candidates exist
	if len(packageExes) > 0 {
		primaryExePath = selectPrimaryExecutable(name, packageExes)
	}

	if primaryExePath == "" {
		return nil
	} // No suitable executable found

	cleanedExePath := filepath.Clean(primaryExePath)
	if isUnwantedEntry(name, cleanedExePath) {
		return nil
	} // Unwanted

	// Final check for existence/type
	statInfo, err := os.Stat(cleanedExePath)
	if err != nil || statInfo.IsDir() {
		return nil
	}

	return &AppEntry{Name: name, Path: cleanedExePath}
}

func getUWPApps() []AppEntry {
	var apps []AppEntry

	packageLocations, err := getUwpPackageLocations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getUWPApps): Failed to get package locations map, proceeding without it: %v\n", err)
		packageLocations = make(map[string]string) // Ensure non-nil map
	}

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", getStartAppsPsCommand)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getUWPApps): Get-StartApps command failed. Error: %v\nStderr:\n%s\n", err, stderr.String())
		return apps // Cannot proceed without app list
	}

	scanner := bufio.NewScanner(&out)
	processedAppIDs := make(map[string]bool) // Deduplicate Get-StartApps output

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		} // Skip malformed

		name := strings.TrimSpace(parts[0])
		appid := strings.TrimSpace(parts[1])
		if name == "" || appid == "" {
			continue
		} // Skip empty

		lowerAppID := strings.ToLower(appid)
		if processedAppIDs[lowerAppID] {
			continue
		} // Skip duplicate AppID
		processedAppIDs[lowerAppID] = true

		// Process the UWP entry using the helper
		if appEntry := processUwpEntry(name, appid, packageLocations); appEntry != nil {
			apps = append(apps, *appEntry)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getUWPApps): Scanner error reading PowerShell output: %v\n", err)
	}
	return apps
}

// --- Main Execution ---

func FetchExecutableApplicationMap() map[string]FinalAppOutput {
	// Step 1: Discover EXE apps from LNK files.
	// Keep using your existing getExeApps function.
	exeApps, seenExeTargets, exeLnkPaths := getExeApps()
	// Log errors from getExeApps if it were modified to return them, otherwise
	// assume it logs internally or they are handled in processLnkEntry.

	// Step 2: Discover UWP apps and resolve executables.
	// Keep using your existing getUWPApps function.
	uwpAppsResolved := getUWPApps()
	// Log errors from getUWPApps if it returned them.

	// Step 3: Combine lists, ensuring UWP entries don't duplicate already found EXEs.
	// Keep your existing combination logic.
	combinedAppEntries := make([]AppEntry, 0, len(exeApps)+len(uwpAppsResolved))
	combinedAppEntries = append(combinedAppEntries, exeApps...)
	combinedAppEntries = addSystemApps(combinedAppEntries)

	for _, uwpApp := range uwpAppsResolved {
		// Assuming uwpApp is of type AppEntry now
		lowerUwpPath := strings.ToLower(uwpApp.Path)
		if !seenExeTargets[lowerUwpPath] {
			combinedAppEntries = append(combinedAppEntries, uwpApp)
			// Add to seen targets to prevent duplicates if UWP list has them
			seenExeTargets[lowerUwpPath] = true
		}
	}

	// Step 4: Sort the combined list for consistent output order.
	// Keep your existing sorting logic.
	sort.Slice(combinedAppEntries, func(i, j int) bool {
		normNameI := normalizeAppName(combinedAppEntries[i].Name)
		normNameJ := normalizeAppName(combinedAppEntries[j].Name)
		if normNameI != normNameJ {
			return normNameI < normNameJ
		}
		// Fallback sort by path
		return strings.ToLower(combinedAppEntries[i].Path) < strings.ToLower(combinedAppEntries[j].Path)
	})

	// Step 5: Build final output map (Path -> FinalAppOutput).
	// Keep your existing logic for building the map and extracting LNK details.
	finalMap := make(map[string]FinalAppOutput, len(combinedAppEntries))
	processedPathsForOutput := make(map[string]bool, len(combinedAppEntries))

	for _, appEntry := range combinedAppEntries {
		pathKey := appEntry.Path
		lowerPathKey := strings.ToLower(pathKey)

		// Deduplicate based on final path key during map creation.
		if _, exists := processedPathsForOutput[lowerPathKey]; exists {
			continue
		}

		// Create the FinalAppOutput value with proper name handling
		outputValue := FinalAppOutput{Name: appEntry.Name}

		// Check if it's a system app and use the system name
		if sysName, isSys := systemApps[strings.ToLower(filepath.Base(pathKey))]; isSys {
			outputValue.Name = sysName
			// For system apps, don't include any LNK data
			finalMap[pathKey] = outputValue
			processedPathsForOutput[lowerPathKey] = true
			continue // Skip LNK processing for system apps
		}

		// Check if this app originated from an LNK file to get extra data
		if originalLnkPath, found := exeLnkPaths[lowerPathKey]; found {
			// Re-parse the LNK to get extra data.
			linkFile, err := lnk.File(originalLnkPath)
			if err == nil {
				flagMap := linkFile.Header.LinkFlags
				if flagMap["HasWorkingDir"] {
					// Consider making WD absolute relative to LNK or Target Dir
					outputValue.WorkingDirectory = linkFile.StringData.WorkingDir
				}
				if flagMap["HasArguments"] {
					outputValue.Args = linkFile.StringData.CommandLineArguments
				}
			} else {
				log.Printf("Warning: Failed to re-parse LNK '%s' for extra data: %v\n", originalLnkPath, err)
			}
		}
		// UWP apps will naturally have empty extra fields.

		finalMap[pathKey] = outputValue
		processedPathsForOutput[lowerPathKey] = true // Mark path as added.
	}

	// Step 6: Return the result map (removed JSON marshaling/printing)
	log.Printf("Application discovery finished. Found %d unique applications.", len(finalMap))
	return finalMap
}
