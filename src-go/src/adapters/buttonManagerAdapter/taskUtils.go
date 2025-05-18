package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
)

// Helper function to get typed properties from a button
func GetButtonProperties[T any](button Button) (T, error) {
	var props T
	if err := json.Unmarshal(button.Properties, &props); err != nil {
		return props, err
	}
	return props, nil
}

// SetButtonProperties updates the properties of a button with new values
func SetButtonProperties[T any](button *Button, props T) error {
	// Marshal the properties to JSON
	jsonData, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %v", err)
	}

	// Set the raw message
	button.Properties = json.RawMessage(jsonData)
	return nil
}
