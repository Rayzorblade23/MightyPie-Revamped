package piemenuConfigManager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// ReadConfigFromFile reads the unified PieMenuConfig from disk.
// It primarily expects the unified format and ensures non-nil maps on success.
// If the file contains a legacy buttons-only structure (ConfigData), it will be
// wrapped into a PieMenuConfig to allow seamless migration.
func ReadConfigFromFile(path string) (PieMenuConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return PieMenuConfig{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return PieMenuConfig{}, err
	}

	// Try unified format first
	var cfg PieMenuConfig
	if err := json.Unmarshal(data, &cfg); err == nil {
		if cfg.Buttons == nil {
			cfg.Buttons = ConfigData{}
		}
		if cfg.Shortcuts == nil {
			cfg.Shortcuts = map[string]ShortcutEntry{}
		}
		return cfg, nil
	}

	// Fallback: legacy buttons-only file. Attempt to unmarshal as ConfigData and wrap.
	var buttonsOnly ConfigData
	if err := json.Unmarshal(data, &buttonsOnly); err == nil && buttonsOnly != nil {
		// Ensure non-nil maps
		wrapped := PieMenuConfig{
			Buttons:   buttonsOnly,
			Shortcuts: map[string]ShortcutEntry{},
			Starred:   nil,
		}
		return wrapped, nil
	}

	return PieMenuConfig{}, errors.New("invalid unified PieMenuConfig format")
}

func WriteConfigToFile(path string, cfg PieMenuConfig) error {
	if path == "" {
		return errors.New("empty config path")
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// BackupFullConfigToFile writes the unified PieMenuConfig to a backup file in the standard backups directory.
func BackupFullConfigToFile(cfg PieMenuConfig) error {
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		return err
	}
	backupsRel := os.Getenv("PUBLIC_DIR_CONFIGBACKUPS")
	backupDir := filepath.Join(appDataDir, backupsRel)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory '%s': %w", backupDir, err)
	}

	baseName := "piemenuConfig_BACKUP.json"
	backupPath := filepath.Join(backupDir, baseName)
	idx := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		idx++
		backupPath = filepath.Join(backupDir, fmt.Sprintf("%s_%d%s", "piemenuConfig_BACKUP", idx, ".json"))
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, data, 0644)
}

func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}
