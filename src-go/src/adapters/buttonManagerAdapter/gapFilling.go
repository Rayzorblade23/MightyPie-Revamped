package buttonManagerAdapter

import (
	"sort"
	"strconv"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

// FillWindowAssignmentGaps implements the user's simpler gap-filling approach:
// 1. Count the number of gaps (empty ShowAnyWindow/ShowProgramWindow buttons).
// 2. Clear that many highest-indexed assigned buttons.
// 3. Return the config (processWindowUpdate should be called separately).
func FillWindowAssignmentGaps(config ConfigData) (ConfigData, int) {

	typesToProcess := []core.ButtonType{
		core.ButtonTypeShowAnyWindow,
		core.ButtonTypeShowProgramWindow,
	}

	totalMoves := 0
	for _, buttonType := range typesToProcess {
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
					if btn.ButtonType == string(buttonType) {
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

		// Minimal-move gap-filling: if a move is made, the process restarts to find the next best move.
		movesThisType := 0
	mainLoop:
		for {
			// Iterate through all buttons to find a gap
			for gapIdx := 0; gapIdx < len(keys); gapIdx++ {
				gapBtn := config[keys[gapIdx].menuID][keys[gapIdx].pageID][keys[gapIdx].btnID]
				if !isButtonEmpty(&gapBtn) {
					continue // Not a gap, try next button
				}

				// Found a gap at gapIdx. Now find the highest-indexed source for it.
				for srcIdx := len(keys) - 1; srcIdx > gapIdx; srcIdx-- {
					srcBtn := config[keys[srcIdx].menuID][keys[srcIdx].pageID][keys[srcIdx].btnID]
					if isButtonEmpty(&srcBtn) {
						continue // Not a source, try next button
					}

					// Check for program match if necessary
					if buttonType == core.ButtonTypeShowProgramWindow {
						propsGap, errGap := GetButtonProperties[core.ShowProgramWindowProperties](gapBtn)
						propsSrc, errSrc := GetButtonProperties[core.ShowProgramWindowProperties](srcBtn)
						if errGap != nil || errSrc != nil {
							continue // Cannot compare, so not a match
						}
						if propsGap.ButtonTextLower != propsSrc.ButtonTextLower {
							continue // Not the same program, not a match
						}
					}

					// We found a valid gap and a source. Perform the move.
					srcKey := keys[srcIdx]
					gapKey := keys[gapIdx]
					log.Info("[GAPFILL] MOVE: %s (%s,%s,%s) -> (%s,%s,%s)", buttonType, srcKey.menuID, srcKey.pageID, srcKey.btnID, gapKey.menuID, gapKey.pageID, gapKey.btnID)

					// Get fresh button structs for the move
					srcBtnToMove := config[srcKey.menuID][srcKey.pageID][srcKey.btnID]
					gapBtnToFill := config[gapKey.menuID][gapKey.pageID][gapKey.btnID]

					switch buttonType {
					case core.ButtonTypeShowAnyWindow:
						props, _ := GetButtonProperties[core.ShowAnyWindowProperties](srcBtnToMove)
						SetButtonProperties(&gapBtnToFill, props)
					case core.ButtonTypeShowProgramWindow:
						props, _ := GetButtonProperties[core.ShowProgramWindowProperties](srcBtnToMove)
						SetButtonProperties(&gapBtnToFill, props)
					}
					clearButtonWindowProperties(&srcBtnToMove)

					config[gapKey.menuID][gapKey.pageID][gapKey.btnID] = gapBtnToFill
					config[srcKey.menuID][srcKey.pageID][srcKey.btnID] = srcBtnToMove

					movesThisType++
					continue mainLoop // Restart the search for the next best move
				}
				// If we are here, no source was found for the current gap.
				// The outer loop will proceed to the next gap candidate.
			}

			// If we complete the gap-finding loop without making a move, we're done.
			break mainLoop
		}
		totalMoves += movesThisType
	}

	return config, totalMoves
}

func isButtonEmpty(btn *Button) bool {
	switch core.ButtonType(btn.ButtonType) {
	case core.ButtonTypeShowProgramWindow:
		props, err := GetButtonProperties[core.ShowProgramWindowProperties](*btn)
		return err == nil && props.WindowHandle == InvalidHandle
	case core.ButtonTypeShowAnyWindow:
		props, err := GetButtonProperties[core.ShowAnyWindowProperties](*btn)
		return err == nil && props.WindowHandle == InvalidHandle
	}
	return true
}
