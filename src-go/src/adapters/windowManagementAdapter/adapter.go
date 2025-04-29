package windowManagementAdapter

import (
	"encoding/json"
	"fmt"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"
)

type WindowManagementAdapter struct {
    natsAdapter *natsAdapter.NatsAdapter
}

type shortcutPressed_Message struct {
    ShortcutPressed int `json:"shortcutPressed"`
}

func New(natsAdapter *natsAdapter.NatsAdapter) *WindowManagementAdapter {
    a := &WindowManagementAdapter{
        natsAdapter: natsAdapter,
    }

    natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_SHORTCUT_PRESSED"), func(msg *nats.Msg) {
        var message shortcutPressed_Message
        if err := json.Unmarshal(msg.Data, &message); err != nil {
            fmt.Printf("Failed to decode command: %v\n", err)
            return
        }

		fmt.Printf("WindowManagementAdapter knows a Shortcut is pressed: %d\n", message.ShortcutPressed)

    })

    return a
}

func (a *WindowManagementAdapter) Run() error {
    fmt.Println("WindowManagementAdapter started")
    select {} 
}

