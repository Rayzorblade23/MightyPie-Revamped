package shortcutSetterAdapter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/jsonUtils"
)

const (
	jsonExtension    = ".json"
	shortcutsFileName = "shortcuts"
)

func (a *ShortcutSetterAdapter) SaveShortcut(index int, shortcut []int) error {
	shortcuts, err := LoadShortcuts()
	if err != nil {
		return fmt.Errorf("failed to load shortcuts: %w", err)
	}

	// Use the helper to build the label
	label := ShortcutCodesToString(shortcut)

	// Save/overwrite the shortcut
	entry := core.ShortcutEntry{
		Codes: shortcut,
		Label: label,
	}
	shortcuts[strconv.Itoa(index)] = entry

	shortcutsPath := getShortcutConfigPath()
	if err := jsonUtils.WriteToFile(shortcutsPath, shortcuts); err != nil {
		return fmt.Errorf("failed to write shortcuts file: %w", err)
	}

	// --- Send NATS message with the whole map ---
	subject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	if a.natsAdapter != nil {
		a.natsAdapter.PublishMessage(subject, shortcuts)
	}

	return nil
}

func LoadShortcuts() (ShortcutMap, error) {
	shortcutsPath := getShortcutConfigPath()
	shortcuts := make(ShortcutMap)

	// Ensure the config file exists.
	if _, err := os.Stat(shortcutsPath); os.IsNotExist(err) {
		log.Printf("INFO: Shortcuts file not found at %s. Creating a new empty file.", shortcutsPath)
		if err := jsonUtils.WriteToFile(shortcutsPath, shortcuts); err != nil {
			return nil, fmt.Errorf("failed to create initial shortcuts file: %w", err)
		}
	}

	// Now, read the file (either the original or the newly created one).
	if err := jsonUtils.ReadFromFile(shortcutsPath, &shortcuts); err != nil {
		return nil, fmt.Errorf("failed to read shortcuts file: %w", err)
	}

	// If the file was empty (but existed), ReadFromFile returns a nil error
	// and shortcuts will be an empty map. We can just return it.
	return shortcuts, nil
}

// ShortcutCodesToString returns a human-readable string for a slice of key codes.
func getShortcutConfigPath() string {
	localAppData := os.Getenv("LOCALAPPDATA")
		return filepath.Join(localAppData, env.Get("PUBLIC_APPNAME"), shortcutsFileName+jsonExtension)
}

func ShortcutCodesToString(codes []int) string {
	names := []string{}
	for _, k := range codes {
		name := core.FindKeyByValue(k)
		if name == "" {
			name = fmt.Sprintf("VK_%d", k)
		}
		names = append(names, name)
	}
	return strings.Join(names, " + ")
}

// Add a function to delete a shortcut by index
func (a *ShortcutSetterAdapter) DeleteShortcut(index int) error {
	shortcuts, err := LoadShortcuts()
	if err != nil {
		return fmt.Errorf("failed to load shortcuts: %w", err)
	}

	// Delete the shortcut at the given index
	delete(shortcuts, strconv.Itoa(index))

	shortcutsPath := getShortcutConfigPath()
	if err := jsonUtils.WriteToFile(shortcutsPath, shortcuts); err != nil {
		return fmt.Errorf("failed to write shortcuts file: %w", err)
	}

	// --- Send NATS message with the whole map ---
	subject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	if a.natsAdapter != nil {
		a.natsAdapter.PublishMessage(subject, shortcuts)
	}

	return nil
}
