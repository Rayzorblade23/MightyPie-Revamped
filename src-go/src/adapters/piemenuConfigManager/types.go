package piemenuConfigManager

import "encoding/json"

// Mirror the frontend structure for the unified config file

type Button struct {
    ButtonType string          `json:"button_type"`
    Properties json.RawMessage `json:"properties"`
}

type PageConfig map[string]Button

type MenuConfig map[string]PageConfig

// Buttons nested record structure
// Top-level is MenuID -> PageID -> ButtonID
// For compatibility with existing files
// Note: keep keys as strings

type ConfigData map[string]MenuConfig

type ShortcutEntry struct {
    Codes []int  `json:"codes"`
    Label string `json:"label"`
}

type StarredFavorite struct {
    MenuID int `json:"menuID"`
    PageID int `json:"pageID"`
}

type PieMenuConfig struct {
    Buttons   ConfigData                `json:"buttons"`
    Shortcuts map[string]ShortcutEntry `json:"shortcuts"`
    Starred   *StarredFavorite          `json:"starred"`
}
