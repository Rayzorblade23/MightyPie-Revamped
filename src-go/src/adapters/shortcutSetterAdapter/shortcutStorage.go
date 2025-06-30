package shortcutSetterAdapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

func (a *ShortcutSetterAdapter) SaveShortcut(index int, shortcut []int) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configDir := filepath.Join(localAppData, "MightyPieRevamped")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	shortcutsPath := filepath.Join(configDir, "shortcuts.json")

	// Read existing shortcuts
	shortcuts := make(ShortcutMap)
	if data, err := os.ReadFile(shortcutsPath); err == nil {
		_ = json.Unmarshal(data, &shortcuts)
	}

	// Use the helper to build the label
	label := ShortcutCodesToString(shortcut)

	// Save/overwrite the shortcut
	entry := core.ShortcutEntry{
		Codes: shortcut,
		Label: label,
	}
	shortcuts[strconv.Itoa(index)] = entry

	// Write back to file
	data, err := json.MarshalIndent(shortcuts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shortcuts: %w", err)
	}
	if err := os.WriteFile(shortcutsPath, data, 0644); err != nil {
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
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	shortcutsPath := filepath.Join(localAppData, "MightyPieRevamped", "shortcuts.json")

	shortcuts := make(ShortcutMap)
	data, err := os.ReadFile(shortcutsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return shortcuts, nil
		}
		return nil, fmt.Errorf("failed to read shortcuts file: %w", err)
	}
	if err := json.Unmarshal(data, &shortcuts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shortcuts: %w", err)
	}
	return shortcuts, nil
}

// ShortcutCodesToString returns a human-readable string for a slice of key codes.
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
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}
	configDir := filepath.Join(localAppData, "MightyPieRevamped")
	shortcutsPath := filepath.Join(configDir, "shortcuts.json")

	// Read existing shortcuts
	shortcuts := make(ShortcutMap)
	if data, err := os.ReadFile(shortcutsPath); err == nil {
		_ = json.Unmarshal(data, &shortcuts)
	}

	// Delete the shortcut at the given index
	delete(shortcuts, strconv.Itoa(index))

	// Write back to file
	data, err := json.MarshalIndent(shortcuts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shortcuts: %w", err)
	}
	if err := os.WriteFile(shortcutsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write shortcuts file: %w", err)
	}

	// --- Send NATS message with the whole map ---
	subject := env.Get("PUBLIC_NATSSUBJECT_SHORTCUTSETTER_UPDATE")
	if a.natsAdapter != nil {
		a.natsAdapter.PublishMessage(subject, shortcuts)
	}

	return nil
}
