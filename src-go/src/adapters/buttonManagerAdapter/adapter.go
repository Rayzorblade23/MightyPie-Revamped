package buttonManagerAdapter

import (
	"encoding/json"
	"fmt"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/adapters/natsAdapter"
	"github.com/nats-io/nats.go"

	env "github.com/Rayzorblade23/MightyPie-Revamped/cmd"
)

type ButtonManagerAdapter struct {
	natsAdapter *natsAdapter.NatsAdapter
}

func New(natsAdapter *natsAdapter.NatsAdapter) *ButtonManagerAdapter {
	a := &ButtonManagerAdapter{
		natsAdapter: natsAdapter,
	}

	a.natsAdapter.SubscribeToSubject(env.Get("NATSSUBJECT_WINDOWMANAGER_UPDATE"), func(msg *nats.Msg) {
		var message WindowsUpdate_Message

		if err := json.Unmarshal(msg.Data, &message); err != nil {
			println("Failed to decode message: %v", err)
			return
		}

		PrintWindowList(message)
	})

	return a
}

func (a *ButtonManagerAdapter) Run() error {
	fmt.Println("ButtonManagerAdapter started")
	select {}
}

// // PrintWindowList prints the current window list for debugging
func PrintWindowList(mapping map[int]WindowInfo_Message) {
    fmt.Println("------------------ Current Window List ------------------")
    if len(mapping) == 0 {
        fmt.Println("(empty)")
        return
    }
    for hwnd, info := range mapping {
        fmt.Printf("Window Handle: %d\n", hwnd)
        fmt.Printf("  Title: %s\n", info.Title)
        fmt.Printf("  ExeName: %s\n", info.ExeName)
        fmt.Printf("  ExePath: %s\n", info.ExePath)
        fmt.Printf("  AppName: %s\n", info.AppName)
        fmt.Printf("  Instance: %d\n", info.Instance)
        fmt.Printf("  IconPath: %s\n", info.IconPath)
        fmt.Println()
    }
    fmt.Println("---------------------------------------------------------")
}
