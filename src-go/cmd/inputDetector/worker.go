package main

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/inputDetectionAdapter"
)

func main() {
	keys := []string{
		"Shift",
		"Ctrl",
		"D",
	}

	// Initialize the shortcut using the values from the slice
	shortcut := inputDetectionAdapter.Shortcut{
		inputDetectionAdapter.KeyMap[keys[0]],
		inputDetectionAdapter.KeyMap[keys[1]],
		inputDetectionAdapter.KeyMap[keys[2]],
	}

    // Create and start the keyboard hook
	inputDetectionAdapter.MyInputDetector(shortcut)
}