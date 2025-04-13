package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/inputDetectionAdapter"
)

func main() {
	var shortcutKeys = []int{
		inputDetectionAdapter.KeyMap["Shift"], // Modifier key "Shift"
		inputDetectionAdapter.KeyMap["Ctrl"],  // Modifier key "Ctrl"
		inputDetectionAdapter.KeyMap["A"],     // Single key "A"
	}
	for {
		inputDetectionAdapter.MyInputDetector(inputDetectionAdapter.IsKeyPressed, shortcutKeys)
	}
}