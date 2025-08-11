package windowManagementAdapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	lnk "github.com/parsiya/golnk"
)

// --- Constants ---

const (
	getPackageLocationsPsCommand = `Get-AppxPackage | Where-Object {$_.InstallLocation} | Select-Object -Property PackageFamilyName, InstallLocation | ConvertTo-Json -Depth 2 -Compress`
	getStartAppsPsCommand        = `Get-StartApps | Select-Object Name, AppID | Where-Object { $_.AppID -ne $null -and $_.AppID -ne '' -and $_.AppID -notlike '*SystemSettings*' -and $_.AppID -notlike '*Search*' } | ForEach-Object { ($_.Name -replace '\t',' ') + "` + "\t" + `" + $_.AppID }`
)

// Web app package identifiers
const (
	DisneyPlusPattern = "Disney."
	NetflixPattern    = ".Netflix"
	YTMusicPattern    = "music.youtube"
)

// Filtering lists
var (
	unwantedKeywords = []string{
		"uninstall", "uninst", "remove", "setup", "install", "update", "updater", "patch",
		"crash", "debug", "wizard", "bootstrapper", "vcredist", "redist", "dotnet", "report",
		"readme", "eula", "license", "windows performance toolkit", "app certification kit",
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

// --- Core Functions ---

// extractExeFromArgs extracts the first .exe filename from command-line arguments.
func extractExeFromArgs(args string) string {
	for _, part := range strings.Fields(args) {
		if strings.HasSuffix(strings.ToLower(part), ".exe") {
			return part
		}
	}
	return ""
}

// resolveExePath tries to resolve an exe path directly, and if not found, searches recursively in baseDir.
func resolveExePath(baseDir, exeName string) string {
	tryPath := exeName
	if !filepath.IsAbs(tryPath) {
		tryPath = filepath.Join(baseDir, exeName)
	}
	tryPathAbs, err := filepath.Abs(tryPath)
	if err == nil {
		tryPathAbs = filepath.Clean(tryPathAbs)
		statInfo, err := os.Stat(tryPathAbs)
		if err == nil && !statInfo.IsDir() {
			return tryPathAbs
		}
	}
	// Not found directly; search recursively
	found := findExeRecursive(baseDir, exeName)
	return found
}

// findExeRecursive searches for exeName recursively in baseDir. Returns absolute path if found, else empty string.
func findExeRecursive(baseDir, exeName string) string {
	var foundPath string
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.EqualFold(info.Name(), exeName) {
			foundPath = path
			return filepath.SkipDir // Stop after first match
		}
		return nil
	})
	return foundPath
}

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

// addSystemApps adds hardcoded system executables to the apps list
// if they exist on disk and their paths haven't been seen yet.
// It updates seenExeTargets for any apps it adds.
func addSystemApps(apps []AppEntry, seenExeTargets map[string]ShortcutInfo) []AppEntry {
	systemPaths := map[string]string{ // Base exe name -> Full Path
		"explorer.exe": `C:\Windows\explorer.exe`,
		"taskmgr.exe":  `C:\Windows\System32\taskmgr.exe`,
		"cmd.exe":      `C:\Windows\System32\cmd.exe`,
		// Add other system apps here
	}

	for exeBaseName, fullPath := range systemPaths {
		lowerFullPath := strings.ToLower(fullPath)

		// Only add if the path hasn't been seen from other sources (LNKs, etc.)
		if _, seen := seenExeTargets[lowerFullPath]; seen {
			continue
		}

		// Check if the system executable actually exists at the specified path
		if _, err := os.Stat(fullPath); err == nil { // File exists
			displayName, nameExists := systemApps[exeBaseName]
			if !nameExists {
				// Fallback name if not in systemApps map (should not happen for predefined list)
				log.Warn("Warning: System app base name '%s' not found in systemApps display name map. Using base name.", exeBaseName)
				displayName = exeBaseName
			}

			apps = append(apps, AppEntry{
				Name: displayName, // Use the display name from systemApps
				Path: fullPath,
				// URI will be empty for these traditional executables
			})
			// Mark this path as seen so it won't be added again by later stages (e.g., Start Menu scan)
			seenExeTargets[lowerFullPath] = ShortcutInfo{HasArguments: false, AppIndex: len(apps) - 1}
		}
	}
	return apps
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
func processLnkEntry(linkPath string, info os.FileInfo, seenTargets map[string]ShortcutInfo) *AppEntry {
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

	// Check if this shortcut has arguments
	linkFile, err := lnk.File(linkPath)
	hasArgs := err == nil && linkFile.StringData.CommandLineArguments != ""
	
	// Check if we've already seen this target path
	if info, exists := seenTargets[lowerAbsPath]; exists {
		// If current shortcut has arguments, skip it (keep the existing one)
		if hasArgs {
			return nil
		}
		
		// If this shortcut doesn't have arguments and previous had arguments, replace it
		if info.HasArguments {
			log.Debug("Using shortcut without arguments '%s' instead of previous one with arguments (index: %d)", linkName, info.AppIndex)
			// Return this entry to replace the previous one with arguments
			// The index will be used in getExeApps to remove the old entry
			return &AppEntry{Name: linkName, Path: absPath, ReplaceIndex: info.AppIndex}
		} else {
			// Both shortcuts don't have arguments, keep the first one
			return nil
		}
	}
	if isUnwantedEntry(linkName, absPath) {
		linkFile, err := lnk.File(linkPath)
		if err == nil && linkFile.StringData.CommandLineArguments != "" {
			args := linkFile.StringData.CommandLineArguments
			exeName := extractExeFromArgs(args)
			if exeName != "" {
				resolvedExe := resolveExePath(filepath.Dir(absPath), exeName)
				if resolvedExe != "" {
					lowerResolvedExe := strings.ToLower(resolvedExe)
					if _, exists := seenTargets[lowerResolvedExe]; !exists {
						seenTargets[lowerResolvedExe] = ShortcutInfo{HasArguments: true, AppIndex: -1}
						return &AppEntry{Name: linkName, Path: resolvedExe, ResolvedFromArguments: true, ReplaceIndex: -1}
					}
				}
			}
		}
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

	// Store whether this shortcut has arguments in the seenTargets map
	seenTargets[lowerAbsPath] = ShortcutInfo{HasArguments: false, AppIndex: -1} // Default to no arguments
	
	// Check if this shortcut has arguments and update the map if it does
	argLinkFile, argErr := lnk.File(linkPath)
	if argErr == nil && argLinkFile.StringData.CommandLineArguments != "" {
		seenTargets[lowerAbsPath] = ShortcutInfo{HasArguments: true, AppIndex: -1}
	}
	
	// Return valid entry using the final absolute path
	return &AppEntry{Name: linkName, Path: absPath} // Store the absolute path
}

// ShortcutInfo tracks information about a shortcut for better deduplication
type ShortcutInfo struct {
	HasArguments bool
	AppIndex     int  // Index in the apps slice, -1 if not added yet
}

// getExeApps finds applications by scanning .lnk files in standard Start Menu directories.
// It resolves LNK targets, filters unwanted entries, verifies target existence, and deduplicates.
// Returns:
// - []AppEntry: List of valid applications found via LNK files.
// - map[string]ShortcutInfo: Map of lowercase absolute target paths to shortcut info (for deduplication).
// - map[string]string: Map from lowercase absolute target path to the original LNK file path.
func getExeApps() ([]AppEntry, map[string]ShortcutInfo, map[string]string) {
	// First pass: collect all valid shortcuts
	type ShortcutData struct {
		Entry    AppEntry
		LnkPath  string
		HasArgs  bool
	}
	
	// Map from lowercase target path to all shortcuts targeting that path
	allShortcuts := make(map[string][]ShortcutData)
	dirs := getStartMenuDirs()

	// First pass: collect all shortcuts without any filtering by arguments
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

			// Get target path and check if it's valid
			absPath, err := resolveLnkTarget(linkPath)
			if err != nil || absPath == "" {
				return nil // Invalid or empty target
			}

			// Get shortcut name (filename without extension)
			linkName := strings.TrimSuffix(filepath.Base(linkPath), filepath.Ext(linkPath))

			// Filter unwanted entries
			if isUnwantedEntry(linkName, absPath) {
				// Check if this unwanted entry has arguments that point to a valid exe
				linkFile, err := lnk.File(linkPath)
				if err == nil && linkFile.StringData.CommandLineArguments != "" {
					args := linkFile.StringData.CommandLineArguments
					exeName := extractExeFromArgs(args)
					if exeName != "" {
						resolvedExe := resolveExePath(filepath.Dir(absPath), exeName)
						if resolvedExe != "" {
							// Found a valid exe in the arguments
							lowerResolvedExe := strings.ToLower(resolvedExe)
							allShortcuts[lowerResolvedExe] = append(allShortcuts[lowerResolvedExe], ShortcutData{
								Entry:    AppEntry{Name: linkName, Path: resolvedExe, ResolvedFromArguments: true},
								LnkPath:  linkPath,
								HasArgs:  true,
							})
						}
					}
				}
				return nil
			}

			// Extension check
			targetExt := strings.ToLower(filepath.Ext(absPath))
			if targetExt != ".exe" && targetExt != ".bat" && targetExt != ".com" {
				return nil
			}

			// Final check for existence and type using the absolute path
			statInfo, err := os.Stat(absPath)
			if err != nil || statInfo.IsDir() {
				return nil
			}

			// Check if this shortcut has arguments
			hasArgs := false
			argLinkFile, argErr := lnk.File(linkPath)
			if argErr == nil && argLinkFile.StringData.CommandLineArguments != "" {
				hasArgs = true
			}

			// Store this shortcut in our collection
			lowerAbsPath := strings.ToLower(absPath)
			allShortcuts[lowerAbsPath] = append(allShortcuts[lowerAbsPath], ShortcutData{
				Entry:    AppEntry{Name: linkName, Path: absPath},
				LnkPath:  linkPath,
				HasArgs:  hasArgs,
			})

			return nil
		})
	}

	// Second pass: select the best shortcut for each target path
	var apps []AppEntry
	seenTargets := make(map[string]ShortcutInfo)
	lnkFilePaths := make(map[string]string)

	// Process all collected shortcuts
	for targetPath, shortcuts := range allShortcuts {
		// Check if we have any shortcuts without arguments
		hasNoArgShortcut := false
		bestShortcutIndex := 0

		// First, look for shortcuts without arguments
		for i, sc := range shortcuts {
			if !sc.HasArgs {
				hasNoArgShortcut = true
				bestShortcutIndex = i
				break
			}
		}

		// If we found a shortcut without arguments, log any discarded shortcuts with arguments
		if hasNoArgShortcut {
			bestShortcut := shortcuts[bestShortcutIndex]
			
			// Log discarded shortcuts with arguments
			for _, sc := range shortcuts {
				if sc.HasArgs {
					log.Debug("Discarding shortcut with arguments: '%s' (from '%s') in favor of: '%s' (from '%s')", 
						sc.Entry.Name, sc.LnkPath, bestShortcut.Entry.Name, bestShortcut.LnkPath)
				}
			}

			// Add the best shortcut to our final list
			apps = append(apps, bestShortcut.Entry)
			appIndex := len(apps) - 1
			
			// Update maps
			lnkFilePaths[targetPath] = bestShortcut.LnkPath
			seenTargets[targetPath] = ShortcutInfo{HasArguments: false, AppIndex: appIndex}
		} else if len(shortcuts) > 0 {
			// If all shortcuts have arguments, just use the first one
			bestShortcut := shortcuts[0]
			apps = append(apps, bestShortcut.Entry)
			appIndex := len(apps) - 1
			
			// Update maps
			lnkFilePaths[targetPath] = bestShortcut.LnkPath
			seenTargets[targetPath] = ShortcutInfo{HasArguments: true, AppIndex: appIndex}
		}
	}

	// Return all three results
	return apps, seenTargets, lnkFilePaths
}

// processLnkEntry remains unchanged from the previous cleaned version
// It still returns *AppEntry containing the ABSOLUTE path if successful.
// resolveLnkTarget also remains unchanged.

// --- Application Discovery (Start Menu) ---

func getPackageLocations() (map[string]string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", getPackageLocationsPsCommand)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getPackageLocations): PowerShell command failed. Error: %v\nStderr:\n%s\n", err, stderr.String())
		return nil, fmt.Errorf("failed to get UWP package locations: %w", err)
	}

	output := out.Bytes()
	if len(output) == 0 {
		return make(map[string]string), nil
	} // No packages found is ok

	var packages []PackageInfo
	if err := json.Unmarshal(output, &packages); err != nil {
		var singlePackage PackageInfo
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

func findUWPExecutables(installLocation string) []string {
	var exes []string

	// Walk through all subdirectories
	filepath.Walk(installLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip on errors
		}

		// Check if it's a file and has .exe extension
		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".exe") {
			// Skip some known system/helper executables
			filename := strings.ToLower(info.Name())
			if !strings.HasPrefix(filename, "runtime") &&
				!strings.HasPrefix(filename, "vcruntime") &&
				!strings.HasPrefix(filename, "msvcp") &&
				!strings.HasPrefix(filename, "api-ms-") {
				exes = append(exes, path)
			}
		}
		return nil
	})

	return exes
}

// processStartMenuEntry resolves, validates, and filters a single entry from Get-StartApps.
// Returns an AppEntry pointer if valid, nil otherwise.
func processStartMenuEntry(name, appid string, packageLocations map[string]string) *AppEntry {
	// Check for web apps first by their package patterns
	if strings.Contains(appid, DisneyPlusPattern) ||
		strings.Contains(appid, NetflixPattern) ||
		strings.Contains(appid, YTMusicPattern) {
		return &AppEntry{
			Name: name,
			Path: fmt.Sprintf("shell:AppsFolder\\%s", appid), // Store the URI as path
			URI:  fmt.Sprintf("shell:AppsFolder\\%s", appid),
		}
	}

	primaryExePath := ""
	var packageExes []string

	// Check if this is a UWP app
	if strings.Contains(appid, "!") {
		familyName := strings.ToLower(strings.SplitN(appid, "!", 2)[0])
		installLocation, found := packageLocations[familyName]

		if found && installLocation != "" {
			// Use the new function to find executables recursively
			packageExes = findUWPExecutables(installLocation)
		}
	}

	if len(packageExes) > 0 {
		primaryExePath = selectPrimaryExecutable(name, packageExes)
	}

	if primaryExePath == "" {
		// if strings.Contains(appid, "!") {
		// 	fmt.Printf("Rejected UWP app - No exe path found: %s (AppID: %s)\n", name, appid)
		// }
		return nil
	}

	cleanedExePath := filepath.Clean(primaryExePath)
	if isUnwantedEntry(name, cleanedExePath) {
		return nil
	} // Unwanted

	// Final check for existence/type
	statInfo, err := os.Stat(cleanedExePath)
	if err != nil || statInfo.IsDir() {
		return nil
	}

	// Create AppEntry with additional URI field
	return &AppEntry{
		Name: name,
		Path: cleanedExePath,
		URI:  fmt.Sprintf("shell:AppsFolder\\%s", appid), // Store the URI
	}
}

func getStartMenuApps() []AppEntry {
	var apps []AppEntry

	packageLocations, err := getPackageLocations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getStartMenuApps): Failed to get package locations map, proceeding without it: %v\n", err)
		packageLocations = make(map[string]string) // Ensure non-nil map
	}

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", getStartAppsPsCommand)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getStartMenuApps): Get-StartApps command failed. Error: %v\nStderr:\n%s\n", err, stderr.String())
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
		if appEntry := processStartMenuEntry(name, appid, packageLocations); appEntry != nil {
			apps = append(apps, *appEntry)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR (getStartMenuApps): Scanner error reading PowerShell output: %v\n", err)
	}
	return apps
}

// --- Main Execution ---

// FetchExecutableApplicationMap discovers applications and returns a map of
// unique application names to their launch information.
func FetchExecutableApplicationMap() map[string]core.AppInfo {
	exeApps, seenExeTargets, exeLnkPaths := getExeApps()
	startMenuApps := getStartMenuApps()

	combinedEntries := prepareCombinedAppList(exeApps, startMenuApps, seenExeTargets)
	sortAppEntries(combinedEntries)

	finalMap := make(map[string]core.AppInfo, len(combinedEntries))

	for _, appEntry := range combinedEntries {
		baseAppName := appEntry.Name
		isSystemApp := false

		if baseFileName := filepath.Base(appEntry.Path); baseFileName != "." && baseFileName != "/" {
			if sysName, isSys := systemApps[strings.ToLower(baseFileName)]; isSys {
				baseAppName = sysName
				isSystemApp = true
			}
		}

		uniqueAppNameKey := baseAppName
		if _, exists := finalMap[uniqueAppNameKey]; exists {
			count := 1
			for {
				uniqueAppNameKey = fmt.Sprintf("%s (%d)", baseAppName, count)
				if _, nameExists := finalMap[uniqueAppNameKey]; !nameExists {
					break
				}
				count++
				if count > 100 {
					log.Warn("Warning: Exceeded max attempts to generate unique name for '%s'. Using: '%s'", baseAppName, uniqueAppNameKey)
					break
				}
			}
		}

		launchInfo := buildLaunchInfo(appEntry, isSystemApp, exeLnkPaths)
		finalMap[uniqueAppNameKey] = launchInfo
	}

	log.Info("Application discovery finished. Found %d applications.", len(finalMap))
	return finalMap
}
