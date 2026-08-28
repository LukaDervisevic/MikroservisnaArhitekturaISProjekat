package domainevent

import "fmt"

type EventCancelled struct {
	BaseEvent
	Reason string `json:"reason"`
}

func (e *EventCancelled) EventType() string { return TypeEventCancelled }

func (e *EventCancelled) Describe() string {
	return fmt.Sprintf("cancelled (reason: %s)", e.Reason)
}
