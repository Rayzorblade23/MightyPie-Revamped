package piemenuConfigManager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core/logger"
	"github.com/nats-io/nats.go"
)

var log = logger.New("PieMenuConfigManager")

type Adapter struct {
	nats *natsAdapter.NatsAdapter
	mu   sync.RWMutex
	cfg  PieMenuConfig
}

// logShortcuts prints a concise summary of current shortcuts for visibility
func logShortcuts(m map[string]ShortcutEntry) {
    if m == nil {
        log.Info("Shortcuts: none (map is nil)")
        return
    }
    log.Info("Shortcuts: %d entries", len(m))
    // Print up to 10 entries for brevity
    shown := 0
    for k, v := range m {
        log.Info("↳ [%s] %s %v", k, v.Label, v.Codes)
        shown++
        if shown >= 10 {
            if len(m) > shown {
                log.Info("↳ ... %d more ...", len(m)-shown)
            }
            break
        }
    }
}

func New(na *natsAdapter.NatsAdapter) *Adapter {
	if na == nil {
		log.Fatal("FATAL: NATS adapter cannot be nil")
	}
	ad := &Adapter{nats: na}

	backendSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_BACKEND_UPDATE")
	frontendSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_FRONTEND_UPDATE")
	relConfig := os.Getenv("PUBLIC_DIR_PIEMENUCONFIG")
	appDataDir, err := core.GetAppDataDir()
	if err != nil {
		log.Warn("Failed to resolve app data dir: %v.", err)
	}
	configPath := filepath.Join(appDataDir, relConfig)

	// Load file (if exists) and publish initial
	if cfg, err := ReadConfigFromFile(configPath); err == nil {
		// If buttons section is empty, populate defaults
		if len(cfg.Buttons) == 0 {
			cfg.Buttons = newDefaultButtons()
			// persist the defaulted config so everyone sees baseline structure
			if err := WriteConfigToFile(configPath, cfg); err != nil {
				log.Error("Failed to write defaulted full config: %v", err)
			}
		}
		ad.setConfig(cfg)
		log.Info("Loaded pie menu config from '%s'", configPath)
		logShortcuts(cfg.Shortcuts)
	} else {
		// Initialize with default buttons and empty shortcuts
		def := PieMenuConfig{Buttons: newDefaultButtons(), Shortcuts: map[string]ShortcutEntry{}, Starred: nil}
		ad.setConfig(def)
		log.Warn("Failed to read config from '%s': %v. Starting with defaults.", configPath, err)
		// Best-effort write to create the file for future runs
		if err := WriteConfigToFile(configPath, def); err != nil {
			log.Error("Failed to write default full config: %v", err)
		}
		logShortcuts(def.Shortcuts)
	}
	ad.publish(backendSubject)

	// Subscribe to frontend updates (full config)
	ad.nats.SubscribeToSubject(frontendSubject, func(msg *nats.Msg) {
		var incoming PieMenuConfig
		if err := json.Unmarshal(msg.Data, &incoming); err != nil {
			log.Error("Failed to unmarshal PieMenuConfig: %v", err)
			return
		}
		// Basic sanity: ensure maps are non-nil
		if incoming.Buttons == nil {
			incoming.Buttons = ConfigData{}
		}
		if incoming.Shortcuts == nil {
			incoming.Shortcuts = map[string]ShortcutEntry{}
		}
		ad.setConfig(incoming)
		if err := WriteConfigToFile(configPath, incoming); err != nil {
			log.Error("Failed to write unified config to file: %v", err)
			return
		}
		ad.publish(backendSubject)
	})

    // Removed partial shortcut update/delete handling. Only full unified config updates are persisted.

    // Backups are owned by the unified config manager
    saveBackupSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_SAVE_BACKUP")
    loadBackupSubject := os.Getenv("PUBLIC_NATSSUBJECT_PIEMENUCONFIG_LOAD_BACKUP")

    // Save backup: ignore payload and snapshot current authoritative config
    ad.nats.SubscribeToSubject(saveBackupSubject, func(msg *nats.Msg) {
        cfg := ad.getConfig()
        if err := BackupFullConfigToFile(cfg); err != nil {
            log.Error("Failed to write full backup: %v", err)
            return
        }
        log.Info("Full config backup successful.")
    })

    // Load backup from provided file path (string)
    ad.nats.SubscribeToSubject(loadBackupSubject, func(msg *nats.Msg) {
        backupPath := string(msg.Data)
        if len(backupPath) > 0 && (backupPath[0] == '"' || backupPath[0] == '\'') {
            backupPath = backupPath[1:]
        }
        if n := len(backupPath); n > 0 && (backupPath[n-1] == '"' || backupPath[n-1] == '\'') {
            backupPath = backupPath[:len(backupPath)-1]
        }
        log.Info("Loading full config from backup: '%s'", backupPath)

        loaded, err := ReadConfigFromFile(backupPath)
        if err != nil {
            log.Error("Failed to load backup file: %v", err)
            return
        }
        // Basic sanity: ensure maps are non-nil
        if loaded.Buttons == nil {
            loaded.Buttons = ConfigData{}
        }
        if loaded.Shortcuts == nil {
            loaded.Shortcuts = map[string]ShortcutEntry{}
        }
        ad.setConfig(loaded)
        if err := WriteConfigToFile(configPath, loaded); err != nil {
            log.Error("Failed to persist loaded backup to unified config file: %v", err)
            return
        }
        ad.publish(backendSubject)
        log.Info("Full config loaded from backup and published.")
    })

	return ad
}

func (a *Adapter) setConfig(c PieMenuConfig) {
	a.mu.Lock()
	a.cfg = c
	a.mu.Unlock()
}

func (a *Adapter) getConfig() PieMenuConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *Adapter) publish(subject string) {
	cfg := a.getConfig()
	a.nats.PublishMessage(subject, cfg)
}

// newDefaultButtons creates a small default buttons configuration
// Structure: 1 menu, 1 page, 8 buttons of type ShowAnyWindow
func newDefaultButtons() ConfigData {
    const (
        numMenus   = 1
        numPages   = 1
        numButtons = 8
    )
    mk := func() MenuConfig { return make(MenuConfig) }
    pk := func() PageConfig { return make(PageConfig) }

    buttons := make(ConfigData)
    for mi := 0; mi < numMenus; mi++ {
        menuID := fmt.Sprintf("%d", mi)
        menu := mk()
        for pi := 0; pi < numPages; pi++ {
            pageID := fmt.Sprintf("%d", pi)
            page := pk()
            for bi := 0; bi < numButtons; bi++ {
                btnID := fmt.Sprintf("%d", bi)
                props, _ := json.Marshal(core.ShowAnyWindowProperties{
                    ButtonTextUpper: "",
                    ButtonTextLower: "",
                    IconPath:        "",
                    WindowHandle:    -1,
                })
                page[btnID] = Button{
                    ButtonType: string(core.ButtonTypeShowAnyWindow),
                    Properties: props,
                }
            }
            menu[pageID] = page
        }
        buttons[menuID] = menu
    }
    return buttons
}
