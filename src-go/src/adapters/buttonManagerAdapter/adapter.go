package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
)

var (
	buttonConfig ConfigData
	windowsList  WindowsUpdate_Message
	mu           sync.RWMutex
)

type ButtonManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ButtonManagerAdapter {
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	config, err := ReadButtonConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Store config at package level
	buttonConfig = config
	PrintConfig(config)

	a.natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_WINDOWMANAGER_UPDATE"), func(msg *nats.Msg) {
		var message WindowsUpdate_Message

		if err := json.Unmarshal(msg.Data, &message); err != nil {
			println("Failed to decode message: %v", err)
			return
		}

		// Update package level windows list with mutex protection
		mu.Lock()
		windowsList = message
		mu.Unlock()

		PrintWindowList(message)
	})

	return a
}

// GetCurrentWindowsList returns a copy of the current windows list
func GetCurrentWindowsList() WindowsUpdate_Message {
	mu.RLock()
	defer mu.RUnlock()
	return windowsList
}

// GetButtonConfig returns the current button configuration
func GetButtonConfig() ConfigData {
	return buttonConfig
}

func (a *ButtonManagerAdapter) Run() error {
	fmt.Println("ButtonManagerAdapter started")
	select {}
}

func ReadButtonConfig() (ConfigData, error) {
	// Get user's AppData Local path
	localAppData := os.Getenv("LOCALAPPDATA")
	configPath := filepath.Join(localAppData, "MightyPieRevamped", "buttonConfig.json")

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON
	var config ConfigData
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

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
