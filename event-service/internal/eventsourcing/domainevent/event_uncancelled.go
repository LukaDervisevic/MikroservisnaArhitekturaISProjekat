package domainevent

import "fmt"

type EventUncancelled struct {
	BaseEvent
	Reason string `json:"reason"`
}

func (e *EventUncancelled) EventType() string { return TypeEventUncancelled }

func (e *EventUncancelled) Describe() string {
	return fmt.Sprintf("un-cancelled (reason: %s)", e.Reason)
}
