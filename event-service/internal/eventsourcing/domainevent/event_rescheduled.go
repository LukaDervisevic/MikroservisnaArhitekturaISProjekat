package domainevent

import (
	"fmt"
	"time"
)

type EventRescheduled struct {
	BaseEvent
	OldDateTime int64 `json:"oldDateTime"`
	NewDateTime int64 `json:"newDateTime"`
}

func (e *EventRescheduled) EventType() string { return TypeEventRescheduled }

func (e *EventRescheduled) Describe() string {
	return fmt.Sprintf("rescheduled from %s to %s",
		time.Unix(e.OldDateTime, 0).UTC().Format(time.RFC3339),
		time.Unix(e.NewDateTime, 0).UTC().Format(time.RFC3339))
}
