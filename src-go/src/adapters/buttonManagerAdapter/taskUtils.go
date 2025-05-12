package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
)

// Helper function to get typed properties from a task
func GetTaskProperties[T any](task Task) (T, error) {
	var props T
	if err := json.Unmarshal(task.Properties, &props); err != nil {
		return props, err
	}
	return props, nil
}

// SetTaskProperties updates the properties of a task with new values
func SetTaskProperties[T any](task *Task, props T) error {
	// Marshal the properties to JSON
	jsonData, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %v", err)
	}

	// Set the raw message
	task.Properties = json.RawMessage(jsonData)
	return nil
}