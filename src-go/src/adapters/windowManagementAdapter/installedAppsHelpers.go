package windowManagementAdapter

import (
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	lnk "github.com/parsiya/golnk"
)

// prepareCombinedAppList consolidates applications from various sources:
// EXEs, Start Menu items, and system-defined applications.
// It also handles initial deduplication based on seen target paths.
func prepareCombinedAppList(
	exeApps []AppEntry,
	startMenuApps []AppEntry,
	seenExeTargets map[string]bool, // This map is now modified by addSystemApps
) []AppEntry {
	// Estimate capacity. Consider that systemApps might add a few more.
	estimatedCapacity := len(exeApps) + len(startMenuApps) + len(systemApps) // Using global systemApps map for count
	combined := make([]AppEntry, 0, estimatedCapacity)

	combined = append(combined, exeApps...) // exeApps already populated seenExeTargets

	// Add system-defined applications.
	// Pass seenExeTargets so addSystemApps can check and update it.
	combined = addSystemApps(combined, seenExeTargets) // MODIFIED CALL

	// Add Start Menu apps, avoiding duplicates already found (either as EXE targets or added system apps).
	for _, smApp := range startMenuApps {
		lowerIdentifier := strings.ToLower(smApp.Path)
		if !seenExeTargets[lowerIdentifier] {
			combined = append(combined, smApp)
			seenExeTargets[lowerIdentifier] = true
		}
	}
	return combined
}

// sortAppEntries sorts the application list primarily by normalized name,
// then by path for stability.
func sortAppEntries(entries []AppEntry) {
	sort.Slice(entries, func(i, j int) bool {
		normNameI := normalizeAppName(entries[i].Name)
		normNameJ := normalizeAppName(entries[j].Name)
		if normNameI != normNameJ {
			return normNameI < normNameJ
		}
		return strings.ToLower(entries[i].Path) < strings.ToLower(entries[j].Path)
	})
}

// buildLaunchInfo constructs a AppInfo struct for a given AppEntry.
// It populates ExePath, URI, IconPath, and LNK-specific data if applicable.
func buildLaunchInfo(
	appEntry AppEntry,
	isSystemApp bool,
	exeLnkPaths map[string]string, // map[lower(targetExePath)]lnkFilePath
) core.AppInfo {
	info := core.AppInfo{
		ExePath: appEntry.Path,
		URI:     appEntry.URI,
	}

	iconPath, errIcon := GetIconPathForExe(appEntry.Path)
	if errIcon != nil {
		log.Printf("Info: Could not retrieve icon for '%s' (identifier: %s): %v", appEntry.Name, appEntry.Path, errIcon)
	}
	info.IconPath = iconPath

	// Populate LNK details (WorkingDirectory, Args) if not a system app and not URI-based.
	if !isSystemApp && appEntry.URI == "" {
		lowerIdentifierPath := strings.ToLower(appEntry.Path)
		if originalLnkPath, lnkFound := exeLnkPaths[lowerIdentifierPath]; lnkFound {
			// lnk.File returns (LnkFile, error) - LnkFile is a struct value.
			linkFile, errLnk := lnk.File(originalLnkPath)
			if errLnk == nil {
				// Assuming that if errLnk is nil, linkFile.Header.LinkFlags is a valid (non-nil) map,
				// as per the direct access pattern in the original code.
				// Accessing a key in a nil map would panic.
				flagMap := linkFile.Header.LinkFlags
				if flagMap["HasWorkingDir"] {
					info.WorkingDirectory = linkFile.StringData.WorkingDir
				}
				if flagMap["HasArguments"] {
					if !appEntry.ResolvedFromArguments {
						info.Args = linkFile.StringData.CommandLineArguments
					}
				}
			} else {
				log.Printf("Warning: Failed to parse LNK file '%s' for app '%s' (exe: '%s'): %v",
					originalLnkPath, appEntry.Name, appEntry.Path, errLnk)
			}
		}
	}
	return info
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
