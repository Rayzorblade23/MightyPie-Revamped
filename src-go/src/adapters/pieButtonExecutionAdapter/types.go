package pieButtonExecutionAdapter

import (
	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
)

const (
	ClickTypeLeftUp   = "left_up"
	ClickTypeRightUp  = "right_up"
	ClickTypeMiddleUp = "middle_up"
)

// Message type for pie button execution
type pieButtonExecute_Message struct {
	PageIndex   int             `json:"page_index"`
	ButtonIndex int             `json:"button_index"`
	ButtonType  core.ButtonType `json:"button_type"`
	Properties  any             `json:"properties"`
	ClickType   string          `json:"click_type"`
}
