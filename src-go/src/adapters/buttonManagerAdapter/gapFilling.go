package buttonManagerAdapter

import (
	"log"
	"sort"
	"strconv"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// FillWindowAssignmentGaps implements the user's simpler gap-filling approach:
// 1. Count the number of gaps (empty ShowAnyWindow/ShowProgramWindow buttons).
// 2. Clear that many highest-indexed assigned buttons.
// 3. Return the config (processWindowUpdate should be called separately).
func FillWindowAssignmentGaps(config ConfigData) (ConfigData, int) {

	types := []struct {
		TypeStr string
		Type    ButtonType
	}{
		{"show_any_window", ButtonTypeShowAnyWindow},
		{"show_program_window", ButtonTypeShowProgramWindow},
	}

	totalMoves := 0
	// Debug: dump Menu 0, Pages 0 and 1 BEFORE ALL GAP-FILL
	dumpFirstMenuPages(config)
	for _, t := range types {
		// Collect all keys for buttons of this type, in config order (across ALL menus/pages)
		type btnKey struct{ menuID, pageID, btnID string }
		var keys []btnKey
		for menuID, menuConfig := range config {
			for pageID, pageConfig := range menuConfig {
				// Sort btnIDs numerically for stable order
				var btnIDs []string
				for btnID := range pageConfig {
					btnIDs = append(btnIDs, btnID)
				}
				sort.Slice(btnIDs, func(i, j int) bool {
					iIdx, _ := strconv.Atoi(btnIDs[i])
					jIdx, _ := strconv.Atoi(btnIDs[j])
					return iIdx < jIdx
				})
				for _, btnID := range btnIDs {
					btn := pageConfig[btnID]
					if btn.ButtonType == string(t.Type) {
						keys = append(keys, btnKey{menuID, pageID, btnID})
					}
				}
			}
		}

		sort.Slice(keys, func(i, j int) bool {
			mi, _ := strconv.Atoi(keys[i].menuID)
			mj, _ := strconv.Atoi(keys[j].menuID)
			if mi != mj {
				return mi < mj
			}
			pi, _ := strconv.Atoi(keys[i].pageID)
			pj, _ := strconv.Atoi(keys[j].pageID)
			if pi != pj {
				return pi < pj
			}
			bi, _ := strconv.Atoi(keys[i].btnID)
			bj, _ := strconv.Atoi(keys[j].btnID)
			return bi < bj
		})

		// Minimal-move gap-filling: for each gap, move the highest-indexed assigned window above it into the gap
		moves := 0
		for {
			// Find the lowest-indexed gap
			gapIdx := -1
			for i := 0; i < len(keys); i++ {
				gapBtn := config[keys[i].menuID][keys[i].pageID][keys[i].btnID]
				if isButtonEmpty(&gapBtn) {
					gapIdx = i
					break
				}
			}
			if gapIdx == -1 {
				break // no more gaps
			}
			// Find the highest-indexed assigned window above the gap
			srcIdx := -1
			for i := len(keys) - 1; i > gapIdx; i-- {
				srcBtn := config[keys[i].menuID][keys[i].pageID][keys[i].btnID]
				if !isButtonEmpty(&srcBtn) {
					if t.Type == ButtonTypeShowProgramWindow {
						gapBtn := config[keys[gapIdx].menuID][keys[gapIdx].pageID][keys[gapIdx].btnID]
						propsGap, errGap := GetButtonProperties[core.ShowProgramWindowProperties](gapBtn)
						propsSrc, errSrc := GetButtonProperties[core.ShowProgramWindowProperties](srcBtn)
						if errGap != nil || errSrc != nil {
							continue // skip if cannot get properties
						}
						if propsGap.ButtonTextLower != propsSrc.ButtonTextLower {
							continue // skip if not the same program
						}
					}
					srcIdx = i
					break
				}
			}
			if srcIdx == -1 {
				break // no more assigned windows above gaps (or no same-program match)
			}
			// Move assignment from srcIdx to gapIdx
			srcKey := keys[srcIdx]
			gapKey := keys[gapIdx]
			log.Printf("[GAPFILL] MOVE: %s (%s,%s,%s) -> (%s,%s,%s)", t.TypeStr, srcKey.menuID, srcKey.pageID, srcKey.btnID, gapKey.menuID, gapKey.pageID, gapKey.btnID)
			srcBtn := config[srcKey.menuID][srcKey.pageID][srcKey.btnID]
			gapBtn := config[gapKey.menuID][gapKey.pageID][gapKey.btnID]
			switch t.Type {
			case ButtonTypeShowAnyWindow:
				props, _ := GetButtonProperties[core.ShowAnyWindowProperties](srcBtn)
				SetButtonProperties(&gapBtn, props)
				clearButtonWindowProperties(&srcBtn)
				config[gapKey.menuID][gapKey.pageID][gapKey.btnID] = gapBtn
				config[srcKey.menuID][srcKey.pageID][srcKey.btnID] = srcBtn
			case ButtonTypeShowProgramWindow:
				propsGap, errGap := GetButtonProperties[core.ShowProgramWindowProperties](gapBtn)
				propsSrc, errSrc := GetButtonProperties[core.ShowProgramWindowProperties](srcBtn)
				if errGap != nil || errSrc != nil {
					break // skip if cannot get properties
				}
				if propsGap.ButtonTextLower != propsSrc.ButtonTextLower {
					break // do not move across programs
				}
				SetButtonProperties(&gapBtn, propsSrc)
				clearButtonWindowProperties(&srcBtn)
				config[gapKey.menuID][gapKey.pageID][gapKey.btnID] = gapBtn
				config[srcKey.menuID][srcKey.pageID][srcKey.btnID] = srcBtn
			}
			moves++
		}
		totalMoves += moves
	}
	// Debug: dump Menu 0, Pages 0 and 1 AFTER ALL GAP-FILL
	dumpFirstMenuPages(config)
	return config, totalMoves
}

// Helper to format handle for debug
func handleToStr(h int) string {
	if h == -1 {
		return "-"
	}
	return strconv.Itoa(h)
}

// dumpFirstMenuPages prints the first two pages of the first menu, showing button IDs, type, and title/handle.
func dumpFirstMenuPages(config ConfigData) {
	menuID := "0"
	for _, pageID := range []string{"0", "1"} {
		log.Printf("[GAPFILL] DUMP Menu 0 Page %s:", pageID)
		pageConfig, ok := config[menuID][pageID]
		if !ok {
			continue
		}
		for btnIdx := 0; btnIdx < 8; btnIdx++ {
			btnID := strconv.Itoa(btnIdx)
			btn, ok := pageConfig[btnID]
			if !ok {
				continue
			}
			var title string
			switch ButtonType(btn.ButtonType) {
			case ButtonTypeShowAnyWindow:
				props, err := GetButtonProperties[core.ShowAnyWindowProperties](btn)
				if err == nil {
					title = props.ButtonTextUpper
				}
			case ButtonTypeShowProgramWindow:
				props, err := GetButtonProperties[core.ShowProgramWindowProperties](btn)
				if err == nil {
					title = "handle=" + handleToStr(props.WindowHandle)
				}
			}
			log.Printf("[GAPFILL] BTN: menu=0 page=%s btn=%s type=%s title='%s'", pageID, btnID, btn.ButtonType, title)
		}
	}
}

func isButtonEmpty(btn *Button) bool {
	switch ButtonType(btn.ButtonType) {
	case ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*btn)
		return err == nil && props.WindowHandle == InvalidHandle
	case ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*btn)
		return err == nil && props.WindowHandle == InvalidHandle
	}
	return true
}
