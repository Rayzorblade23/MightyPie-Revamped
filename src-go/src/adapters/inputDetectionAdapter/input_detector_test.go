package inputDetectionAdapter_test

import (
	"testing"

	. "github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/inputDetectionAdapter"
)

func Test_Detector_givenValidKeyInput_whenCalled_thenSuccess(t *testing.T) {

	dummyAllKeyPressed := func(key int) bool {
        if KeyMap["Shift"] == key{
			return true

		}
		if KeyMap["Ctrl"] == key{
			return true

		}
		if KeyMap["A"] == key{
			return true
		}
		return false
    }

	dummyIsKeyPressedMods := func(key int) bool {
        if KeyMap["Shift"] == key{
			return true

		}
		if KeyMap["Ctrl"] == key{
			return true

		}
		if KeyMap["A"] == key{
			return false
		}
		return false
    }

    var shortcutKeys = []int{
        KeyMap["Shift"], // Modifier key "Shift"
        KeyMap["Ctrl"],  // Modifier key "Ctrl"
        KeyMap["A"],     // Single key "A"
    }

    shortcutSuccessfull := MyInputDetector(dummyAllKeyPressed, shortcutKeys)

	if shortcutSuccessfull {
		t.Errorf("Shortcut should not be pressed")
	}

	shortcutSuccessfull = MyInputDetector(dummyIsKeyPressedMods, shortcutKeys)

	if shortcutSuccessfull {
		t.Errorf("Shortcut should not be pressed")
	}

	shortcutSuccessfull = MyInputDetector(dummyAllKeyPressed, shortcutKeys)
	if !shortcutSuccessfull {
		t.Errorf("Should be pressed")
	}
}


func Test_Detector_givenValidKeyInputWithTwoKeys_whenCalled_thenSuccess(t *testing.T) {

	dummyAllKeyPressed := func(key int) bool {
        if KeyMap["Alt"] == key{
			return true

		}
		if KeyMap["A"] == key{
			return true
		}
		return false
    }

	dummyIsKeyPressedMods := func(key int) bool {
        if KeyMap["Alt"] == key{
			return true

		}
		if KeyMap["A"] == key{
			return false
		}
		return false
    }

    var shortcutKeys = []int{
        KeyMap["Alt"], // Modifier key "Shift"
        KeyMap["A"],     // Single key "A"
    }

    shortcutSuccessfull := MyInputDetector(dummyAllKeyPressed, shortcutKeys)

	if shortcutSuccessfull {
		t.Errorf("Shortcut should not be pressed")
	}

	shortcutSuccessfull = MyInputDetector(dummyIsKeyPressedMods, shortcutKeys)

	if shortcutSuccessfull {
		t.Errorf("Shortcut should not be pressed")
	}

	shortcutSuccessfull = MyInputDetector(dummyAllKeyPressed, shortcutKeys)
	if !shortcutSuccessfull {
		t.Errorf("Should be pressed")
	}
}